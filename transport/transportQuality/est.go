package transportQuality

import (
	"fmt"
	"github.com/xiaokangwang/VLite/proto"
	"time"
)

func NewQualityEstimator() *QualityEstimator {
	qe := &QualityEstimator{}
	return qe
}

type QualityEstimator struct {
	lastPing proto.PingHeader
}

func (qe *QualityEstimator) OnSendPing(ping proto.PingHeader) {
	qe.lastPing = ping
}
func (qe *QualityEstimator) OnReceivePong(pong proto.PongHeader) {
	if pong.SeqCopy != qe.lastPing.Seq {
		fmt.Println("Out of Order pong ignored")
		return
	}

	timePassed := uint64(time.Now().UnixNano()) - qe.lastPing.Seq2

	RTT := float64(timePassed) / float64(1e6)

	PacketLossingTx := float64(float64(qe.lastPing.SentPacket)-float64(pong.RecvPacket)) / float64(qe.lastPing.SentPacket)

	PacketLossingRx := float64(float64(pong.SentPacket)-float64(qe.lastPing.RecvPacket)) / float64(pong.SentPacket)

	fmt.Printf("Pong Seq: %v, RTT %.5f ms, TxLoss %.5f%%, RxLoss %.5f%%\n", pong.SeqCopy, RTT, PacketLossingTx*100, PacketLossingRx*100)
}
