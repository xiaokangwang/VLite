package assembly

import (
	"bytes"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"math"
	"sync/atomic"
	"time"
)

type packetAssemblyTxChunkHolder struct {
	ef              interfaces.ErrorCorrectionFacility
	enabled         bool
	DataShardWithin int
	Seq             uint32

	InitialSendEpochSeq uint64
	InitialRemainShard  int64
}

func (pa *PacketAssembly) Tx() {

	packetSentThisEpoch := 0
	CurrentTxBufferSlot := 0

	pochTimer := time.NewTimer(time.Duration(pa.TxEpochTimeInMs) * time.Millisecond)

	pa.TxRingBuffer[CurrentTxBufferSlot] = packetAssemblyTxChunkHolder{
		ef:              pa.ecff.Create(pa.ctx),
		enabled:         false,
		DataShardWithin: 0,
	}

	for {
		select {
		case <-pochTimer.C:

			if pa.TxRingBuffer[CurrentTxBufferSlot].DataShardWithin != 0 {
				var donez bool
				donez, CurrentTxBufferSlot, packetSentThisEpoch =
					pa.finishThisSeq(CurrentTxBufferSlot, packetSentThisEpoch)
				if donez {
					return
				}
			}

			sendingQuota := pa.TxFECSoftPacketSoftLimitPerEpoch - packetSentThisEpoch

			if sendingQuota > 0 {
				sendingGuidance := pa.SelectPacketToSend(sendingQuota)
				more := true
				for more {
					more = false
					for i := 0; i < pa.TxRingBufferSize; i++ {
						if sendingGuidance[i] > 0 {
							sendingGuidance[i]--
							pa.sendFECPacket(i)
							more = true
						}
					}
				}

			}

			pa.TxEpocSeq++
			pochTimer.Reset(time.Duration(pa.TxEpochTimeInMs) * time.Millisecond)
			packetSentThisEpoch = 0
		case packet := <-pa.TxChan:
			fecenabled := atomic.LoadUint32(&pa.FECEnabled)
			if fecenabled == 0 {
				pa.TxWithoutFEC(packet)
				continue
			}
			id, wd := pa.TxRingBuffer[CurrentTxBufferSlot].ef.AddData(packet)
			pa.TxRingBuffer[CurrentTxBufferSlot].DataShardWithin++
			pa.TxRingBuffer[CurrentTxBufferSlot].Seq = pa.TxNextSeq
			pa.TxRingBuffer[CurrentTxBufferSlot].InitialSendEpochSeq = pa.TxEpocSeq
			pa.TxRingBuffer[CurrentTxBufferSlot].InitialRemainShard = 0
			seq := pa.TxNextSeq
			if wd != nil {
				if pa.packAndSend(seq, id, wd) {
					return
				}
				pa.TxShardOriginal += 1
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

//Return number of packet to send for each slot
func (pa *PacketAssembly) SelectPacketToSend(quota int) []int {
	//First we will detect how many shards are released

	unlocks := make([]int, pa.TxRingBufferSize)
	totalUnlocks := 0
	//First Pass How many shreds are allowed to send
	for i := range unlocks {
		releasedShard := pa.GetReleasedShard(i)
		SentShard := pa.GetSentShard(i)
		diff := releasedShard - SentShard
		remainShard := pa.GetRemainingShard(i)
		if diff > remainShard {
			diff = remainShard
		}
		totalUnlocks += diff
		unlocks[i] = diff
	}

	if totalUnlocks <= quota {
		return unlocks
	}

	//Or we will scale them based on traffic quota

	scale := float64(totalUnlocks / quota)
	for i := range unlocks {
		unlocks[i] = int(math.Round(float64(unlocks[i]) / scale))
	}
	return unlocks

}
func (pa *PacketAssembly) GetRemainingShard(slot int) int {
	if !pa.TxRingBuffer[slot].enabled {
		return 0
	}
	return pa.TxRingBuffer[slot].ef.MaxShardYieldRemaining()
}
func (pa *PacketAssembly) GetReleasedShard(slot int) int {
	if !pa.TxRingBuffer[slot].enabled {
		return 0
	}
	process := pa.GetProcess(slot)
	shards := pa.GetTotalShard(slot)
	return int(math.Round(process * float64(shards)))
}
func (pa *PacketAssembly) GetProcess(slot int) float64 {
	if !pa.TxRingBuffer[slot].enabled {
		return 1
	}
	EpocSpend := pa.TxEpocSeq - pa.TxRingBuffer[slot].InitialSendEpochSeq
	EpocAll := pa.TxRingBufferSize
	return float64(EpocSpend) / float64(EpocAll)
}
func (pa *PacketAssembly) GetSentShard(slot int) int {
	if !pa.TxRingBuffer[slot].enabled {
		return 0
	}
	Sent := int(pa.TxRingBuffer[slot].InitialRemainShard) - pa.GetRemainingShard(slot)
	return Sent
}

func (pa *PacketAssembly) GetTotalShard(slot int) int {
	if !pa.TxRingBuffer[slot].enabled {
		return 0
	}
	sum := pa.TxRingBuffer[slot].InitialRemainShard
	return int(sum)
}

func (pa *PacketAssembly) TrafficShapingFunc(process float64) float64 {
	if process <= 0.1 {
		return 0
	}

	if process <= 0.8 {
		return (process - 0.1) * 1.4
	}

	return 1
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
	pa.TxRingBuffer[CurrentTxBufferSlot].InitialRemainShard = int64(pa.TxRingBuffer[CurrentTxBufferSlot].
		ef.MaxShardYieldRemaining())

	pa.TxShardFEC++

	CurrentTxBufferSlot++

	if CurrentTxBufferSlot == pa.TxRingBufferSize {
		CurrentTxBufferSlot = 0
	}
	pa.TxRingBuffer[CurrentTxBufferSlot] = packetAssemblyTxChunkHolder{
		ef:              pa.ecff.Create(pa.ctx),
		enabled:         false,
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

	pa.TxShardFEC += 1
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
	pa.TxBytes += uint64(len(res))
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
	pa.TxBytes += uint64(len(res))
	pa.TxShardOriginalNoFEC += 1
	return false
}
