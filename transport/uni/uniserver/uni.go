package uniserver

import (
	"context"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/antiReplayWindow"
	"net"
	"sync"
)

func NewUnifiedConnectionTransportHub(Uplistener transport.UnderlayTransportListener, ctx context.Context) *UnifiedConnectionTransportHub {
	return &UnifiedConnectionTransportHub{Uplistener: Uplistener, ctx: ctx}
}

type UnifiedConnectionTransportHub struct {
	Conns      sync.Map
	Uplistener transport.UnderlayTransportListener
	ctx        context.Context
}

func (uic *UnifiedConnectionTransportHub) Connection(conn net.Conn, ctx context.Context) context.Context {
	return uic.onConnection(conn, ctx)
}

type UnifiedConnectionTransport struct {
	ConnID                 []byte
	Arw                    *antiReplayWindow.AntiReplayWindow
	LastConnIter           int32
	LastCOnnIterCancelFunc map[string]context.CancelFunc
	TxChan                 chan []byte
	RxChan                 chan []byte
	connctx                context.Context
}

func (u UnifiedConnectionTransport) Close() error {
	for _, v := range u.LastCOnnIterCancelFunc {
		v()
	}
	return nil
}
