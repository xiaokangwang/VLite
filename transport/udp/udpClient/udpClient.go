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

	masking := ""

	if v := ctx.Value(interfaces.ExtraOptionsUDPMask); v != nil {
		masking = v.(string)
	}

	return &udpClient{dest: addr, ctx: ctx, masking: masking}
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
	if v := ctx.Value(interfaces.ExtraOptionsUDPShouldMask); v != nil && v.(bool) == true {
		usageConn = masker2conn.NewMaskerAdopter(prependandxor.GetPrependAndPolyXorMask(string(u.masking), []byte{}), conn)
	}

	id := []byte(conn.LocalAddr().String())
	connctx := context.WithValue(u.ctx, interfaces.ExtraOptionsConnID, id)

	connctx = context.WithValue(connctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())

	return usageConn, nil, connctx
}
