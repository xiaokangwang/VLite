package puniClient

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
	"io"
	"net"
	"time"
)

func NewPacketUniClient(TxChannel chan interfaces.TrafficWithChannelTag,
	TxDataChannel chan interfaces.TrafficWithChannelTag,
	RxChannel chan interfaces.TrafficWithChannelTag,
	Password []byte,
	ctx context.Context) *PacketUniClient {
	puc := &PacketUniClient{TxChannel: TxChannel, TxDataChannel: TxDataChannel, RxChannel: RxChannel, Password: Password, ctx: ctx}
	return puc
}

type PacketUniClient struct {
	TxChannel     chan interfaces.TrafficWithChannelTag
	TxDataChannel chan interfaces.TrafficWithChannelTag
	RxChannel     chan interfaces.TrafficWithChannelTag
	Password      []byte
	ctx           context.Context
	carrierCtx    context.Context
	carrierCancel context.CancelFunc

	relay *udpsctpserver.PacketSCTPRelay
}

func (pu *PacketUniClient) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	if pu.relay == nil {
		return 0, 0
	}
	return pu.relay.GetTransmitLayerSentRecvStats()
}

func (pu *PacketUniClient) OnCarrier(conn net.Conn, connctx context.Context) {
	if pu.carrierCancel != nil {
		pu.carrierCancel()
	}
	pu.carrierCtx, pu.carrierCancel = context.WithCancel(pu.ctx)

	C2STraffic := make(chan interfaces.TrafficWithChannelTag, 8)
	C2SDataTraffic := make(chan interfaces.TrafficWithChannelTag, 8)
	S2CTraffic := make(chan interfaces.TrafficWithChannelTag, 8)

	sctps := udpsctpserver.NewPacketRelayClient(contextAwareConn.NewContextAwareConn(conn, pu.carrierCtx), C2STraffic, C2SDataTraffic, S2CTraffic, pu.Password, pu.carrierCtx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-pu.TxChannel:
				//spew.Dump(data)
				C2STraffic <- interfaces.TrafficWithChannelTag(data)
			case <-ctx.Done():
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

func (pu *PacketUniClient) OnAutoCarrier(conn net.Conn, connctx context.Context) {
	go pu.onAutoCarrier(connctx, conn)
}

func (pu *PacketUniClient) onAutoCarrier(connctx context.Context, conn net.Conn) {
	ConnIDString := connidutil.ConnIDToString(connctx)
	BusTopic := ibusTopic.ConnReHandShake(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(connctx)

	mbus.RegisterTopics(BusTopic)

	handshakeModeOptChan := make(chan ibusInterface.ConnReHandshake, 8)

	mbus.RegisterHandler(BusTopic+"PacketUniClient", &bus.Handler{
		Handle: func(e *bus.Event) {
			d := e.Data.(ibusInterface.ConnReHandshake)
			select {
			case handshakeModeOptChan <- d:
				fmt.Println("PacketUniClient Rehandshake M")
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
			fmt.Println("PacketUniClient Done")
			return
		case opt := <-handshakeModeOptChan:
			if opt.FullReHandshake == true {
				go pu.OnCarrier(conn, connctx)
			}
			fmt.Println("PacketUniClient Rehandshake")
		}
	}
}

func (pu *PacketUniClient) ClientOpenStream() io.ReadWriteCloser {
	for i := 0; i <= 6000; i++ {
		if pu.relay == nil {
			time.Sleep(time.Second / 10)
		}
	}
	return pu.relay.ClientOpenStream()

}
