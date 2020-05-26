package uniclient

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
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

	connAdpCancel context.CancelFunc
}

func (u *UnifiedConnectionClient) Close() error {
	u.IterCancel()
	return nil
}

func NewUnifiedConnectionClient(dialer transport.UnderlayTransportDialer, ctx context.Context) *UnifiedConnectionClient {
	Connid := make([]byte, 24)
	io.ReadFull(rand.Reader, Connid)

	ucc := &UnifiedConnectionClient{dialer: dialer, ctx: ctx, ConnID: Connid, TxChan: make(chan []byte, 8), RxChan: make(chan []byte, 8)}

	return ucc
}
func (u *UnifiedConnectionClient) Connect(ctx context.Context) (net.Conn, error, context.Context) {
	if u.connAdpCancel != nil {
		u.connAdpCancel()
	}
	if u.IterCancel != nil {
		u.IterCancel()
	}
	u.connctx = ctx
	conn, err, connctx := u.connectUnder2(ctx, true)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err, nil
	}
	ctx2, canc := context.WithCancel(connctx)
	u.IterCancel = canc

	go func() {
		<-ctx2.Done()
		fmt.Println("closing conn", conn.Close())

	}()

	go u.Tx(conn, ctx2)
	go u.Rx(conn, ctx2)

	var ctxAdp context.Context
	ctxAdp, u.connAdpCancel = context.WithCancel(connctx)

	if u.Iter == 1 {
		go u.ReconnectListener(connctx)
	}

	return adp.NewRxTxToConn(u.TxChan, u.RxChan, u, ctxAdp), nil, connctx
}

func (u *UnifiedConnectionClient) connectUnder2(ctx context.Context, shouldHandshake bool) (net.Conn, error, context.Context) {
	u.Iter++

	iterv := u.Iter
	if shouldHandshake {
		iterv = -iterv
	}
	Eouic := &interfaces.ExtraOptionsUniConnAttribValue{
		ID:   u.ConnID,
		Rand: nil,
		Iter: int32(iterv),
	}
	vctx := context.WithValue(ctx, interfaces.ExtraOptionsUniConnAttrib, Eouic)
	conn, err, ctx2 := u.dialer.Connect(vctx)
	return conn, err, ctx2
}

func (u *UnifiedConnectionClient) ReconnectUnder(ctx context.Context) error {
	u.IterCancel()
	conn, err, connctx := u.connectUnder2(ctx, false)
	if err != nil {
		return err
	}
	ctx2, canc := context.WithCancel(connctx)
	u.IterCancel = canc

	go func() {
		<-ctx2.Done()
		fmt.Println("closing conn", conn.Close())

	}()

	go u.Tx(conn, ctx2)
	go u.Rx(conn, ctx2)
	return nil
}

func (u *UnifiedConnectionClient) ReconnectUnderR(ctx context.Context) error {
	u.IterCancel()
	conn, err, connctx := u.connectUnder2(ctx, true)
	if err != nil {
		return err
	}
	ctx2, canc := context.WithCancel(connctx)
	u.IterCancel = canc

	go func() {
		<-ctx2.Done()
		fmt.Println("closing conn", conn.Close())

	}()

	go u.Tx(conn, ctx2)
	go u.Rx(conn, ctx2)
	return nil
}

func (u *UnifiedConnectionClient) ReconnectListener(connctx context.Context) {
	ConnIDString := connidutil.ConnIDToString(connctx)
	BusTopic := ibusTopic.ConnReHandShake(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(connctx)

	mbus.RegisterTopics(BusTopic)

	handshakeModeOptChan := make(chan ibusInterface.ConnReHandshake, 8)

	mbus.RegisterHandler(BusTopic+"UniClient", &bus.Handler{
		Handle: func(e *bus.Event) {
			d := e.Data.(ibusInterface.ConnReHandshake)
			select {
			case handshakeModeOptChan <- d:
			default:
				fmt.Println("WARNING: boost mode hint discarded")
			}

		},
		Matcher: BusTopic,
	})

	for {
		select {
		case <-connctx.Done():
			fmt.Println("UniClient Done")
			return
		case opt := <-handshakeModeOptChan:
			if opt.FullReHandshake {
				go u.ReconnectUnderR(connctx)
			} else {
				go u.ReconnectUnder(connctx)
			}

			fmt.Println("UniClient Rehandshake")
		}
	}

}

func (uct *UnifiedConnectionClient) Rx(conn net.Conn, ctx context.Context) {
	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("closing conn", conn.Close())
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
				fmt.Println("closing conn", conn.Close())
				return
			}
		case <-ctx.Done():
			return
		case <-uct.connctx.Done():
			return
		}
	}
}
