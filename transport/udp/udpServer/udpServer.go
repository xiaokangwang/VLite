package udpServer

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/transport"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func NewUDPServer(address string, ctx context.Context, listener transport.UnderlayTransportListener) *udpServer {
	var err error

	us := &udpServer{
		conn:  nil,
		ctx:   ctx,
		under: listener,
	}

	us.conn, err = net.ListenPacket("udp", address)
	if err != nil {
		println(err.Error())
	}

	go us.Listener()
	return us
}

type udpServer struct {
	conn  net.PacketConn
	ctx   context.Context
	under transport.UnderlayTransportListener

	remoteConnTracker sync.Map

	masking string
}

func (u *udpServer) Listener() {
	for {
		bm := make([]byte, 2000)
		c, a, e := u.conn.ReadFrom(bm)
		if e != nil {
			log.Println(e)
		}

		conn := &connImpl{
			server:     u,
			remoteAddr: a,
			readchan:   make(chan []byte, 8),
		}

		connx, ok := u.remoteConnTracker.LoadOrStore(a.String(), conn)
		if ok {
			conn = connx.(*connImpl)
		} else {
			usageConn := conn
			//usageConn := masker2conn.NewMaskerAdopter(prependandxor.GetPrependAndXorMask(string(u.masking), []byte{0x1f, 0x0d}), conn)
			connid := []byte(conn.remoteAddr.String())
			connctx := context.WithValue(u.ctx, interfaces.ExtraOptionsConnID, connid)
			connctx = context.WithValue(u.ctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())
			go u.under.Connection(usageConn, connctx)
		}

		conn.readchan <- bm[:c]

	}
}

type connImpl struct {
	server     *udpServer
	remoteAddr net.Addr
	readchan   chan []byte
}

func (c connImpl) Read(b []byte) (n int, err error) {
	select {
	case by := <-c.readchan:
		copy(b, by)
		return len(by), nil
	case <-time.Tick(time.Second * 400):
		return 0, io.ErrClosedPipe
	}

}

func (c connImpl) Write(b []byte) (n int, err error) {
	return c.server.conn.WriteTo(b, c.remoteAddr)
}

func (c connImpl) Close() error {
	c.server.remoteConnTracker.Delete(c.remoteAddr.String())
	return nil
}

func (c connImpl) LocalAddr() net.Addr {
	panic("implement me")
}

func (c connImpl) RemoteAddr() net.Addr {
	panic("implement me")
}

func (c connImpl) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (c connImpl) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (c connImpl) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}
