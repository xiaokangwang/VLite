package udptlssctp

import (
	"context"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/transport/http/httpServer"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"github.com/xiaokangwang/VLite/transport/udp/udpServer"
	"github.com/xiaokangwang/VLite/workers/server"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpServer"
	"net"
	"strings"
)

func NewUdptlsSctpServer(localAddress string, password string, ctx context.Context) *UdptlsSctpServer {
	utss := &UdptlsSctpServer{}
	utss.Address = localAddress
	utss.password = []byte(password)

	utss.msgbus = ibus.NewMessageBus()
	ctxwbus := context.WithValue(ctx, interfaces.ExtraOptionsMessageBus, utss.msgbus)

	utss.ctx = ctxwbus

	utss.ctx = ctx
	return utss
}

type UdptlsSctpServer struct {
	Address string
	ctx     context.Context

	udplistener interface{}

	password []byte

	ratelimitServerTCPWriteBytePerSecond int
	ratelimitServerTCPWriteMaxBucketSize int
	ratelimitServerTCPWriteInitialSize   int

	msgbus *bus.Bus
}

func (s UdptlsSctpServer) Connection(conn net.Conn, ctx context.Context) {
	s.Process(conn, ctx)
}

func (s *UdptlsSctpServer) Process(conn net.Conn, connctx context.Context) {

	ts := tcpServer.TCPServer{}

	S_S2CTraffic := make(chan server.UDPServerTxToClientTraffic, 8)
	S_S2CDataTraffic := make(chan server.UDPServerTxToClientDataTraffic, 8)
	S_C2STraffic := make(chan server.UDPServerRxFromClientTraffic, 8)

	S_S2CTraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)
	S_S2CDataTraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)
	S_C2STraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-S_S2CTraffic:
				S_S2CTraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}
	}(connctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-S_S2CDataTraffic:
				S_S2CDataTraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-S_C2STraffic2:
				S_C2STraffic <- server.UDPServerRxFromClientTraffic(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	relay := udpsctpserver.NewPacketRelayServer(conn, S_S2CTraffic2, S_S2CDataTraffic2, S_C2STraffic2, &ts, s.password, connctx)

	_ = relay

	udpserver := server.UDPServer(connctx, S_S2CTraffic, S_S2CDataTraffic, S_C2STraffic, relay)

	_ = udpserver

	relay.RateLimitTcpServerWrite(s.ratelimitServerTCPWriteBytePerSecond, s.ratelimitServerTCPWriteMaxBucketSize, s.ratelimitServerTCPWriteInitialSize)
}
func (s *UdptlsSctpServer) Up() {
	//Open Connection
	if strings.HasPrefix(s.Address, "http") {
		address := s.Address[4:]
		var v = httpServer.NewProviderServerSide(address, string(s.password), s, s.ctx)
		s.udplistener = v
	} else {
		var v = udpServer.NewUDPServer(s.Address, s.ctx, s)
		s.udplistener = v
	}

}

func (s *UdptlsSctpServer) RateLimitTcpServerWrite(ratelimitServerTCPWriteBytePerSecond int,
	ratelimitServerTCPWriteMaxBucketSize int,
	ratelimitServerTCPWriteInitialSize int) {
	s.ratelimitServerTCPWriteBytePerSecond = ratelimitServerTCPWriteBytePerSecond
	s.ratelimitServerTCPWriteMaxBucketSize = ratelimitServerTCPWriteMaxBucketSize
	s.ratelimitServerTCPWriteInitialSize = ratelimitServerTCPWriteInitialSize
}
