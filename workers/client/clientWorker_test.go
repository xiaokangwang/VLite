package client

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"testing"
)

func TestInitializeNil(t *testing.T) {
	client := UDPClient(context.Background(),
		make(chan UDPClientTxToServerTraffic),
		make(chan UDPClientTxToServerDataTraffic),
		make(chan UDPClientRxFromServerTraffic),
		make(chan interfaces.UDPPacket),
		make(chan interfaces.UDPPacket))
	_ = client
}
