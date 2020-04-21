package assembly

import (
	"bytes"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"sync/atomic"
	"time"
)

type packetAssemblyTxChunkHolder struct {
	ef              interfaces.ErrorCorrectionFacility
	enabled         bool
	DataShardWithin int
	Seq             uint32

	LastPollEpochSeq              int
	MaxSendRemainingInLastPollSeq int
}

func (pa *PacketAssembly) Tx() {
	EpochSeq := 0
	packetSentThisEpoch := 0
	CurrentTxBufferSlot := 0

	pochTimer := time.NewTimer(time.Duration(pa.TxEpochTimeInMs) * time.Millisecond)

	pa.TxRingBuffer[CurrentTxBufferSlot] = packetAssemblyTxChunkHolder{
		ef:              pa.ecff.Create(pa.ctx),
		enabled:         true,
		DataShardWithin: 0,
	}

	for {
		select {
		case <-pochTimer.C:
			EpochSeq++
			var donez bool
			donez, CurrentTxBufferSlot, packetSentThisEpoch =
				pa.finishThisSeq(CurrentTxBufferSlot, packetSentThisEpoch)
			if donez {
				return
			}
			for pa.TxFECSoftPacketSoftLimitPerEpoch >= packetSentThisEpoch {
				more := false
				for thisslot, v := range pa.TxRingBuffer {
					if v.enabled && (pa.TxRingBuffer[thisslot].LastPollEpochSeq != EpochSeq ||
						pa.TxRingBuffer[thisslot].MaxSendRemainingInLastPollSeq > 0) {
						if pa.sendFECPacket(thisslot) {
							return
						}
						if pa.TxRingBuffer[thisslot].LastPollEpochSeq != EpochSeq {
							pa.TxRingBuffer[thisslot].LastPollEpochSeq = EpochSeq
							pa.TxRingBuffer[thisslot].MaxSendRemainingInLastPollSeq =
								pa.TxRingBuffer[thisslot].ef.MaxShardYieldRemaining() / 3
						} else {
							pa.TxRingBuffer[thisslot].MaxSendRemainingInLastPollSeq--
						}

						packetSentThisEpoch++
						more = true
					}
				}
				if !more {
					break
				}
			}
			pochTimer.Reset(time.Duration(pa.TxEpochTimeInMs) * time.Millisecond)
		case packet := <-pa.TxChan:
			fecenabled := atomic.LoadUint32(&pa.FECEnabled)
			if fecenabled == 0 {
				pa.TxWithoutFEC(packet)
			}
			id, wd := pa.TxRingBuffer[CurrentTxBufferSlot].ef.AddData(packet)
			pa.TxRingBuffer[CurrentTxBufferSlot].DataShardWithin++
			pa.TxRingBuffer[CurrentTxBufferSlot].Seq = pa.TxNextSeq
			seq := pa.TxNextSeq
			if wd != nil {
				if pa.packAndSend(seq, id, wd) {
					return
				}
				packetSentThisEpoch++
			}
			if pa.TxRingBuffer[CurrentTxBufferSlot].DataShardWithin >=
				pa.MaxDataShardPerChunk {
				var donez bool
				donez, CurrentTxBufferSlot, packetSentThisEpoch =
					pa.finishThisSeq(CurrentTxBufferSlot, packetSentThisEpoch)
				if donez {
					return
				}
			}
		case packet := <-pa.TxNoFECChan:
			pa.TxWithoutFEC(packet)
		case <-pa.ctx.Done():
			return
		}
	}

}

func (pa *PacketAssembly) finishThisSeq(CurrentTxBufferSlot int, packetSentThisEpoch int) (bool, int, int) {
	//finish this seq
	thisid, pack, more := pa.TxRingBuffer[CurrentTxBufferSlot].
		ef.ConstructReconstructShard()
	if pa.packAndSend(pa.TxNextSeq, thisid, pack) {
		return true, CurrentTxBufferSlot, packetSentThisEpoch
	}
	packetSentThisEpoch++
	pa.TxRingBuffer[CurrentTxBufferSlot].enabled = more

	SendingSlot := CurrentTxBufferSlot
	SendAmount := pa.TxRingBuffer[CurrentTxBufferSlot].
		ef.MaxShardYieldRemaining() / 3
	//Initial Burst Send
	SentAmount := 0
	for pa.TxRingBuffer[SendingSlot].enabled && SendAmount > SentAmount {
		if pa.sendFECPacket(SendingSlot) {
			return true, CurrentTxBufferSlot, packetSentThisEpoch
		}
		packetSentThisEpoch++
		SentAmount++
	}

	CurrentTxBufferSlot++

	if CurrentTxBufferSlot == pa.TxRingBufferSize {
		CurrentTxBufferSlot = 0
	}
	pa.TxRingBuffer[CurrentTxBufferSlot] = packetAssemblyTxChunkHolder{
		ef:              pa.ecff.Create(pa.ctx),
		enabled:         true,
		DataShardWithin: 0,
	}
	pa.TxNextSeq++
	if pa.TxNextSeq == 0 {
		pa.TxNextSeq = 1
	}
	return false, CurrentTxBufferSlot, packetSentThisEpoch
}

func (pa *PacketAssembly) sendFECPacket(SendingSlot int) bool {
	thisid2, pack2, more2 := pa.TxRingBuffer[SendingSlot].
		ef.ConstructReconstructShard()
	if pa.packAndSend(pa.TxRingBuffer[SendingSlot].Seq, thisid2, pack2) {
		return true
	}
	pa.TxRingBuffer[SendingSlot].enabled = more2
	return false
}

func (pa *PacketAssembly) packAndSend(seq uint32, id int, wd []byte) bool {
	err, res := pa.packPacket(seq, id, wd)
	if err != nil {
		panic(err)
	}
	_, err = pa.conn.Write(res)
	if err != nil {
		fmt.Println(err.Error())
		return true
	}
	return false
}

func (pa *PacketAssembly) packPacket(seq uint32, id int, wd []byte) (error, []byte) {
	bbuf := bytes.NewBuffer(nil)
	pw := &PacketWireHead{
		Seq: seq,
		Id:  id,
	}
	err := struc.Pack(bbuf, pw)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(bbuf, bytes.NewReader(wd))
	if err != nil {
		panic(err)
	}

	res := bbuf.Bytes()
	return err, res
}
func (pa *PacketAssembly) TxWithoutFEC(pack []byte) bool {
	pk, res := pa.packPacket(0, 0, pack)
	if pk != nil {
		panic(pk)
	}
	_, err := pa.conn.Write(res)
	if err != nil {
		fmt.Println(err.Error())
		return true
	}
	return false
}
