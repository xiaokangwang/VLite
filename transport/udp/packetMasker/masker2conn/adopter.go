package masker2conn

import (
	"bytes"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"net"
	"time"
)

type MaskerAdopter struct {
	masker interfaces.Masker
	in     net.Conn
}

func (m MaskerAdopter) Read(b []byte) (n int, err error) {
	var buf [65536]byte
	n, err = m.in.Read(buf[:])
	if err != nil {
		return 0, err
	}

	reader := bytes.NewReader(buf[:n])
	writer := bytes.NewBuffer(nil)

	err = m.masker.UnMask(reader, writer)
	if err != nil {
		fmt.Println(err.Error())
	}
	n = copy(b, writer.Bytes())

	if n != writer.Len() {
		return 0, io.ErrShortBuffer
	}

	return n, nil
}

func (m MaskerAdopter) Write(b []byte) (n int, err error) {
	reader := bytes.NewReader(b)
	writer := bytes.NewBuffer(nil)

	err = m.masker.Mask(reader, writer)
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = m.in.Write(writer.Bytes())
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (m MaskerAdopter) Close() error {
	return m.in.Close()
}

func (m MaskerAdopter) LocalAddr() net.Addr {
	return m.in.LocalAddr()
}

func (m MaskerAdopter) RemoteAddr() net.Addr {
	return m.in.RemoteAddr()
}

func (m MaskerAdopter) SetDeadline(t time.Time) error {
	return m.in.SetDeadline(t)
}

func (m MaskerAdopter) SetReadDeadline(t time.Time) error {
	return m.in.SetDeadline(t)
}

func (m MaskerAdopter) SetWriteDeadline(t time.Time) error {
	return m.in.SetWriteDeadline(t)
}

func NewMaskerAdopter(masker interfaces.Masker, in net.Conn) *MaskerAdopter {
	mA := &MaskerAdopter{
		masker: masker,
		in:     in,
	}
	return mA
}
