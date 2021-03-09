package udpsctpserver

import (
	"bufio"
	"io"
)

type BufferedConn struct {
	rwc  io.ReadWriteCloser
	rwcb *bufio.Reader
}

func (b *BufferedConn) prepare() {
	b.rwcb = bufio.NewReaderSize(b.rwc, 65535*4)
}

func NewBufferedConn(str io.ReadWriteCloser) *BufferedConn {
	r := &BufferedConn{rwc: str}
	r.prepare()
	return r
}

func (b BufferedConn) Read(p []byte) (n int, err error) {
	return b.rwcb.Read(p)
}

func (b BufferedConn) Write(p []byte) (n int, err error) {
	return b.rwc.Write(p)
}

func (b BufferedConn) Close() error {
	return b.rwc.Close()
}
