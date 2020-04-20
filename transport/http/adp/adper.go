package adp

import (
	"io"
	"net"
	"time"
)

func NewRxTxToConn(TxChan chan []byte,
	RxChan chan []byte, closer io.Closer) *RxTxToConn {
	adpr := &RxTxToConn{TxChan: TxChan, RxChan: RxChan, closer: closer}
	return adpr
}

type RxTxToConn struct {
	TxChan chan []byte
	RxChan chan []byte
	closer io.Closer
}

func (r RxTxToConn) Read(b []byte) (n int, err error) {
	select {
	case rx := <-r.RxChan:
		n = copy(b, rx)
		if n != len(rx) {
			return 0, io.ErrShortBuffer
		}
		return n, nil
	case <-time.NewTimer(time.Second * 400).C:
		return 0, io.EOF
	}

}

func (r RxTxToConn) Write(b []byte) (n int, err error) {
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
