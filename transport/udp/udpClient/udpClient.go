package udpClient

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"net"
)

func NewUdpClient(addr string, ctx context.Context) *udpClient {
	return &udpClient{dest: addr, ctx: ctx}
}

type udpClient struct {
	masking string
	dest    string
	ctx     context.Context
}

func (u *udpClient) Connect() (net.Conn, error, context.Context) {
	conn, err := net.Dial("udp", u.dest)
	if err != nil {
		return nil, err, nil
	}
	usageConn := conn
	//usageConn := masker2conn.NewMaskerAdopter(prependandxor.GetPrependAndXorMask(string(u.masking), []byte{0x1f, 0x0d}), conn)

	id := []byte(conn.LocalAddr().String())
	connctx := context.WithValue(u.ctx, interfaces.ExtraOptionsConnID, id)

	connctx = context.WithValue(u.ctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())

	return usageConn, nil, connctx
}
