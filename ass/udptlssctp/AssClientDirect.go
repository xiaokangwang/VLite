package udptlssctp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/httpClient"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"github.com/xiaokangwang/VLite/transport/packetuni/puniClient"
	"github.com/xiaokangwang/VLite/transport/packetuni/puniCommon"
	"github.com/xiaokangwang/VLite/transport/udp/udpClient"
	"github.com/xiaokangwang/VLite/transport/udp/udpuni/udpunic"
	"github.com/xiaokangwang/VLite/transport/uni/uniclient"
	client2 "github.com/xiaokangwang/VLite/workers/client"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpClient"
	"io"
	"log"
	"net"
	"strings"
)

func NewUdptlsSctpClientDirect(remoteAddress string, password string, ctx context.Context) *UdptlsSctpClientDirect {
	utsc := &UdptlsSctpClientDirect{}
	utsc.Address = remoteAddress
	utsc.password = []byte(password)
	utsc.msgbus = ibus.NewMessageBus()

	ctxwbus := context.WithValue(ctx, interfaces.ExtraOptionsMessageBus, utsc.msgbus)

	utsc.ctx = ctxwbus

	useUniConn := false

	if strings.HasPrefix(remoteAddress, "uni+") {
		useUniConn = true
		remoteAddress = remoteAddress[4:]

	}

	useWs := false

	if strings.HasPrefix(remoteAddress, "ws+") {
		useWs = true
		remoteAddress = remoteAddress[3:]
	}

	if strings.HasPrefix(remoteAddress, "http") {
		if useWs {
			utsc.ctx = context.WithValue(utsc.ctx, interfaces.ExtraOptionsUseWebSocketInsteadOfHTTP, useWs)
		}
		utsc.udpdialer = httpClient.NewProviderClientCreator(remoteAddress, 2, 2, password, utsc.ctx)
	} else {
		if strings.HasPrefix(remoteAddress, "fec+") {
			remoteAddress = remoteAddress[4:]
			utsc.ctx = context.WithValue(utsc.ctx, interfaces.ExtraOptionsUDPFECEnabled, true)
		}
		utsc.udpdialer = udpClient.NewUdpClient(remoteAddress, utsc.ctx)
		if useUniConn {
			utsc.udpdialer = udpunic.NewUdpUniClient(string(utsc.password), utsc.ctx, utsc.udpdialer)
		}
	}
	if useUniConn {
		unis := uniclient.NewUnifiedConnectionClient(utsc.udpdialer, utsc.ctx)
		utsc.udpdialer = unis
		utsc.uni = unis
	}
	return utsc
}

type UdptlsSctpClientDirect struct {
	Address string
	ctx     context.Context

	udpdialer transport.UnderlayTransportDialer
	udprelay  *udpsctpserver.PacketSCTPRelay
	udpserver *client2.UDPClientContext

	uni  *uniclient.UnifiedConnectionClient
	puni *puniClient.PacketUniClient

	password []byte

	TunnelTxToTun   chan interfaces.UDPPacket
	TunnelRxFromTun chan interfaces.UDPPacket

	msgbus *bus.Bus

	connCtx context.Context
}

func (s *UdptlsSctpClientDirect) Dial(network, address string, port uint16, ctx context.Context) (net.Conn, error) {
	return s.DialDirect(address, port)
}
func (s *UdptlsSctpClientDirect) DialDirect(address string, port uint16) (net.Conn, error) {
	var Stream io.ReadWriteCloser
	if s.uni != nil {
		Stream = s.puni.ClientOpenStream()
	} else {
		Stream = s.udprelay.ClientOpenStream()
	}

	var w = &bytes.Buffer{}
	tcpClient.WriteTcpDialHeader(w, address, port)

	lw := &bytes.Buffer{}

	lenb := &proto.StreamConnectDomainHeaderLen{}

	lenb.Length = uint16(w.Len())

	struc.Pack(lw, lenb)

	_, _ = Stream.Write(lw.Bytes())

	len2, err := Stream.Write(w.Bytes())
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if len2 != w.Len() {
		fmt.Println("cannot write exact length")
		return nil, errors.New("cannot write exact length")
	}
	//fmt.Println("Connected")
	return &UdptlsSctpClientStramToNetConnAdp{rwc: Stream}, nil
}

func (s *UdptlsSctpClientDirect) NotifyMeltdown(reason error) {
	panic("implement me")
}

func (s *UdptlsSctpClientDirect) Up() {
	//Open Connection
	conn, err, connctx := s.udpdialer.Connect(s.ctx)
	if err != nil {
		log.Println(err)
	}

	C_C2STraffic := make(chan client2.UDPClientTxToServerTraffic, 8)
	C_C2SDataTraffic := make(chan client2.UDPClientTxToServerDataTraffic, 8)
	C_S2CTraffic := make(chan client2.UDPClientRxFromServerTraffic, 8)

	C_C2STraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)
	C_C2SDataTraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)
	C_S2CTraffic2 := make(chan interfaces.TrafficWithChannelTag, 8)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-C_C2STraffic:
				C_C2STraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-C_C2SDataTraffic:
				C_C2SDataTraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-C_S2CTraffic2:
				C_S2CTraffic <- client2.UDPClientRxFromServerTraffic(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	s.TunnelTxToTun = TunnelTxToTun
	s.TunnelRxFromTun = TunnelRxFromTun
	if s.uni != nil && UsePuni {
		s.puni = puniClient.NewPacketUniClient(C_C2STraffic2, C_C2SDataTraffic2, C_S2CTraffic2, s.password, connctx)
		s.puni.OnAutoCarrier(conn, connctx)
		s.udpserver = client2.UDPClient(connctx, C_C2STraffic, C_C2SDataTraffic, C_S2CTraffic, TunnelTxToTun, TunnelRxFromTun, s.puni)
	} else {
		s.udprelay = udpsctpserver.NewPacketRelayClient(conn, C_C2STraffic2, C_C2SDataTraffic2, C_S2CTraffic2, s.password, connctx)
		s.udpserver = client2.UDPClient(connctx, C_C2STraffic, C_C2SDataTraffic, C_S2CTraffic, TunnelTxToTun, TunnelRxFromTun, s.udprelay)
	}

	s.connCtx = connctx
}

func (s *UdptlsSctpClientDirect) Reconnect() {
	if UsePuni {
		puniCommon.ReHandshake(s.connCtx)
	} else {
		s.uni.ReconnectUnder(s.connCtx)
	}

}
