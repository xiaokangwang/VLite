package udptlssctp

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"github.com/xiaokangwang/VLite/transport/udp/udpServer"
	"github.com/xiaokangwang/VLite/workers/server"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpServer"
	"net"
)

func NewUdptlsSctpServer(localAddress string, password string, ctx context.Context) *UdptlsSctpServer {
	utss := &UdptlsSctpServer{}
	utss.Address = localAddress
	utss.password = []byte(password)
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
}

func (s UdptlsSctpServer) Connection(conn net.Conn) {
	s.Process(conn)
}

func (s *UdptlsSctpServer) Process(conn net.Conn) {

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
	}(s.ctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-S_S2CDataTraffic:
				S_S2CDataTraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(s.ctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-S_C2STraffic2:
				S_C2STraffic <- server.UDPServerRxFromClientTraffic(data)
			case <-ctx.Done():
				return
			}
		}

	}(s.ctx)

	relay := udpsctpserver.NewPacketRelayServer(conn, S_S2CTraffic2, S_S2CDataTraffic2, S_C2STraffic2, &ts, s.password, s.ctx)

	_ = relay

	udpserver := server.UDPServer(s.ctx, S_S2CTraffic, S_S2CDataTraffic, S_C2STraffic)

	_ = udpserver

	relay.RateLimitTcpServerWrite(s.ratelimitServerTCPWriteBytePerSecond, s.ratelimitServerTCPWriteMaxBucketSize, s.ratelimitServerTCPWriteInitialSize)
}
func (s *UdptlsSctpServer) Up() {
	//Open Connection
	var v = udpServer.NewUDPServer(s.Address, s.ctx, s)
	s.udplistener = v
}

func (s *UdptlsSctpServer) RateLimitTcpServerWrite(ratelimitServerTCPWriteBytePerSecond int,
	ratelimitServerTCPWriteMaxBucketSize int,
	ratelimitServerTCPWriteInitialSize int) {
	s.ratelimitServerTCPWriteBytePerSecond = ratelimitServerTCPWriteBytePerSecond
	s.ratelimitServerTCPWriteMaxBucketSize = ratelimitServerTCPWriteMaxBucketSize
	s.ratelimitServerTCPWriteInitialSize = ratelimitServerTCPWriteInitialSize
}
