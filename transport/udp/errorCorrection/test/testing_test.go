package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/udp/errorCorrection/assembly"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
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

func TestSendingRateTrail(t *testing.T) {
	ctx := context.TODO()
	opt := &interfaces.ExtraOptionsFECPacketAssemblyOptValue{}
	opt.TxEpochTimeInMs = 2000
	opt.RxMaxTimeInSecond = 40

	ctx = context.WithValue(ctx, interfaces.ExtraOptionsFECPacketAssemblyOpt,
		opt)

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

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

	finishWait := &sync.Mutex{}

	go func() {
		var buffer [65536]byte
		n, err := rxConn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)
		finishWait.Unlock()
	}()

	n, err := txConn.Write(TestingData)

	assert.Equal(t, TestingDataLen, n)
	assert.Nil(t, err)
	finishWait.Lock()

}

func TestSendingRateRealOne(t *testing.T) {
	ctx := context.TODO()
	opt := &interfaces.ExtraOptionsFECPacketAssemblyOptValue{}
	opt.TxEpochTimeInMs = 2000
	opt.RxMaxTimeInSecond = 40

	ctx = context.WithValue(ctx, interfaces.ExtraOptionsFECPacketAssemblyOpt,
		opt)

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

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

	finishWait2 := &sync.Mutex{}
	finishWait2.Lock()
	go func() {
		var buffer [65536]byte
		n, err := rxConn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)
		finishWait2.Lock()
	}()

	n, err := txConn.Write(TestingData)

	assert.Equal(t, TestingDataLen, n)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	tx1 := *mcTx.TxCount
	time.Sleep(time.Second * 2)
	tx2 := *mcTx.TxCount
	time.Sleep(time.Second * 2)
	tx3 := *mcTx.TxCount
	time.Sleep(time.Second * 2)
	tx4 := *mcTx.TxCount
	time.Sleep(time.Second * 2)
	tx5 := *mcTx.TxCount

	assert.Equal(t, 1, tx1)
	assert.Equal(t, 3, tx2) // Why?
	assert.Equal(t, 4, tx3)
	assert.Equal(t, 5, tx4)
	assert.Equal(t, 5, tx5)

	finishWait2.Unlock()

}

func TestSendingLossRepair(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 200)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 200)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 95)

	assert.Less(t, *mcTx.TxCount, 500)
}

func TestSendingLossNoFEC(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 200)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 200)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()
	tx.FECEnabled = 0
	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Equal(t, recSum, 84)

	assert.Equal(t, *mcTx.TxCount, 100)
}

func TestSendingLossRepairHuge(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 200)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 200)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 950)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairHugeLossPlus(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 400)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 400)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 950)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairHugeLossPlusPlus(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 600)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 600)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 700)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairLossPlus(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 400)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 400)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 95)

	assert.Less(t, *mcTx.TxCount, 500)
}

func TestSendingLossRepairLossPlusPlus(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConn(r, w2, 600)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConn(r3, w, 600)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 85)

	assert.Less(t, *mcTx.TxCount, 500)
}

func NewMockConnRandDelay(r io.Reader, w io.Writer, f int) *MockConnRandDelay {
	return &MockConnRandDelay{
		RxCount:         new(int),
		TxCount:         new(int),
		Reader:          r,
		Writer:          w,
		LossyConn:       rand.New(rand.NewSource(1234)),
		LossyConnFactor: f,
	}
}

type MockConnRandDelay struct {
	RxCount *int
	TxCount *int
	Reader  io.Reader
	Writer  io.Writer

	LossyConn       *rand.Rand
	LossyConnFactor int
}

func (m MockConnRandDelay) Read(b []byte) (n int, err error) {
	(*m.RxCount)++
	return m.Reader.Read(b)
}

func (m MockConnRandDelay) Write(b []byte) (n int, err error) {
	(*m.TxCount)++
	if m.LossyConnFactor != 0 {
		if m.LossyConn.Intn(1000) < m.LossyConnFactor {
			return len(b), nil
		}
	}
	go func() {
		timetoDelay := rand.Intn(200)
		time.Sleep(time.Duration(timetoDelay) * time.Millisecond)
		m.Writer.Write(b)
	}()

	return len(b), nil
}

func (m MockConnRandDelay) Close() error {
	return nil
}

func (m MockConnRandDelay) LocalAddr() net.Addr {
	panic("implement me")
}

func (m MockConnRandDelay) RemoteAddr() net.Addr {
	panic("implement me")
}

func (m MockConnRandDelay) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (m MockConnRandDelay) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (m MockConnRandDelay) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

func TestSendingLossRepairWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 200)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 200)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 95)

	assert.Less(t, *mcTx.TxCount, 500)
}

func TestSendingLossRepairHugeWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 200)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 200)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 950)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairHugeLossPlusWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 400)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 400)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 950)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairHugeLossPlusPlusWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 600)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 600)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 1000; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(10)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 700)

	assert.Less(t, *mcTx.TxCount, 5000)
}

func TestSendingLossRepairLossPlusWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 400)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 400)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 95)

	assert.Less(t, *mcTx.TxCount, 500)
}

func TestSendingLossRepairLossPlusPlusWriteRandDelay(t *testing.T) {
	ctx := context.TODO()

	r, w := io.Pipe()

	r2, w2 := io.Pipe()

	r3, w3 := io.Pipe()

	_ = r2

	_ = w3

	mcRx := NewMockConnRandDelay(r, w2, 600)
	rx := assembly.NewPacketAssembly(ctx, mcRx)
	rxConn := rx.AsConn()
	_ = rxConn

	mcTx := NewMockConnRandDelay(r3, w, 600)
	tx := assembly.NewPacketAssembly(ctx, mcTx)
	txConn := tx.AsConn()

	_ = txConn

	recSum := 0
	go func() {
		var buffer [65536]byte
		for {
			n, err := rxConn.Read(buffer[:])
			if err == nil {
				recSum += 1
				s := buffer[:n]
				assert.Regexp(t, "^TestingData([0-9])+$", string(s))
			} else {
				t.Fail()
			}
		}

	}()

	sleepr := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		S := "TestingData"
		is := strconv.Itoa(i)
		a := S + is
		txConn.Write([]byte(a))
		sleepTime := sleepr.Intn(100)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	time.Sleep(time.Second)

	assert.Greater(t, recSum, 85)

	assert.Less(t, *mcTx.TxCount, 500)
}
