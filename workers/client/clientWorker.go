package client

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport/transportQuality"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func UDPClient(context context.Context,
	TxToServer chan UDPClientTxToServerTraffic,
	TxToServerData chan UDPClientTxToServerDataTraffic,
	RxFromServer chan UDPClientRxFromServerTraffic,
	LocalTxToTun chan interfaces.UDPPacket,
	LocalRxFromTun chan interfaces.UDPPacket,
	GetTransmitLayerSentRecvStatsInt interfaces.GetTransmitLayerSentRecvStats) *UDPClientContext {
	ucc := &UDPClientContext{
		TrackedChannel:                   sync.Map{},
		TrackedAddr:                      sync.Map{},
		TxToServer:                       TxToServer,
		TxToServerData:                   TxToServerData,
		RxFromServer:                     RxFromServer,
		context:                          context,
		LocalTxToTun:                     LocalTxToTun,
		LocalRxFromTun:                   LocalRxFromTun,
		LastPongRecv:                     time.Now(),
		QualityInt:                       transportQuality.NewQualityEstimator(),
		GetTransmitLayerSentRecvStatsInt: GetTransmitLayerSentRecvStatsInt,
	}
	go ucc.RxFromServerWorker()
	go ucc.TxToServerWorker()
	go ucc.pingRoutine()
	return ucc
}

type UDPClientContext struct {
	TrackedChannel sync.Map //Key:uint16 Channel
	TrackedAddr    sync.Map //Key:

	TxToServer     chan UDPClientTxToServerTraffic
	TxToServerData chan UDPClientTxToServerDataTraffic
	RxFromServer   chan UDPClientRxFromServerTraffic

	context context.Context

	LocalTxToTun   chan interfaces.UDPPacket
	LocalRxFromTun chan interfaces.UDPPacket

	LastPongRecv time.Time

	GetTransmitLayerSentRecvStatsInt interfaces.GetTransmitLayerSentRecvStats

	pingSeq uint64

	QualityInt interfaces.QualityEstimator
}

type UDPClientTxToServerTraffic interfaces.TrafficWithChannelTag
type UDPClientTxToServerDataTraffic interfaces.TrafficWithChannelTag
type UDPClientRxFromServerTraffic interfaces.TrafficWithChannelTag

type UDPClientTrackedAddrKey struct {
	Source net.UDPAddr
	Dest   net.UDPAddr
}

func (uctak *UDPClientTrackedAddrKey) Key() string {
	return uctak.Source.String() + uctak.Dest.String()
}

type UDPClientTrackedAddrContext struct {
	Channel uint16
}

func (ucc *UDPClientContext) pingRoutine() {
	for {
		select {
		case <-ucc.context.Done():
			return
		case timenow := <-time.NewTimer(time.Second / 2).C:
			shouldPingBeSend := false
			if timenow.Sub(ucc.LastPongRecv).Seconds() > 5 {
				shouldPingBeSend = true
			}
			t := timenow.Sub(ucc.LastPongRecv).Seconds()
			if t > 10 {
				fmt.Printf("No pong were received in last %v second\n", t)
			}

			if t > 180 {
				os.Exit(0)
			}

			if shouldPingBeSend {
				ucc.sendPing()
			}
		}
	}
}

func (ucc *UDPClientContext) sendPing() {
	var buf bytes.Buffer

	Header := &proto.CommandHeader{CommandByte: proto.CommandByte_Ping}

	err := struc.Pack(&buf, Header)

	if err != nil {
		println(err)
	}

	ucc.pingSeq += 1

	PingHeader := &proto.PingHeader{}
	PingHeader.Seq = ucc.pingSeq
	PingHeader.Seq2 = uint64(time.Now().UnixNano())

	if ucc.GetTransmitLayerSentRecvStatsInt != nil {
		sent, recv := ucc.GetTransmitLayerSentRecvStatsInt.GetTransmitLayerSentRecvStats()
		PingHeader.SentPacket = sent
		PingHeader.RecvPacket = recv
	}

	err2 := struc.Pack(&buf, PingHeader)

	if err2 != nil {
		println(err2)
	}

	ucc.QualityInt.OnSendPing(*PingHeader)

	ucc.TxToServer <- UDPClientTxToServerTraffic{Channel: 0, Payload: buf.Bytes()}
}

func (ucc *UDPClientContext) RxFromServerWorker() {
	var err error
	_ = err
	for {
		select {
		case traffic := <-ucc.RxFromServer:
			if traffic.Channel == 0 {
				//Decode this as a control packet
				payloadData := bytes.NewReader(traffic.Payload)
				ch := &proto.CommandHeader{}
				err = struc.Unpack(payloadData, ch)
				if err != nil {
					log.Println(err)
					continue
				}
				switch ch.CommandByte {
				case proto.CommandByte_Send:
					ucc.rxFromServerWorker_OnControlSend(payloadData)
					break
				case proto.CommandByte_Associate:
					ucc.rxFromServerWorker_OnControlAssociate(payloadData)
					break
				case proto.CommandByte_ChannelDestroy:
					ucc.rxFromServerWorker_OnControlChannelDestroy(payloadData)
					break
				case proto.CommandByte_Pong:
					ucc.LastPongRecv = time.Now()
					ucc.rxFromServerWorker_OnControlPong(payloadData)
					break
				}
			} else {
				ucc.rxFromServerWorker_Data(traffic.Payload, traffic.Channel)
			}
		case <-ucc.context.Done():
			return
		}
	}
}

func (ucc *UDPClientContext) rxFromServerWorker_OnControlPong(reader io.Reader) {
	pongHeader := &proto.PongHeader{}

	err := struc.Unpack(reader, pongHeader)

	if err != nil {
		log.Println(err)
	}

	//We send this into insight

	ucc.QualityInt.OnReceivePong(*pongHeader)
}

func (ucc *UDPClientContext) rxFromServerWorker_OnControlSend(reader io.Reader) {
	var err error
	_ = err
	sendHeader := &proto.SendHeader{}
	err = struc.Unpack(reader, sendHeader)
	if err != nil {
		log.Println(err)
	}

	sourceaddr := &net.UDPAddr{IP: proto.IPv4ByteToAddr(sendHeader.SourceIP), Port: int(sendHeader.SourcePort)}
	destaddr := &net.UDPAddr{IP: proto.IPv4ByteToAddr(sendHeader.DestIP), Port: int(sendHeader.DestPort)}
	udpPacket := interfaces.UDPPacket{
		Source:  sourceaddr,
		Dest:    destaddr,
		Payload: sendHeader.Payload,
	}

	ucc.LocalTxToTun <- udpPacket
}
func (ucc *UDPClientContext) rxFromServerWorker_OnControlAssociate(reader io.Reader) {
	var err error
	_ = err
	associateHeader := &proto.AssociateHeader{}
	err = struc.Unpack(reader, associateHeader)
	if err != nil {
		log.Println(err)
	}
	sourceaddr := &net.UDPAddr{IP: proto.IPv4ByteToAddr(associateHeader.SourceIP), Port: int(associateHeader.SourcePort)}
	destaddr := &net.UDPAddr{IP: proto.IPv4ByteToAddr(associateHeader.DestIP), Port: int(associateHeader.DestPort)}

	key := UDPClientTrackedAddrKey{Source: *sourceaddr, Dest: *destaddr}
	value := &UDPClientTrackedAddrContext{Channel: associateHeader.Channel}
	ucc.TrackedAddr.Store(key.Key(), value)
	ucc.TrackedChannel.Store(associateHeader.Channel, key)

	ret := proto.AssociateDoneHeader{}
	ret = proto.AssociateDoneHeader(*associateHeader)

	reth := &proto.CommandHeader{CommandByte: proto.CommandByte_AssociateDone}

	retbuf := bytes.NewBuffer(nil)

	err = struc.Pack(retbuf, reth)
	if err != nil {
		log.Println(err)
	}

	err = struc.Pack(retbuf, &ret)
	if err != nil {
		log.Println(err)
	}

	retpack := UDPClientTxToServerTraffic{Channel: 0, Payload: retbuf.Bytes()}

	ucc.TxToServer <- retpack
}
func (ucc *UDPClientContext) rxFromServerWorker_OnControlChannelDestroy(reader io.Reader) {
	var err error
	_ = err
	channelDestoryHeader := &proto.AssociateChannelDestroy{}
	err = struc.Unpack(reader, channelDestoryHeader)
	if err != nil {
		log.Println(err)
	}
	ChannelI, ok := ucc.TrackedChannel.Load(channelDestoryHeader.Channel)
	if !ok {
		return
	}

	ChannelD := ChannelI.(UDPClientTrackedAddrKey)

	ucc.TrackedAddr.Delete(ChannelD.Key())

	ucc.TrackedChannel.Delete(channelDestoryHeader.Channel)
}
func (ucc *UDPClientContext) rxFromServerWorker_Data(p []byte, channel uint16) {
	ChannelI, ok := ucc.TrackedChannel.Load(channel)
	if !ok {
		return
	}

	ChannelD := ChannelI.(UDPClientTrackedAddrKey)

	udpPacket := interfaces.UDPPacket{
		Source:  &ChannelD.Dest,
		Dest:    &ChannelD.Source,
		Payload: p,
	}

	ucc.LocalTxToTun <- udpPacket

}

func (ucc *UDPClientContext) txToServerWorker(pack *interfaces.UDPPacket) {
	var err error
	_ = err

	key := UDPClientTrackedAddrKey{Source: *pack.Source, Dest: *pack.Dest}
	chann, ok := ucc.TrackedAddr.Load(key.Key())
	if ok {
		channelID := chann.(*UDPClientTrackedAddrContext)
		ucc.TxToServerData <- UDPClientTxToServerDataTraffic{Channel: channelID.Channel, Payload: pack.Payload}
	} else {
		sendingBuf := &bytes.Buffer{}
		sendH := &proto.CommandHeader{CommandByte: proto.CommandByte_Send}

		send := &proto.SendHeader{
			SourceIP:   proto.IPv4AddrToByte(pack.Source.IP),
			DestIP:     proto.IPv4AddrToByte(pack.Dest.IP),
			SourcePort: uint16(pack.Source.Port),
			DestPort:   uint16(pack.Dest.Port),
			PayloadLen: uint16(len(pack.Payload)),
			Payload:    pack.Payload,
		}

		err = struc.Pack(sendingBuf, sendH)
		if err != nil {
			log.Println(err)
		}

		err = struc.Pack(sendingBuf, send)
		if err != nil {
			log.Println(err)
		}

		ucc.TxToServer <- UDPClientTxToServerTraffic{Channel: 0, Payload: sendingBuf.Bytes()}
	}
}

func (ucc *UDPClientContext) TxToServerWorker() {
	for {
		select {
		case <-ucc.context.Done():
			return
		case pack := <-ucc.LocalRxFromTun:
			ucc.txToServerWorker(&pack)
		}
	}
}
