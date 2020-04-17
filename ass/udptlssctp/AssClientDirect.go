package udptlssctp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"github.com/xiaokangwang/VLite/transport/udp/udpClient"
	client2 "github.com/xiaokangwang/VLite/workers/client"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpClient"
	"log"
	"net"
)

func NewUdptlsSctpClientDirect(remoteAddress string, password string, ctx context.Context) *UdptlsSctpClientDirect {
	utsc := &UdptlsSctpClientDirect{}
	utsc.Address = remoteAddress
	utsc.password = []byte(password)
	utsc.ctx = ctx
	utsc.udpdialer = udpClient.NewUdpClient(remoteAddress)
	return utsc
}

type UdptlsSctpClientDirect struct {
	Address string
	ctx     context.Context

	udpdialer transport.UnderlayTransportDialer
	udprelay  *udpsctpserver.PacketSCTPRelay
	udpserver *client2.UDPClientContext

	password []byte

	TunnelTxToTun   chan interfaces.UDPPacket
	TunnelRxFromTun chan interfaces.UDPPacket
}

func (s *UdptlsSctpClientDirect) Dial(network, address string, port uint16, ctx context.Context) (net.Conn, error) {
	return s.DialDirect(address, port)
}
func (s *UdptlsSctpClientDirect) DialDirect(address string, port uint16) (net.Conn, error) {
	Stream := s.udprelay.ClientOpenStream()
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
	conn, err := s.udpdialer.Connect()
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

	}(s.ctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-C_C2SDataTraffic:
				C_C2SDataTraffic2 <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(s.ctx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-C_S2CTraffic2:
				C_S2CTraffic <- client2.UDPClientRxFromServerTraffic(data)
			case <-ctx.Done():
				return
			}
		}

	}(s.ctx)

	TunnelTxToTun := make(chan interfaces.UDPPacket)
	TunnelRxFromTun := make(chan interfaces.UDPPacket)

	s.TunnelTxToTun = TunnelTxToTun
	s.TunnelRxFromTun = TunnelRxFromTun

	s.udprelay = udpsctpserver.NewPacketRelayClient(conn, C_C2STraffic2, C_C2SDataTraffic2, C_S2CTraffic2, s.password, s.ctx)
	s.udpserver = client2.UDPClient(s.ctx, C_C2STraffic, C_C2SDataTraffic, C_S2CTraffic, TunnelTxToTun, TunnelRxFromTun)
}
