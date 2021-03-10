package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/workers"
	"github.com/xiaokangwang/VLite/workers/channelNumberFinder"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func UDPServer(context context.Context,
	TxToClient chan UDPServerTxToClientTraffic, TxToClientData chan UDPServerTxToClientDataTraffic,
	RxFromClient chan UDPServerRxFromClientTraffic,
	GetTransmitLayerSentRecvStatsInt interfaces.GetTransmitLayerSentRecvStats) *UDPServerContext {

	usc := &UDPServerContext{
		TrackedRemoteAddr:                sync.Map{},
		ClientLogicConnection:            sync.Map{},
		TxToClient:                       TxToClient,
		TxToClientData:                   TxToClientData,
		RxFromClient:                     RxFromClient,
		context:                          context,
		GetTransmitLayerSentRecvStatsInt: GetTransmitLayerSentRecvStatsInt,
	}

	usc.ChannelNumberGenerator = channelNumberFinder.NewChannelNumberFinder(0, 0, usc)

	usc.opts = UDPServerContext_Opts{UDPTimeoutTime: workers.UDPReadingTimeout}

	ExtraOptionsUDPTimeoutTime := context.Value(interfaces.ExtraOptionsUDPTimeoutTime)

	if ExtraOptionsUDPTimeoutTime != nil {
		usc.opts.UDPTimeoutTime = ExtraOptionsUDPTimeoutTime.(*interfaces.ExtraOptionsUDPTimeoutTimeValue).TimeoutTimeInSeconds
	}

	go usc.RxFromClientWorker()
	return usc
}

type UDPServerContext struct {
	TrackedRemoteAddr      sync.Map //Key:uint16 Channel
	ClientLogicConnection  sync.Map //Key:UDPAddr Client
	TxToClient             chan UDPServerTxToClientTraffic
	TxToClientData         chan UDPServerTxToClientDataTraffic
	RxFromClient           chan UDPServerRxFromClientTraffic
	ChannelNumberGenerator *channelNumberFinder.ChannelNumberFinder

	opts UDPServerContext_Opts

	context context.Context

	GetTransmitLayerSentRecvStatsInt interfaces.GetTransmitLayerSentRecvStats
}

type UDPServerContext_Opts struct {
	UDPTimeoutTime int
}

func (uscc *UDPServerContext) IsChannelUsed(u uint16) bool {
	_, ok := uscc.TrackedRemoteAddr.Load(u)
	return ok
}

type UDPServerTrackedRemoteAddrContext struct {
	RemoteAddr net.UDPAddr
	LocalAddr  net.UDPAddr
}

type UDPServerClientLogicConnectionContext struct {
	LocalAddr net.UDPAddr

	TrackedConnection sync.Map //Key:UDPAddr Remote

	lock sync.Mutex
	Conn net.Conn

	//Sending the first data packet and receiving data will refresh this
	LastSeen time.Time

	NoWait bool
}

type UDPTrackedConnectionContext struct {
	Tracking     bool
	TrackPending bool
	Tracked      bool
	Channel      uint16
}

type UDPServerTxToClientTraffic interfaces.TrafficWithChannelTag
type UDPServerTxToClientDataTraffic interfaces.TrafficWithChannelTag

type UDPServerRxFromClientTraffic interfaces.TrafficWithChannelTag

//Must in new goroutine, ALWAYS block!
func (uscc *UDPServerContext) RxFromClientWorker() {
	var err error
	_ = err
	for {
		select {
		case traffic := <-uscc.RxFromClient:
			//First check if it is send through control channel
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
					uscc.rxFromClientWorker_OnControlSend(payloadData)
					break
				case proto.CommandByte_SendV6:
					uscc.rxFromClientWorker_OnControlSendV6(payloadData)
					break
				case proto.CommandByte_AssociateDone:
					uscc.rxFromClientWorker_OnControlAssociateDone(payloadData)
					break
				case proto.CommandByte_AssociateDoneV6:
					uscc.rxFromClientWorker_OnControlAssociateV6Done(payloadData)
					break
				case proto.CommandByte_Ping:
					uscc.sendPong(payloadData)
				}
			} else {
				//Decode this as a data packet
				uscc.rxFromClientWorker_OnData(traffic.Channel, traffic.Payload)
			}
			continue
		case <-uscc.context.Done():
			fmt.Println("Server quiting")
			return
		}
	}
}

func (uscc *UDPServerContext) sendPong(reader io.Reader) {
	//fmt.Println("Pong responding")
	PingHeader := &proto.PingHeader{}

	err0 := struc.Unpack(reader, PingHeader)

	if err0 != nil {
		println(err0.Error())
		return
	}

	PongHeader := &proto.PongHeader{}

	PongHeader.SeqCopy = PingHeader.Seq
	PongHeader.Seq2Copy = PingHeader.Seq2

	if uscc.GetTransmitLayerSentRecvStatsInt != nil {
		sent, recv := uscc.GetTransmitLayerSentRecvStatsInt.GetTransmitLayerSentRecvStats()
		PongHeader.SentPacket = sent
		PongHeader.RecvPacket = recv
	}

	var buf bytes.Buffer

	Header := &proto.CommandHeader{CommandByte: proto.CommandByte_Pong}

	err := struc.Pack(&buf, Header)

	if err != nil {
		println(err)
		return
	}

	err2 := struc.Pack(&buf, PongHeader)

	if err2 != nil {
		println(err2)
		return
	}

	uscc.TxToClient <- UDPServerTxToClientTraffic{Channel: 0, Payload: buf.Bytes()}
}

func (uscc *UDPServerContext) rxFromClientWorker_OnControlSend(reader io.Reader) {
	sendHeader := &proto.SendHeader{}
	var err error
	_ = err

	err = struc.Unpack(reader, sendHeader)
	if err != nil {
		log.Println(err)
	}

	sourceAddr := &net.UDPAddr{}
	sourceAddr.Port = int(sendHeader.SourcePort)
	sourceAddr.IP = proto.IPv4ByteToAddr(sendHeader.SourceIP)

	UntrackedDefault := &UDPServerClientLogicConnectionContext{LocalAddr: *sourceAddr}

	Tracker := UntrackedDefault

	actualI, loaded := uscc.ClientLogicConnection.LoadOrStore(sourceAddr.String(), UntrackedDefault)
	if loaded {
		Tracker = actualI.(*UDPServerClientLogicConnectionContext)
	}

	Tracker.lock.Lock()

	destaddr := &net.UDPAddr{IP: proto.IPv4ByteToAddr(sendHeader.DestIP), Port: int(sendHeader.DestPort)}
	uscc.trackConnection(*sourceAddr, *destaddr)

	//Is there a socket ready? Create one if there is none

	if Tracker.Conn == nil {
		dialCtx := context.WithValue(uscc.context,
			interfaces.ExtraOptionsLocalUDPBindPort,
			interfaces.ExtraOptionsLocalUDPBindPortValue{LocalPort: uint16(sourceAddr.Port)})
		conn, err := workers.Dialer.Dial("udp",
			proto.IPv4ByteToAddr(sendHeader.DestIP).String(),
			sendHeader.DestPort, dialCtx)
		if err != nil {
			log.Println(err)
			return
		}
		Tracker.Conn = conn
		if destaddr.Port == 53 {
			Tracker.NoWait = true
		}
		go uscc.rxFromRemoteListener(Tracker)
	}

	//OK, we have got a conn and now its time to send the payload
	UDPConn := Tracker.Conn.(net.PacketConn)

	_, err = UDPConn.WriteTo(sendHeader.Payload,
		&net.UDPAddr{Port: int(sendHeader.DestPort), IP: proto.IPv4ByteToAddr(sendHeader.DestIP)})
	if err != nil {
		log.Println(err)
	}
	Tracker.lock.Unlock()

}
func (uscc *UDPServerContext) rxFromClientWorker_OnControlSendV6(reader io.Reader) {
	sendHeader := &proto.SendV6Header{}
	var err error
	_ = err

	err = struc.Unpack(reader, sendHeader)
	if err != nil {
		log.Println(err)
	}

	sourceAddr := &net.UDPAddr{}
	sourceAddr.Port = int(sendHeader.SourcePort)
	sourceAddr.IP = proto.IPv6ByteToAddr(sendHeader.SourceIP)

	UntrackedDefault := &UDPServerClientLogicConnectionContext{LocalAddr: *sourceAddr}

	Tracker := UntrackedDefault

	actualI, loaded := uscc.ClientLogicConnection.LoadOrStore(sourceAddr.String(), UntrackedDefault)
	if loaded {
		Tracker = actualI.(*UDPServerClientLogicConnectionContext)
	}

	Tracker.lock.Lock()

	destaddr := &net.UDPAddr{IP: proto.IPv6ByteToAddr(sendHeader.DestIP), Port: int(sendHeader.DestPort)}
	uscc.trackConnection(*sourceAddr, *destaddr)

	//Is there a socket ready? Create one if there is none

	if Tracker.Conn == nil {
		dialCtx := context.WithValue(uscc.context,
			interfaces.ExtraOptionsLocalUDPBindPort,
			interfaces.ExtraOptionsLocalUDPBindPortValue{LocalPort: uint16(sourceAddr.Port)})
		conn, err := workers.Dialer.Dial("udp",
			proto.IPv6ByteToAddr(sendHeader.DestIP).String(),
			sendHeader.DestPort, dialCtx)
		if err != nil {
			log.Println(err)
			return
		}
		Tracker.Conn = conn
		if destaddr.Port == 53 {
			Tracker.NoWait = true
		}
		go uscc.rxFromRemoteListener(Tracker)
	}

	//OK, we have got a conn and now its time to send the payload
	UDPConn := Tracker.Conn.(net.PacketConn)

	_, err = UDPConn.WriteTo(sendHeader.Payload,
		&net.UDPAddr{Port: int(sendHeader.DestPort), IP: proto.IPv6ByteToAddr(sendHeader.DestIP)})
	if err != nil {
		log.Println(err)
	}
	Tracker.lock.Unlock()

}

//Must in new goroutine, ALWAYS block!
func (uscc *UDPServerContext) rxFromRemoteListener(tracker *UDPServerClientLogicConnectionContext) {
	var buffer [1600]byte

	var err error
	_ = err

	UDPConn := tracker.Conn.(net.PacketConn)

	shutDown := func() {
		tracker.lock.Lock()
		tracker.Conn = nil
		err = UDPConn.Close()
		if err != nil {
			log.Println(err)
		}
		go uscc.untrackAll(tracker)
		tracker.lock.Unlock()
	}

	tracker.LastSeen = time.Now()

	initialTrackTime := 600
	overallTrackTime := uscc.opts.UDPTimeoutTime
	if tracker.NoWait {
		initialTrackTime = 5
		overallTrackTime = 30
	}
	for {

		err = UDPConn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(initialTrackTime)))

		if err != nil {
			log.Println(err)
			shutDown()
			return
		}

		n, addr, err2 := UDPConn.ReadFrom(buffer[:])

		if err2 != nil {
			if err3, ok := err2.(net.Error); ok && (err3.Timeout() || err3.Temporary()) {
				//Try again is fine

				//See if we have not wait long enough to stop looking for packet
				if tracker.LastSeen.Add(time.Second*time.Duration(overallTrackTime)).Sub(time.Now()) < 0 {
					//It have been a while, close this listener
					log.Println("Listener Closed for inactivity")
					shutDown()
					return
				}

				// If we are not cancelled yet, read again
				if uscc.context.Err() == nil {
					continue
				}

			}
			log.Println(err)
			shutDown()
			return
		}

		// We've got a incoming packet, now we should relay it

		//First, refresh the LastSeen
		tracker.LastSeen = time.Now()

		//Then actually relay the packet
		uscc.txToClient(buffer[:n], tracker, *(addr.(*net.UDPAddr)))

	}
}

//Callee don't own payload, only copy exists after return
func (uscc *UDPServerContext) txToClient(payload []byte, tracker *UDPServerClientLogicConnectionContext, from net.UDPAddr) {
	var err error
	_ = err

	copyOfPayload := make([]byte, len(payload))
	copy(copyOfPayload, payload)

	//ask to track the connection if not yet

	tracked, chann := uscc.isConnectionTracked(tracker.LocalAddr, from, true)
	if tracked {
		uscc.TxToClientData <- UDPServerTxToClientDataTraffic{Channel: chann, Payload: copyOfPayload}
	} else {
		sendingBuf := bytes.NewBuffer(nil)

		command := proto.CommandHeader{CommandByte: proto.CommandByte_Send}

		if from.IP.To4() != nil {
			send := proto.SendHeader{
				SourceIP:   proto.IPv4AddrToByte(from.IP.To4()),
				DestIP:     proto.IPv4AddrToByte(tracker.LocalAddr.IP.To4()),
				SourcePort: uint16(from.Port),
				DestPort:   uint16(tracker.LocalAddr.Port),
				PayloadLen: uint16(len(payload)),
				Payload:    payload,
			}
			err = struc.Pack(sendingBuf, &command)
			if err != nil {
				log.Println(err)
			}

			err = struc.Pack(sendingBuf, &send)
			if err != nil {
				log.Println(err)
			}

			uscc.TxToClient <- UDPServerTxToClientTraffic{Channel: 0, Payload: sendingBuf.Bytes()}
		} else {
			command.CommandByte = proto.CommandByte_SendV6
			send := proto.SendV6Header{
				SourceIP:   proto.IPv6AddrToByte(from.IP.To16()),
				DestIP:     proto.IPv6AddrToByte(tracker.LocalAddr.IP.To16()),
				SourcePort: uint16(from.Port),
				DestPort:   uint16(tracker.LocalAddr.Port),
				PayloadLen: uint16(len(payload)),
				Payload:    payload,
			}
			err = struc.Pack(sendingBuf, &command)
			if err != nil {
				log.Println(err)
			}

			err = struc.Pack(sendingBuf, &send)
			if err != nil {
				log.Println(err)
			}

			uscc.TxToClient <- UDPServerTxToClientTraffic{Channel: 0, Payload: sendingBuf.Bytes()}
		}

	}
}

//Must non-block, source always represent a address at client's network space
func (uscc *UDPServerContext) trackConnection(source net.UDPAddr, dest net.UDPAddr) {
	uscc.isConnectionTracked(source, dest, true)
}

//Must non-block, source always represent a address at client's network space
func (uscc *UDPServerContext) trackConnectionAnnounceIfPossible(source net.UDPAddr, dest net.UDPAddr, channel uint16) bool {
	var err error
	_ = err

	if source.IP.To4() != nil && dest.IP.To4() != nil {
		s := &proto.AssociateHeader{}

		s.Channel = channel

		s.SourceIP = proto.IPv4AddrToByte(source.IP)
		s.SourcePort = uint16(source.Port)

		s.DestIP = proto.IPv4AddrToByte(dest.IP)
		s.DestPort = uint16(dest.Port)

		header := proto.CommandHeader{CommandByte: proto.CommandByte_Associate}

		buf := bytes.NewBuffer(nil)
		err = struc.Pack(buf, header)
		if err != nil {
			log.Println(err)
			return false
		}
		err = struc.Pack(buf, s)
		if err != nil {
			log.Println(err)
			return false
		}

		ToClientTxTraffic := UDPServerTxToClientTraffic{}
		ToClientTxTraffic.Channel = 0x00
		ToClientTxTraffic.Payload = buf.Bytes()

		select {
		case uscc.TxToClient <- ToClientTxTraffic:
			return true
		default:
			log.Println("Connection Track Request Discarded As It Would Block")
			return false
		}
	} else {
		s := &proto.AssociateV6Header{}

		s.Channel = channel

		s.SourceIP = proto.IPv6AddrToByte(source.IP)
		s.SourcePort = uint16(source.Port)

		s.DestIP = proto.IPv6AddrToByte(dest.IP)
		s.DestPort = uint16(dest.Port)

		header := proto.CommandHeader{CommandByte: proto.CommandByte_AssociateV6}

		buf := bytes.NewBuffer(nil)
		err = struc.Pack(buf, header)
		if err != nil {
			log.Println(err)
			return false
		}
		err = struc.Pack(buf, s)
		if err != nil {
			log.Println(err)
			return false
		}

		ToClientTxTraffic := UDPServerTxToClientTraffic{}
		ToClientTxTraffic.Channel = 0x00
		ToClientTxTraffic.Payload = buf.Bytes()

		select {
		case uscc.TxToClient <- ToClientTxTraffic:
			return true
		default:
			log.Println("Connection Track Request Discarded As It Would Block")
			return false
		}
	}

}

//Must non-block, source always represent a address at client's network space
func (uscc *UDPServerContext) isConnectionTracked(source net.UDPAddr, dest net.UDPAddr,
	TrackIfNotCurrentlyTrackingOrTracked bool) (bool, uint16) {
	trackerI, ok := uscc.ClientLogicConnection.Load(source.String())
	if !ok {
		return false, 0
	}
	tracker := trackerI.(*UDPServerClientLogicConnectionContext)

	trackerR := &UDPTrackedConnectionContext{}

	actual, found := tracker.TrackedConnection.LoadOrStore(dest.String(), trackerR)

	if found {
		trackerR = actual.(*UDPTrackedConnectionContext)
	}

	if trackerR.Tracked {
		return true, trackerR.Channel
	}

	if TrackIfNotCurrentlyTrackingOrTracked {
		if trackerR.Tracking {
			return false, trackerR.Channel
		}
		trackerR.Tracking = true
		trackingNumber := uscc.ChannelNumberGenerator.FindNext()
		trackerR.Channel = trackingNumber
		if !trackerR.TrackPending {
			tc := &UDPServerTrackedRemoteAddrContext{}
			tc.LocalAddr = source
			tc.RemoteAddr = dest

			uscc.TrackedRemoteAddr.LoadOrStore(trackerR.Channel, tc)

			trackerR.TrackPending = uscc.trackConnectionAnnounceIfPossible(source, dest, trackingNumber)
		}
		return false, trackingNumber
	}
	return false, 0
}
func (uscc *UDPServerContext) untrackAll(tracker *UDPServerClientLogicConnectionContext) {
	tracker.TrackedConnection.Range(func(key, value interface{}) bool {
		d := value.(*UDPTrackedConnectionContext)
		if d.Tracking {
			for i := 0; i <= 10; i++ {
				if !d.Tracked {
					break
				}
				if !uscc.untrackConnection(d.Channel) {
					break
				}
				<-time.NewTimer(time.Second * 3).C
			}

		}
		return true
	})

}

//Must non-block
func (uscc *UDPServerContext) untrackConnection(channel uint16) bool {
	var err error
	_ = err

	Sh := &proto.CommandHeader{CommandByte: proto.CommandByte_ChannelDestroy}
	s := &proto.AssociateChannelDestroy{}
	s.Channel = channel

	buf := bytes.NewBuffer(nil)

	err = struc.Pack(buf, Sh)
	if err != nil {
		log.Println(err)
	}

	err = struc.Pack(buf, s)
	if err != nil {
		log.Println(err)
	}

	ToClientTxTraffic := UDPServerTxToClientTraffic{}
	ToClientTxTraffic.Channel = 0x00
	ToClientTxTraffic.Payload = buf.Bytes()

	select {
	case uscc.TxToClient <- ToClientTxTraffic:

		inter, ok := uscc.TrackedRemoteAddr.Load(channel)

		if !ok {
			return false
		}
		interT := inter.(*UDPServerTrackedRemoteAddrContext)

		source := interT.LocalAddr
		dest := interT.RemoteAddr

		trackerI, ok := uscc.ClientLogicConnection.Load(source.String())
		if !ok {
			return false
		}
		tracker := trackerI.(*UDPServerClientLogicConnectionContext)

		trackerR := &UDPTrackedConnectionContext{}

		actual, found := tracker.TrackedConnection.LoadOrStore(dest.String(), trackerR)

		if found {
			trackerR = actual.(*UDPTrackedConnectionContext)
		}

		trackerR.Tracked = false
		trackerR.Channel = 0
		trackerR.TrackPending = false
		trackerR.Tracking = false

		uscc.TrackedRemoteAddr.Delete(channel)

		return true
	default:
		log.Println("Connection Untrack Request Discarded As It Would Block")
		return false
	}
}

func (uscc *UDPServerContext) rxFromClientWorker_OnControlAssociateDone(reader io.Reader) {
	var err error
	_ = err

	h := &proto.AssociateDoneHeader{}
	err = struc.Unpack(reader, h)

	if err != nil {
		log.Println(err)
		return
	}

	source := net.UDPAddr{Port: int(h.SourcePort), IP: proto.IPv4ByteToAddr(h.SourceIP)}
	dest := net.UDPAddr{Port: int(h.DestPort), IP: proto.IPv4ByteToAddr(h.DestIP)}

	trackerI, ok := uscc.ClientLogicConnection.Load(source.String())
	if !ok {
		return
	}
	tracker := trackerI.(*UDPServerClientLogicConnectionContext)

	trackerR := &UDPTrackedConnectionContext{}

	actual, found := tracker.TrackedConnection.LoadOrStore(dest.String(), trackerR)

	if found {
		trackerR = actual.(*UDPTrackedConnectionContext)
	}

	if trackerR.Channel == h.Channel {
		trackerR.Tracked = true
	}

}
func (uscc *UDPServerContext) rxFromClientWorker_OnControlAssociateV6Done(reader io.Reader) {
	var err error
	_ = err

	h := &proto.AssociateDoneV6Header{}
	err = struc.Unpack(reader, h)

	if err != nil {
		log.Println(err)
		return
	}

	source := net.UDPAddr{Port: int(h.SourcePort), IP: proto.IPv6ByteToAddr(h.SourceIP)}
	dest := net.UDPAddr{Port: int(h.DestPort), IP: proto.IPv6ByteToAddr(h.DestIP)}

	trackerI, ok := uscc.ClientLogicConnection.Load(source.String())
	if !ok {
		return
	}
	tracker := trackerI.(*UDPServerClientLogicConnectionContext)

	trackerR := &UDPTrackedConnectionContext{}

	actual, found := tracker.TrackedConnection.LoadOrStore(dest.String(), trackerR)

	if found {
		trackerR = actual.(*UDPTrackedConnectionContext)
	}

	if trackerR.Channel == h.Channel {
		trackerR.Tracked = true
	}

}
func (uscc *UDPServerContext) rxFromClientWorker_OnData(channel uint16, p []byte) {

	var err error
	_ = err

	inter, ok := uscc.TrackedRemoteAddr.Load(channel)

	if !ok {
		return
	}
	interT := inter.(*UDPServerTrackedRemoteAddrContext)

	source := interT.LocalAddr
	dest := interT.RemoteAddr

	trackerI, ok := uscc.ClientLogicConnection.Load(source.String())
	if !ok {
		return
	}
	tracker := trackerI.(*UDPServerClientLogicConnectionContext)

	if tracker.Conn == nil {
		uscc.untrackConnection(channel)
		return
	}

	//We Assume this operation will not block
	packetconn := tracker.Conn.(net.PacketConn)
	_, err = packetconn.WriteTo(p, &dest)
	if err != nil {
		log.Println(err)
	}
}
