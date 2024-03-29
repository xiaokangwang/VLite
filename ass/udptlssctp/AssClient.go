package udptlssctp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/clientInbound/stackstub"
	"github.com/xiaokangwang/VLite/clientInbound/tun"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/httpClient"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"github.com/xiaokangwang/VLite/transport/packetuni/puniClient"
	"github.com/xiaokangwang/VLite/transport/udp/udpClient"
	"github.com/xiaokangwang/VLite/transport/udp/udpuni/udpunic"
	"github.com/xiaokangwang/VLite/transport/uni/uniclient"
	client2 "github.com/xiaokangwang/VLite/workers/client"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpClient"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"runtime/debug"
	"strings"
	"time"
)

func NewUdptlsSctpClient(remoteAddress string, password string, tunName string, ctx context.Context) *UdptlsSctpClient {
	utsc := &UdptlsSctpClient{}
	utsc.Address = remoteAddress
	utsc.password = []byte(password)
	utsc.tunName = tunName
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
		utsc.ctx = context.WithValue(utsc.ctx, interfaces.ExtraOptionsUDPMask, string(utsc.password))
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

type UdptlsSctpClient struct {
	Address string
	ctx     context.Context

	udpdialer transport.UnderlayTransportDialer
	udprelay  *udpsctpserver.PacketSCTPRelay
	udpserver *client2.UDPClientContext
	//st        *stack.NetstackHolder

	password []byte
	msgbus   *bus.Bus

	uni  *uniclient.UnifiedConnectionClient
	puni *puniClient.PacketUniClient

	httpac *httpClient.ProviderClient

	tunName string
}
type UdptlsSctpClientStramToNetConnAdp struct {
	rwc io.ReadWriteCloser
}

func (u UdptlsSctpClientStramToNetConnAdp) Read(b []byte) (n int, err error) {
	/*
		n2, err2 := u.rwc.Read(b)
		if err2 != nil {
			fmt.Println(err2)
			return 0, io.EOF
		}
		return n2, nil*/

	return u.rwc.Read(b)
}

func (u UdptlsSctpClientStramToNetConnAdp) Write(b []byte) (n int, err error) {

	/*
		n2, err2 := u.rwc.Write(b)
		if err2 != nil {
			fmt.Println(err2)
			return n2, io.EOF
		}

		return n2, nil*/

	return u.rwc.Write(b)
}

func (u UdptlsSctpClientStramToNetConnAdp) Close() error {
	return u.rwc.Close()
}

func (u UdptlsSctpClientStramToNetConnAdp) LocalAddr() net.Addr {
	panic("implement me")
}

func (u UdptlsSctpClientStramToNetConnAdp) RemoteAddr() net.Addr {
	panic("implement me")
}

func (u UdptlsSctpClientStramToNetConnAdp) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (u UdptlsSctpClientStramToNetConnAdp) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (u UdptlsSctpClientStramToNetConnAdp) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

const OUTCHANNEL = false

func (s *UdptlsSctpClient) Dial(network, address string, port uint16, ctx context.Context) (net.Conn, error) {

	if OUTCHANNEL {
		return proxy.FromEnvironment().Dial(network, fmt.Sprintf("%v:%v", address, port))
	}

	return s.DialDirect(address, port)
}

func (s *UdptlsSctpClient) DialDirect(address string, port uint16) (net.Conn, error) {
	var Stream io.ReadWriteCloser
	if s.httpac != nil {
		Stream = s.httpac.DialWsAlternativeChannelConnection(s.ctx)
	} else if s.uni != nil && UsePuni {
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

func (s *UdptlsSctpClient) NotifyMeltdown(reason error) {
	panic("implement me")
}

func (s *UdptlsSctpClient) Up() {
	//Open Connection
	conn, err, connctx := s.udpdialer.Connect(s.ctx)
	if err != nil {
		log.Println(err)
		debug.PrintStack()
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
				//spew.Dump(data)
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
				//spew.Dump(data)
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
				//spew.Dump(data)
				C_S2CTraffic <- client2.UDPClientRxFromServerTraffic(data)
			case <-ctx.Done():
				return
			}
		}

	}(connctx)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	waterw := tun.NewTun(s.tunName)

	tunudplink, err := tun.NewTunToUDPLink(TunnelTxToTun, TunnelRxFromTun, *waterw)
	if err != nil {
		log.Println(err)
		debug.PrintStack()
	}

	stack2 := &stackstub.StackStub{}

	stack2.HostWR(tunudplink)

	//s.st = stack2

	if s.uni != nil && UsePuni {
		s.puni = puniClient.NewPacketUniClient(C_C2STraffic2, C_C2SDataTraffic2, C_S2CTraffic2, s.password, connctx)
		s.puni.OnAutoCarrier(conn, connctx)
		s.udpserver = client2.UDPClient(connctx, C_C2STraffic, C_C2SDataTraffic, C_S2CTraffic, TunnelTxToTun, TunnelRxFromTun, s.puni)
	} else {
		s.udprelay = udpsctpserver.NewPacketRelayClient(conn, C_C2STraffic2, C_C2SDataTraffic2, C_S2CTraffic2, s.password, connctx)
		s.udpserver = client2.UDPClient(connctx, C_C2STraffic, C_C2SDataTraffic, C_S2CTraffic, TunnelTxToTun, TunnelRxFromTun, s.udprelay)
	}
}
func (s *UdptlsSctpClient) AlternativeChannel(url string) {
	s.httpac = httpClient.NewProviderClientCreator(url, 0, 0, string(s.password), s.ctx).ConnectL(s.ctx)
}

type TCPSocketDialer interface {
	DialDirect(address string, port uint16) (net.Conn, error)
}
