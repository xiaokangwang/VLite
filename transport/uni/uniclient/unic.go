package uniclient

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"io"
	"net"
)

type UnifiedConnectionClient struct {
	dialer     transport.UnderlayTransportDialer
	ConnID     []byte
	Iter       int
	ctx        context.Context
	RxChan     chan []byte
	TxChan     chan []byte
	IterCancel context.CancelFunc
	connctx    context.Context
}

func (u *UnifiedConnectionClient) Close() error {
	u.IterCancel()
	return nil
}

func NewUnifiedConnectionClient(dialer transport.UnderlayTransportDialer, ctx context.Context) *UnifiedConnectionClient {
	Connid := make([]byte, 24)
	io.ReadFull(rand.Reader, Connid)
	return &UnifiedConnectionClient{dialer: dialer, ctx: ctx, ConnID: Connid, TxChan: make(chan []byte, 8), RxChan: make(chan []byte, 8)}
}
func (u *UnifiedConnectionClient) Connect(ctx context.Context) (net.Conn, error, context.Context) {

	u.connctx = ctx
	conn, err, connctx := u.connectUnder(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err, nil
	}
	ctx2, canc := context.WithCancel(connctx)
	u.IterCancel = canc

	go u.Tx(conn, ctx2)
	go u.Rx(conn, ctx2)

	return adp.NewRxTxToConn(u.TxChan, u.RxChan, u), nil, connctx
}

func (u *UnifiedConnectionClient) connectUnder(ctx context.Context) (net.Conn, error, context.Context) {
	u.Iter++
	Eouic := &interfaces.ExtraOptionsUniConnAttribValue{
		ID:   u.ConnID,
		Rand: nil,
		Iter: int32(u.Iter),
	}
	vctx := context.WithValue(ctx, interfaces.ExtraOptionsUniConnAttrib, Eouic)
	return u.dialer.Connect(vctx)
}

func (u *UnifiedConnectionClient) ReconnectUnder(ctx context.Context) error {
	u.IterCancel()
	conn, err, connctx := u.connectUnder(ctx)
	if err != nil {
		return err
	}
	ctx2, canc := context.WithCancel(connctx)
	u.IterCancel = canc

	go u.Tx(conn, ctx2)
	go u.Rx(conn, ctx2)
	return nil
}

func (uct *UnifiedConnectionClient) Rx(conn net.Conn, ctx context.Context) {
	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		data := buf[:n]
		uct.RxChan <- data
		if ctx.Err() != nil {
			return
		}
		if uct.connctx.Err() != nil {
			return
		}
	}
}
func (uct *UnifiedConnectionClient) Tx(conn net.Conn, ctx context.Context) {
	for {
		select {
		case data := <-uct.TxChan:
			_, err := conn.Write(data)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case <-ctx.Done():
			return
		case <-uct.connctx.Done():
			return
		}
	}
}
