package httpClient

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"golang.org/x/net/websocket"
	"sync/atomic"
	"time"
)

func (pc *ProviderClient) DialWsConnection(ctx context.Context) {
	masking, _, _ := pc.prepareHTTP()
	h := pc.createBearToken(masking, ctx)
	HttpRequestEndpointws := "ws" + pc.HttpRequestEndpoint[4:]
	ws, err := websocket.Dial(HttpRequestEndpointws+"/"+h, "", pc.HttpRequestEndpoint)
	if err != nil {
		fmt.Println(err.Error())
	}
	go wrapper.ReceivePacketOverReader(masking, ws, pc.RxChan, ctx)
	wrapper.SendPacketOverWriter(masking, ws, pc.TxChan, 0, ctx)
}

func (pc *ProviderClient) StartConnectionsWS() {
	var i int
	more := true
	toDial := pc.MaxRxConnection + pc.MaxTxConnection
	for more {
		more = false
		if i < toDial {
			go pc.DialWsConnectionD(pc.connctx)
			more = true
		}
		i++
		<-time.NewTimer(time.Second / 2).C
	}
}
func (pc *ProviderClient) DialWsConnectionD(ctx context.Context) {
	var shouldNotRedialDef = int32(0)
	var shouldNotRedial *int32
	shouldNotRedial = &shouldNotRedialDef

	noredial := ctx.Value(interfaces.ExtraOptionsBoostConnectionShouldNotRedial)
	if noredial != nil {
		shouldNotRedial = &noredial.(*interfaces.ExtraOptionsBoostConnectionShouldNotRedialValue).
			ShouldNotReDial
	}

	for ctx.Err() == nil && atomic.LoadInt32(shouldNotRedial) == 0 {
		nobust := time.NewTimer(time.Second)
		pc.DialWsConnection(ctx)
		<-nobust.C
	}
}
