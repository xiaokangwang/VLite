package transport

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"net"
)

type Transport interface {
	Register(TxChannel chan interfaces.TrafficWithChannelTag,
		TxDataChannel chan interfaces.TrafficWithChannelTag,
		RxChannel chan interfaces.TrafficWithChannelTag)
	Up()
}

type UnderlayTransportListener interface {
	Connection(conn net.Conn, ctx context.Context) context.Context
}

type UnderlayTransportDialer interface {
	Connect(ctx context.Context) (net.Conn, error, context.Context)
}
