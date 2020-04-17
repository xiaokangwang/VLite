package transport

import (
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
	Connection(conn net.Conn)
}

type UnderlayTransportDialer interface {
	Connect()(net.Conn, error)
}