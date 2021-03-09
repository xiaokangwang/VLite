package client

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"testing"
)

type Stub struct {
}

func (s Stub) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	panic("implement me")
}

func TestInitializeNil(t *testing.T) {
	client := UDPClient(context.Background(),
		make(chan UDPClientTxToServerTraffic),
		make(chan UDPClientTxToServerDataTraffic),
		make(chan UDPClientRxFromServerTraffic),
		make(chan interfaces.UDPPacket),
		make(chan interfaces.UDPPacket), &Stub{})
	_ = client
}
