package contextAwareConn

import (
	"context"
	"net"
	"time"
)

func NewContextAwareConn(conn net.Conn, ctx context.Context) *ContextAwareConn {
	c := &ContextAwareConn{conn: conn, ctx: ctx}
	return c
}

type ContextAwareConn struct {
	conn net.Conn
	ctx  context.Context
}

func (c ContextAwareConn) Read(b []byte) (n int, err error) {
	if c.ctx.Err() != nil {
		return 0, c.ctx.Err()
	}
	return c.conn.Read(b)
}

func (c ContextAwareConn) Write(b []byte) (n int, err error) {
	if c.ctx.Err() != nil {
		return 0, c.ctx.Err()
	}
	return c.conn.Write(b)
}

func (c ContextAwareConn) Close() error {
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}
	return c.conn.Close()
}

func (c ContextAwareConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c ContextAwareConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c ContextAwareConn) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (c ContextAwareConn) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (c ContextAwareConn) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}
