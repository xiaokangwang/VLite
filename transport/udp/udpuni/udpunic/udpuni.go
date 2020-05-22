package udpunic

import (
	"context"
	"crypto/rand"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"io"
	mrand "math/rand"
	"net"
	"reflect"
	"time"
)

func NewUdpUniClient(password string,
	ctx context.Context, under transport.UnderlayTransportDialer) *UdpUniClient {
	return &UdpUniClient{
		password: password,
		ctx:      ctx,
		hh:       headerHolder.NewHttpHeaderHolderProcessor2(password, "UdpUniSecret"),
		under:    under,
	}
}

type UdpUniClient struct {
	password string
	ctx      context.Context

	hh *headerHolder.HttpHeaderHolderProcessor

	under transport.UnderlayTransportDialer
}

func (uuc *UdpUniClient) Connect(ctx context.Context) (net.Conn, error, context.Context) {
	conn, err, connctx := uuc.under.Connect(ctx)
	if err != nil {
		return nil, err, nil
	}
	uc := &udpUniClientProxy{
		ctx:     connctx,
		initBuf: nil,
		conn:    conn,
	}

	ph := proto.HttpHeaderHolder{}

	id := make([]byte, 24)
	io.ReadFull(rand.Reader, id)

	unival := ctx.Value(interfaces.ExtraOptionsUniConnAttrib)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		copy(id, uniAtt.ID)
	}

	copy(ph.ConnID[:], id)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		ph.ConnIter = uniAtt.Iter
	}

	mrand.Read(ph.Rand[:])
	ph.Time = time.Now().Unix()

	w := uuc.hh.Seal(ph)

	err = uc.UniHandShake(w)
	if err != nil {
		return nil, err, nil
	}
	return uc, nil, uc.ctx
}

func (uucp *udpUniClientProxy) UniHandShake(token string) error {
	var err error
	var n int
	for i := 0; i < 300; i++ {
		uucp.conn.SetReadDeadline(time.Now().Add(time.Second / 4))
		_, err = uucp.conn.Write([]byte(token))
		if err != nil {
			return err
		}
		var buf [1600]byte
		n, err = uucp.conn.Read(buf[:])

		if err == nil {
			if !reflect.DeepEqual(buf[:n], []byte(token)) {
				uucp.initBuf = buf[:n]
			}
			uucp.conn.SetReadDeadline(time.Time{})
			return nil
		}
	}
	return err
}

type udpUniClientProxy struct {
	ctx     context.Context
	initBuf []byte
	conn    net.Conn
}

func (uucp *udpUniClientProxy) Read(b []byte) (n int, err error) {
	if uucp.initBuf != nil {
		n = copy(b, uucp.initBuf)
		uucp.initBuf = nil
		return n, err
	}
	return uucp.conn.Read(b)
}

func (uucp *udpUniClientProxy) Write(b []byte) (n int, err error) {
	return uucp.conn.Write(b)
}

func (uucp *udpUniClientProxy) Close() error {
	return uucp.conn.Close()
}

func (uucp *udpUniClientProxy) LocalAddr() net.Addr {
	return uucp.conn.LocalAddr()
}

func (uucp *udpUniClientProxy) RemoteAddr() net.Addr {
	return uucp.conn.RemoteAddr()
}

func (uucp *udpUniClientProxy) SetDeadline(t time.Time) error {
	return uucp.conn.SetDeadline(t)
}

func (uucp *udpUniClientProxy) SetReadDeadline(t time.Time) error {
	return uucp.conn.SetReadDeadline(t)
}

func (uucp *udpUniClientProxy) SetWriteDeadline(t time.Time) error {
	return uucp.conn.SetWriteDeadline(t)
}
