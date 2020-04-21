package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/transport/udp/errorCorrection/assembly"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"
)

type MockConn struct {
	RxCount *int
	TxCount *int
	Reader  io.Reader
	Writer  io.Writer

	LossyConn       *rand.Rand
	LossyConnFactor int
}

func (m MockConn) Read(b []byte) (n int, err error) {
	(*m.RxCount)++
	return m.Reader.Read(b)
}

func (m MockConn) Write(b []byte) (n int, err error) {
	(*m.TxCount)++
	if m.LossyConnFactor != 0 {
		if m.LossyConn.Intn(1000) < m.LossyConnFactor {
			return len(b), nil
		}
	}
	return m.Writer.Write(b)
}

func (m MockConn) Close() error {
	return nil
}

func (m MockConn) LocalAddr() net.Addr {
	panic("implement me")
}

func (m MockConn) RemoteAddr() net.Addr {
	panic("implement me")
}

func (m MockConn) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (m MockConn) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (m MockConn) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

func NewMockConn(r io.Reader, w io.Writer, f int) *MockConn {
	return &MockConn{
		RxCount:         new(int),
		TxCount:         new(int),
		Reader:          r,
		Writer:          w,
		LossyConn:       rand.New(rand.NewSource(1234)),
		LossyConnFactor: f,
	}
}

func TestInitialize(t *testing.T) {
	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	ctx := context.TODO()
	mcRx := NewMockConn(r, w2, 0)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 0)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn
}

func TestSingleMessage(t *testing.T) {
	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	ctx := context.TODO()
	mcRx := NewMockConn(r, w2, 0)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 0)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	var TestingData = []byte("Demo")
	var TestingDataLen = len(TestingData)

	n, err := txConn.Write(TestingData)

	assert.Equal(t, TestingDataLen, n)
	assert.Nil(t, err)
	var buffer [65536]byte
	n, err = rxConn.Read(buffer[:])
	assert.Equal(t, TestingDataLen, n)
	assert.Equal(t, TestingData, buffer[:n])
	assert.Nil(t, err)
}

func TestManyMessage(t *testing.T) {
	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	ctx := context.TODO()
	mcRx := NewMockConn(r, w2, 0)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 0)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	for i := 0; i <= 10000; i++ {
		var TestingData = []byte("Demo")
		var TestingDataLen = len(TestingData)

		n, err := txConn.Write(TestingData)

		assert.Equal(t, TestingDataLen, n)
		assert.Nil(t, err)

		var buffer [65536]byte
		n, err = rxConn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)
	}

}
