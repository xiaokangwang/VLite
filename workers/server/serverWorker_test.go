package server

import (
	"context"
	"testing"
)

type stub struct {
}

func (s stub) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	panic("implement me")
}

func TestInitializeNil(t *testing.T) {
	server := UDPServer(context.Background(),
		make(chan UDPServerTxToClientTraffic),
		make(chan UDPServerTxToClientDataTraffic),
		make(chan UDPServerRxFromClientTraffic), &stub{})
	_ = server
}
