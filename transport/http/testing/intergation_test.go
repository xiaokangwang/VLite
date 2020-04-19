package testing

import (
	"bytes"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport/http/httpClient"
	"github.com/xiaokangwang/VLite/transport/http/httpServer"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"net"
	"testing"
	"time"
)

type listenerStub struct {
}

func (l listenerStub) Connection(conn net.Conn) {
	var buf [1601]byte
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Println(err)
			return
		}

		payload := buf[:n]

		conn.Write(payload)
	}

}

func TestSetupServer(t *testing.T) {
	listener := new(listenerStub)
	httpServer.NewProviderServerSide("127.0.0.1:8801", "pw", listener)
}

func TestSetupServer2(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8802", "pw", listener)
	_ = hs
}

func TestClientDial(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8803", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8803", 1, 1, "pw")

	conn := httpClientI.AsConn()

	var TestingData = []byte("Demo")
	var TestingDataLen = len(TestingData)

	n, err := conn.Write(TestingData)

	assert.Equal(t, TestingDataLen, n)
	assert.Nil(t, err)
	var buffer [65536]byte
	n, err = conn.Read(buffer[:])
	assert.Equal(t, TestingDataLen, n)
	assert.Equal(t, TestingData, buffer[:n])
	assert.Nil(t, err)

}

func TestFillerLen(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	l := &proto.HTTPLenHeader{}

	struc.Pack(buf, l)

	assert.Len(t, buf.Bytes(), 8)
}

func TestClientDialRepeat(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8804", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8804", 1, 1, "pw")

	conn := httpClientI.AsConn()

	for i := 0; i <= 600; i++ {
		var TestingData = []byte("Demo")
		var TestingDataLen = len(TestingData)

		n, err := conn.Write(TestingData)

		assert.Equal(t, TestingDataLen, n)
		assert.Nil(t, err)
		var buffer [65536]byte
		n, err = conn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)

	}

}

func TestClientDialRepeatMutiThreaded(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8805", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8805", 4, 4, "pw")

	time.Sleep(5 * time.Second)

	conn := httpClientI.AsConn()

	for i := 0; i <= 600; i++ {
		var TestingData = []byte("Demo")
		var TestingDataLen = len(TestingData)

		n, err := conn.Write(TestingData)

		assert.Equal(t, TestingDataLen, n)
		assert.Nil(t, err)
		var buffer [65536]byte
		n, err = conn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)

	}

}

func TestClientDialRepeatMutiThreadedMassive(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8806", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8806", 16, 16, "pw")

	time.Sleep(8 * time.Second)

	conn := httpClientI.AsConn()

	for i := 0; i <= 6000; i++ {
		var TestingData = []byte("Demo")
		var TestingDataLen = len(TestingData)

		n, err := conn.Write(TestingData)

		assert.Equal(t, TestingDataLen, n)
		assert.Nil(t, err)
		var buffer [65536]byte
		n, err = conn.Read(buffer[:])
		assert.Equal(t, TestingDataLen, n)
		assert.Equal(t, TestingData, buffer[:n])
		assert.Nil(t, err)

	}

}

func TestBufferSizeDeduction(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8807", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8807", 1, 1, "pw")

	time.Sleep(1 * time.Second)

	conn := httpClientI.AsConn()

	_ = conn

	res := httpClientI.DialRxTestConnection(2)

	assert.Equal(t, wrapper.ShorterThanExpected, res)
}

func TestBufferSizeDeductionTx(t *testing.T) {
	listener := new(listenerStub)
	hs := httpServer.NewProviderServerSide("127.0.0.1:8808", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8808", 1, 1, "pw")

	time.Sleep(1 * time.Second)

	conn := httpClientI.AsConn()

	_ = conn

	res := httpClientI.DialTxConnectionTest(2)

	assert.Equal(t, wrapper.ShorterThanExpected, res)

}

func TestBufferSizeDeductionMockRxBuffered(t *testing.T) {
	listener := new(listenerStub)
	httpServer.WriteBufferSize = 4096

	hs := httpServer.NewProviderServerSide("127.0.0.1:8809", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8809", 1, 1, "pw")

	time.Sleep(1 * time.Second)

	conn := httpClientI.AsConn()

	_ = conn

	res := httpClientI.DialRxTestConnection(2)

	assert.Equal(t, wrapper.LongerThanExpected, res)
}

/*
func TestBufferSizeDeductionMockRxBuffered1(t *testing.T) {
	listener := new(listenerStub)
	httpServer.WriteBufferSize = 4096

	hs := httpServer.NewProviderServerSide("127.0.0.1:8810", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8810", 1, 1, "pw")

	time.Sleep(1 * time.Second)

	conn := httpClientI.AsConn()

	_ = conn

	res := httpClientI.DialRxTestConnection(4096)

	_ = res
}

func TestBufferSizeDeductionMockRxBuffered2(t *testing.T) {
	listener := new(listenerStub)
	httpServer.WriteBufferSize = 4096

	hs := httpServer.NewProviderServerSide("127.0.0.1:8819", "pw", listener)
	_ = hs

	time.Sleep(time.Second)

	httpClientI := httpClient.NewProviderClient("http://127.0.0.1:8819", 1, 1, "pw")

	time.Sleep(1 * time.Second)

	conn := httpClientI.AsConn()

	_ = conn

	ProbeWindow := 65536
	ProbeCurrent := 65536

	for i := 20; i > 0; i++ {
		res := httpClientI.DialRxTestConnection(ProbeCurrent)
		if res == wrapper.Exact {
			fmt.Println("Rx buffer is ", ProbeCurrent)
			return
		}
		if res == wrapper.LongerThanExpected {
			fmt.Println("Rx buffer is longer than ", ProbeCurrent)
			ProbeWindow /= 2
			if ProbeWindow == 0 {
				ProbeWindow = 1
			}
			ProbeCurrent += ProbeWindow
		}
		if res == wrapper.ShorterThanExpected {
			fmt.Println("Rx buffer is shorter than ", ProbeCurrent)
			ProbeWindow /= 2
			if ProbeWindow == 0 {
				ProbeWindow = 1
			}
			ProbeCurrent -= ProbeWindow
		}
	}

}
*/
