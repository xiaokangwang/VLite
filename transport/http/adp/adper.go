package adp

import (
	"context"
	"io"
	"net"
	"time"
)

func NewRxTxToConn(TxChan chan []byte,
	RxChan chan []byte, closer io.Closer, ctx context.Context) *RxTxToConn {
	adpr := &RxTxToConn{TxChan: TxChan, RxChan: RxChan, closer: closer, ctx: ctx}
	return adpr
}

type RxTxToConn struct {
	TxChan chan []byte
	RxChan chan []byte
	closer io.Closer
	ctx    context.Context
}

func (r RxTxToConn) Read(b []byte) (n int, err error) {
	timer := time.NewTimer(time.Second * 400)
	defer timer.Stop()
	select {
	case rx := <-r.RxChan:
		n = copy(b, rx)
		if n != len(rx) {
			return 0, io.ErrShortBuffer
		}
		return n, nil
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	case <-timer.C:
		return 0, io.EOF
	}

}

func (r RxTxToConn) Write(b []byte) (n int, err error) {
	if r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}
	r.TxChan <- b
	return len(b), nil
}

func (r RxTxToConn) Close() error {
	return r.closer.Close()
}

func (r RxTxToConn) LocalAddr() net.Addr {
	panic("implement me")
}

func (r RxTxToConn) RemoteAddr() net.Addr {
	return nil
}

func (r RxTxToConn) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (r RxTxToConn) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (r RxTxToConn) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}
