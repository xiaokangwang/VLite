package puniServer

import (
	"context"
	"fmt"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"github.com/xiaokangwang/VLite/transport/contextAwareConn"
	udpsctpserver "github.com/xiaokangwang/VLite/transport/packetsctp/sctprelay"
	"net"
)

func NewPacketUniServer(TxChannel chan interfaces.TrafficWithChannelTag,
	TxDataChannel chan interfaces.TrafficWithChannelTag,
	RxChannel chan interfaces.TrafficWithChannelTag,
	streamrelay interfaces.StreamRelayer,
	Password []byte,
	ctx context.Context) *PacketUniServer {
	pus := &PacketUniServer{TxChannel: TxChannel, TxDataChannel: TxDataChannel, RxChannel: RxChannel, streamrelay: streamrelay, Password: Password, ctx: ctx}
	return pus
}

type PacketUniServer struct {
	TxChannel     chan interfaces.TrafficWithChannelTag
	TxDataChannel chan interfaces.TrafficWithChannelTag
	RxChannel     chan interfaces.TrafficWithChannelTag
	Password      []byte
	ctx           context.Context
	carrierCtx    context.Context
	carrierCancel context.CancelFunc
	streamrelay   interfaces.StreamRelayer

	relay *udpsctpserver.PacketSCTPRelay
}

func (pu *PacketUniServer) OnCarrier(conn net.Conn, connctx context.Context) {
	if pu.carrierCancel != nil {
		pu.carrierCancel()
	}
	pu.carrierCtx, pu.carrierCancel = context.WithCancel(connctx)

	C2STraffic := make(chan interfaces.TrafficWithChannelTag, 8)
	C2SDataTraffic := make(chan interfaces.TrafficWithChannelTag, 8)
	S2CTraffic := make(chan interfaces.TrafficWithChannelTag, 8)

	sctps := udpsctpserver.NewPacketRelayServer(contextAwareConn.NewContextAwareConn(conn, pu.carrierCtx), C2STraffic, C2SDataTraffic, S2CTraffic, pu.streamrelay, pu.Password, pu.carrierCtx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-pu.TxChannel:
				//spew.Dump(data)
				C2STraffic <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done(): //TODO Leak
				return
			}
		}

	}(pu.carrierCtx)
	go func(ctx context.Context) {
		for {
			select {
			case data := <-pu.TxDataChannel:
				//spew.Dump(data)
				C2SDataTraffic <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(pu.carrierCtx)
	go func(ctx context.Context) {
		for {
			select {
			case data := <-S2CTraffic:
				//spew.Dump(data)
				pu.RxChannel <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
				return
			}
		}

	}(pu.carrierCtx)

	pu.relay = sctps
}

func (pu *PacketUniServer) OnAutoCarrier(conn net.Conn, connctx context.Context) {
	go pu.onAutoCarrier(connctx, conn)
}

func (pu *PacketUniServer) onAutoCarrier(connctx context.Context, conn net.Conn) {
	ConnIDString := connidutil.ConnIDToString(connctx)
	BusTopic := ibusTopic.ConnReHandShake(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(connctx)

	mbus.RegisterTopics(BusTopic)

	handshakeModeOptChan := make(chan ibusInterface.ConnReHandshake, 8)

	mbus.RegisterHandler(BusTopic+"PacketUniDServer", &bus.Handler{
		Handle: func(e *bus.Event) {
			d := e.Data.(ibusInterface.ConnReHandshake)
			select {
			case handshakeModeOptChan <- d:
			default:
				fmt.Println("WARNING: boost mode hint discarded")
			}

		},
		Matcher: BusTopic,
	})

	go pu.OnCarrier(conn, connctx)

	for {
		select {
		case <-connctx.Done():
			fmt.Println("PacketUniServer Done")
			return
		case <-handshakeModeOptChan:
			go pu.OnCarrier(conn, connctx)
			fmt.Println("PacketUniServer Rehandshake")
		}
	}
}

func (pu *PacketUniServer) RateLimitTcpServerWrite(second int, size int, size2 int) {
	//pu.relay.RateLimitTcpServerWrite(second, size , size2)
}

func (pu *PacketUniServer) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	if pu.relay == nil {
		return 0, 0
	}
	return pu.relay.GetTransmitLayerSentRecvStats()
}
