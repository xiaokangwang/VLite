package udpClient

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/masker2conn"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/presets/prependandxor"
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

func (u *udpClient) Connect(ctx context.Context) (net.Conn, error, context.Context) {
	conn, err := net.Dial("udp", u.dest)
	if err != nil {
		return nil, err, nil
	}
	usageConn := conn
	if v := ctx.Value(interfaces.ExtraOptionsUDPShouldMask); v.(bool) == true {
		usageConn = masker2conn.NewMaskerAdopter(prependandxor.GetPrependAndPolyXorMask(string(u.masking), []byte{0x1f, 0x0d}), conn)
	}

	id := []byte(conn.LocalAddr().String())
	connctx := context.WithValue(u.ctx, interfaces.ExtraOptionsConnID, id)

	connctx = context.WithValue(connctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())

	return usageConn, nil, connctx
}
