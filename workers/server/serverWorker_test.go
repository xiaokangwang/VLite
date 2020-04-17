package server

import (
	"context"
	"testing"
)

func TestInitializeNil(t *testing.T) {
	server := UDPServer(context.Background(),
		make(chan UDPServerTxToClientTraffic),
		make(chan UDPServerTxToClientDataTraffic),
		make(chan UDPServerRxFromClientTraffic))
	_ = server
}
