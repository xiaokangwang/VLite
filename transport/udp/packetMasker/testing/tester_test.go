package testing

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/constantXor"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/layers"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/masker2conn"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/prepend"
	"net"
	"testing"
	"time"
)

type SynConnW struct {
	b *bytes.Buffer
}

func (s SynConnW) Read(b []byte) (n int, err error) {
	panic("implement me")
}

func (s SynConnW) Write(b []byte) (n int, err error) {
	return s.b.Write(b)
}

func (s SynConnW) Close() error {
	panic("implement me")
}

func (s SynConnW) LocalAddr() net.Addr {
	panic("implement me")
}

func (s SynConnW) RemoteAddr() net.Addr {
	panic("implement me")
}

func (s SynConnW) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (s SynConnW) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (s SynConnW) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

type SynConnR struct {
	b *bytes.Reader
}

func (s SynConnR) Read(b []byte) (n int, err error) {
	return s.b.Read(b)
}

func (s SynConnR) Write(b []byte) (n int, err error) {
	panic("implement me")
}

func (s SynConnR) Close() error {
	panic("implement me")
}

func (s SynConnR) LocalAddr() net.Addr {
	panic("implement me")
}

func (s SynConnR) RemoteAddr() net.Addr {
	panic("implement me")
}

func (s SynConnR) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (s SynConnR) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (s SynConnR) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

func TestConnectionWithoutMasker(t *testing.T) {
	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	connadp.Write(TestingIn)
	assert.Equal(t, TestingIn, synconn.b.Bytes())
}

func TestConnectionReadingWithoutMasker(t *testing.T) {
	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{})
	TestingIn := []byte("Testing Input")
	synconn := SynConnR{b: bytes.NewReader(TestingIn)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)

	var buf [65536]byte
	n, err := connadp.Read(buf[:])
	assert.Nil(t, err)

	assert.Equal(t, TestingIn, buf[:n])
}

func TestConnectionWithPrependMasker(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte("AAZZ"))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	TestingOut := []byte("AAZZTesting Input")
	connadp.Write(TestingIn)
	assert.Equal(t, TestingOut, synconn.b.Bytes())

}

func TestConnectionReadingWithPrependMaskerMasker(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte("AAZZ"))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	TestingIn := []byte("AAZZTesting Input")
	synconn := SynConnR{b: bytes.NewReader(TestingIn)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)

	var buf [65536]byte
	n, err := connadp.Read(buf[:])
	assert.Nil(t, err)

	TestingOut := []byte("Testing Input")
	assert.Equal(t, TestingOut, buf[:n])
}

func TestConnectionWithXorMasker(t *testing.T) {
	Masker := constantXor.NewXorMasker("Testing")

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	connadp.Write(TestingIn)
	assert.NotEqual(t, TestingIn, synconn.b.Bytes())
	assert.Equal(t, len(TestingIn), synconn.b.Len())

}

func TestConnectionWithXorMaskerReverse(t *testing.T) {
	Masker := constantXor.NewXorMasker("Testing")

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	connadp.Write(TestingIn)
	assert.NotEqual(t, TestingIn, synconn.b.Bytes())
	assert.Equal(t, len(TestingIn), synconn.b.Len())

	Masker2 := constantXor.NewXorMasker("Testing")

	ZeroLayer2 := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker2})
	TestingIn2 := synconn.b.Bytes()
	synconn2 := SynConnR{b: bytes.NewReader(TestingIn2)}
	connadp2 := masker2conn.NewMaskerAdopter(ZeroLayer2, synconn2)

	var buf2 [65536]byte
	n2, err2 := connadp2.Read(buf2[:])
	assert.Nil(t, err2)

	assert.Equal(t, TestingIn, buf2[:n2])
}

func TestConnectionWithDoublePrependMasker(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte("AAZZ"))
	Masker2 := prepend.NewPrependingMasker([]byte("XXVV"))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker, Masker2})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	TestingOut := []byte("XXVVAAZZTesting Input")
	connadp.Write(TestingIn)
	assert.Equal(t, TestingOut, synconn.b.Bytes())

}

func TestConnectionReadingWithDoublePrependMaskerMasker(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte("AAZZ"))
	Masker2 := prepend.NewPrependingMasker([]byte("XXVV"))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker, Masker2})
	TestingIn := []byte("XXVVAAZZTesting Input")
	synconn := SynConnR{b: bytes.NewReader(TestingIn)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)

	var buf [65536]byte
	n, err := connadp.Read(buf[:])
	assert.Nil(t, err)

	TestingOut := []byte("Testing Input")
	assert.Equal(t, TestingOut, buf[:n])
}

func TestConnectionWithDoubleMaskerReverse(t *testing.T) {
	Masker := constantXor.NewXorMasker("Testing")
	Masker2 := constantXor.NewXorMasker("Testing2")

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker, Masker2})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	connadp.Write(TestingIn)
	assert.NotEqual(t, TestingIn, synconn.b.Bytes())
	assert.Equal(t, len(TestingIn), synconn.b.Len())

	Masker3 := constantXor.NewXorMasker("Testing")
	Masker4 := constantXor.NewXorMasker("Testing2")

	ZeroLayer2 := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker3, Masker4})
	TestingIn2 := synconn.b.Bytes()
	synconn2 := SynConnR{b: bytes.NewReader(TestingIn2)}
	connadp2 := masker2conn.NewMaskerAdopter(ZeroLayer2, synconn2)

	var buf2 [65536]byte
	n2, err2 := connadp2.Read(buf2[:])
	assert.Nil(t, err2)

	assert.Equal(t, TestingIn, buf2[:n2])
}


func TestConnectionWithPrependMaskerCornerCaseZeroLengthPrefix(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte(""))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	synconn := SynConnW{b: bytes.NewBuffer(nil)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)
	TestingIn := []byte("Testing Input")
	TestingOut := []byte("Testing Input")
	connadp.Write(TestingIn)
	assert.Equal(t, TestingOut, synconn.b.Bytes())

}

func TestConnectionReadingWithPrependMaskerCornerCaseZeroLengthPrefix(t *testing.T) {
	Masker := prepend.NewPrependingMasker([]byte(""))

	ZeroLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{Masker})
	TestingIn := []byte("Testing Input")
	synconn := SynConnR{b: bytes.NewReader(TestingIn)}
	connadp := masker2conn.NewMaskerAdopter(ZeroLayer, synconn)

	var buf [65536]byte
	n, err := connadp.Read(buf[:])
	assert.Nil(t, err)

	TestingOut := []byte("Testing Input")
	assert.Equal(t, TestingOut, buf[:n])
}