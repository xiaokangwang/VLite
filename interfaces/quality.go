package interfaces

import "github.com/xiaokangwang/VLite/proto"

type QualityEstimator interface {
	OnSendPing(ping proto.PingHeader)
	OnReceivePong(pong proto.PongHeader)
}
