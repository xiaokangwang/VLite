package udpunis

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"net"
	"reflect"
	"time"
)

func NewUdpUniServer(password string,
	ctx context.Context, upper transport.UnderlayTransportListener) *UdpUniServer {
	return &UdpUniServer{
		password: password,
		ctx:      ctx,
		hh:       headerHolder.NewHttpHeaderHolderProcessor2(password, "UdpUniSecret"),
		upper:    upper,
	}
}

type UdpUniServer struct {
	password string
	ctx      context.Context

	hh *headerHolder.HttpHeaderHolderProcessor

	upper transport.UnderlayTransportListener
}

func (uus *UdpUniServer) Connection(conn net.Conn, ctx context.Context) context.Context {

	InitialDataVal := ctx.Value(interfaces.ExtraOptionsUDPInitialData)
	InitialData := InitialDataVal.(*interfaces.ExtraOptionsUDPInitialDataValue).Data
	phv := uus.hh.Open(string(InitialData))
	if phv == nil {
		fmt.Println("Unable to decrypt initial data")
		return nil
	}

	univ := &interfaces.ExtraOptionsUniConnAttribValue{
		ID:   phv.ConnID[:],
		Rand: phv.Rand[:],
		Iter: phv.ConnIter,
	}

	ctx = context.WithValue(ctx, interfaces.ExtraOptionsUniConnAttrib, univ)

	return uus.upper.Connection(&udpUniClientProxy{conn: conn, initBuf: InitialData}, ctx)

}

func (uus *UdpUniServer) AsUnderlayTransportListener() transport.UnderlayTransportListener {
	return uus
}

type udpUniClientProxy struct {
	initBuf []byte
	conn    net.Conn
}

func (uucp *udpUniClientProxy) Read(b []byte) (n int, err error) {
	for {
		n, err = uucp.conn.Read(b)
		if reflect.DeepEqual(b[:n], uucp.initBuf) {
			uucp.Write(uucp.initBuf)
			continue
		}
		return
	}
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
