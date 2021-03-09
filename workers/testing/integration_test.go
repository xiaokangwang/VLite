package testing

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/interfaces"
	client2 "github.com/xiaokangwang/VLite/workers/client"
	server2 "github.com/xiaokangwang/VLite/workers/server"
	"log"
	"net"
	"testing"
	"time"
)

type Stub struct {
}

func (s Stub) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	panic("implement me")
}

func TestInitializeNil(t *testing.T) {
	server := server2.UDPServer(context.Background(),
		make(chan server2.UDPServerTxToClientTraffic),
		make(chan server2.UDPServerTxToClientDataTraffic),
		make(chan server2.UDPServerRxFromClientTraffic), &Stub{})
	_ = server

	client := client2.UDPClient(context.Background(),
		make(chan client2.UDPClientTxToServerTraffic),
		make(chan client2.UDPClientTxToServerDataTraffic),
		make(chan client2.UDPClientRxFromServerTraffic),
		make(chan interfaces.UDPPacket),
		make(chan interfaces.UDPPacket), &Stub{})
	_ = client
}

func DataCopier(context context.Context,
	TxToServer chan client2.UDPClientTxToServerTraffic,
	TxToServerData chan client2.UDPClientTxToServerDataTraffic,
	RxFromServer chan client2.UDPClientRxFromServerTraffic,
	TxToClient chan server2.UDPServerTxToClientTraffic,
	TxToClientData chan server2.UDPServerTxToClientDataTraffic,
	RxFromClient chan server2.UDPServerRxFromClientTraffic) {

	go func() {
		for {
			select {
			case <-context.Done():
				return
			case data := <-TxToServer:
				spew.Dump(data)
				RxFromClient <- server2.UDPServerRxFromClientTraffic(data)
				break
			case data := <-TxToServerData:
				spew.Dump(data)
				RxFromClient <- server2.UDPServerRxFromClientTraffic(data)
				break
			case data := <-TxToClient:
				spew.Dump(data)
				RxFromServer <- client2.UDPClientRxFromServerTraffic(data)
				break
			case data := <-TxToClientData:
				spew.Dump(data)
				RxFromServer <- client2.UDPClientRxFromServerTraffic(data)
				break
			}
		}

	}()

}

func TestInitializeWithDataCopy(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()
	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy2(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy3(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy4(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy5(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TxPacket.Source.Port += 1
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy6(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 10; i++ {
			TxPacket.Source.Port = 19000 + i
			TunnelRxFromTun <- TxPacket

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
		}
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy6WithShortTimeout(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	ccontext = context.WithValue(ccontext, interfaces.ExtraOptionsUDPTimeoutTime, &interfaces.ExtraOptionsUDPTimeoutTimeValue{TimeoutTimeInSeconds: 1})

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 10; i++ {
			TxPacket.Source.Port = 19000 + i
			TunnelRxFromTun <- TxPacket

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
		}
		<-time.NewTimer(time.Second * 4).C
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopyFullCone(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	l2, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 19999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
			<-time.NewTimer(time.Second * 1).C
			_, err = l2.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TxPacket2 := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 19999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	RxPacket2 := <-TunnelTxToTun

	assert.Equal(t, TxPacket2.Payload, RxPacket2.Payload)
	assert.Equal(t, TxPacket2.Source.IP.To4(), RxPacket2.Dest.IP.To4())
	assert.Equal(t, TxPacket2.Source.Port, RxPacket2.Dest.Port)
	assert.Equal(t, TxPacket2.Dest.IP.To4(), RxPacket2.Source.IP.To4())
	assert.Equal(t, TxPacket2.Dest.Port, RxPacket2.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}

func TestInitializeWithDataCopy5_Plus(t *testing.T) {
	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	//ccontext = context.WithValue(ccontext,interfaces.ExtraOptionsUDPTimeoutTime,&interfaces.ExtraOptionsUDPTimeoutTimeValue{TimeoutTimeInSeconds:1})

	S_S2CTraffic := make(chan server2.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server2.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server2.UDPServerRxFromClientTraffic, 8)

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	DataCopier(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic)

	server := server2.UDPServer(ccontext,
		S_S2CTraffic,
		S_S2CDataTraffic,
		S_C2STraffic, &Stub{})
	_ = server

	client := client2.UDPClient(ccontext,
		C_C2STraffic,
		C_C2SDataTraffic,
		C_S2CTraffic,
		TunnelTxToTun,
		TunnelRxFromTun, &Stub{})
	_ = client

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 800; i++ {
			TxPacketL := interfaces.UDPPacket{
				Source: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 11000,
					Zone: "",
				},
				Dest: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 18999,
					Zone: "",
				},
				Payload: []byte("Test"),
			}
			TxPacketL.Source.Port += i
			TunnelRxFromTun <- TxPacketL

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacketL.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacketL.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacketL.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacketL.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacketL.Dest.Port, RxPacket.Source.Port)
			<-time.NewTimer(time.Nanosecond * 1).C
		}
	}

	<-time.NewTimer(time.Second * 4).C

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 800; i++ {
			TxPacketL := interfaces.UDPPacket{
				Source: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 11000,
					Zone: "",
				},
				Dest: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 18999,
					Zone: "",
				},
				Payload: []byte("Test"),
			}
			TxPacketL.Source.Port += i
			TunnelRxFromTun <- TxPacketL

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacketL.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacketL.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacketL.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacketL.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacketL.Dest.Port, RxPacket.Source.Port)
			<-time.NewTimer(time.Nanosecond * 1).C
		}
	}

	<-time.NewTimer(time.Second * 4).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C
}
