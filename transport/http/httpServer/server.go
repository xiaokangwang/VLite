package httpServer

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/antiReplayWindow"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/httpconsts"
	"github.com/xiaokangwang/VLite/transport/http/websocketadp"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	mrand "math/rand"
)

func NewProviderServerSide(listenaddr string, password string, Uplistener transport.UnderlayTransportListener, ctx context.Context) *ProviderServerSide {
	pss := &ProviderServerSide{
		clientSet:    new(sync.Map),
		addr:         listenaddr,
		hh:           headerHolder.NewHttpHeaderHolderProcessor(password),
		Uplistener:   Uplistener,
		authlocation: httpconsts.Authlocation_Path,
		ctx:          ctx,
	}

	streamrel := ctx.Value(interfaces.ExtraOptionsHTTPServerStreamRelay)

	if streamrel != nil {
		pss.streamrelay = streamrel.(*interfaces.ExtraOptionsHTTPServerStreamRelayValue).Relay
	}

	var useAbsListener = false
	var AbsListener interfaces.AbstractListener

	if val := ctx.Value(interfaces.ExtraOptionsAbstractListener); val != nil {
		AbsListener = val.(*interfaces.ExtraOptionsAbstractListenerValue).AbsListener
		useAbsListener = val.(*interfaces.ExtraOptionsAbstractListenerValue).UseAbsListener
	}

	if useAbsListener {
		go pss.StartAbs(AbsListener)
	} else {
		go pss.Start()
	}

	return pss
}

type ProviderServerSide struct {
	clientSet    *sync.Map
	addr         string
	hh           *headerHolder.HttpHeaderHolderProcessor
	Uplistener   transport.UnderlayTransportListener
	authlocation int
	ctx          context.Context

	streamrelay interfaces.StreamRelayer
}

func (pss ProviderServerSide) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	var bearerc string

	switch pss.authlocation {
	case httpconsts.Authlocation_Header:
		res := r.Header.Get("Authorization")
		head := "Bearer "
		if !strings.HasPrefix(res, head) {
			http.NotFound(rw, r)
			return
		}

		bearerc = res[len(head):]
		break
	case httpconsts.Authlocation_Path:
		bearerc = r.URL.Path[1:]
	}

	if len(bearerc) < 16 {
		http.NotFound(rw, r)
		return
	}

	beardata := pss.hh.Open(bearerc)

	if beardata == nil {
		http.NotFound(rw, r)
		return
	}
	timediff := math.Abs(float64(beardata.Time - time.Now().Unix()))
	//fmt.Println(timediff)
	if timediff > 120 {
		http.NotFound(rw, r)
		return
	}

	if beardata.ConnID == [24]byte{0x00} {
		if r.Method == "GET" {
			fmt.Println("GET Test")
			if beardata.Masker >= (1 << 24) {
				http.NotFound(rw, r)
				return
			}
			rw.Header().Add("X-Accel-Buffering", "no")
			size := int(beardata.Masker)
			if WriteBufferSize == 0 {
				wrapper.TestBufferSizePayload(size, mrand.New(mrand.NewSource(time.Now().UnixNano())), rw)
			} else {
				bufw := bufio.NewWriterSize(rw, WriteBufferSize)

				rw.WriteHeader(200)

				wrapper.TestBufferSizePayload(size, mrand.New(mrand.NewSource(time.Now().UnixNano())), bufw)

				err := bufw.Flush()
				if err != nil {
					fmt.Println(err)
				}

			}
			return
		}

		if r.Method == "POST" {
			fmt.Println("POST Test")
			if beardata.Masker >= (1 << 24) {
				http.NotFound(rw, r)
				return
			}
			size := int(beardata.Masker)
			result := wrapper.TestBufferSizePayloadClient(size, r.Body, time.Now())
			rw.WriteHeader(500 + result)
			r.Body.Close()
			return
		}
	}

	ppsd, currentHTTPRequestCtx, done := pss.parpareforconnection(beardata)
	if done {
		http.NotFound(rw, r)
		return
	}

	upg := websocket.Upgrader{}
	if beardata.Flags&proto.HttpHeaderFlag_WebsocketConnection != 0 {
		conn, err := upg.Upgrade(rw, r, nil)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		wsconn := websocketadp.NewWsAdp(conn)
		if beardata.Flags&proto.HttpHeaderFlag_AlternativeChannelConnection != 0 {
			pss.streamrelay.RelayStream(wsconn, pss.ctx)
			return
		}
		go wrapper.ReceivePacketOverReader(beardata.Masker, wsconn, ppsd.RxChan, currentHTTPRequestCtx)
		wrapper.SendPacketOverWriter(beardata.Masker, wsconn, ppsd.TxChan, 0, currentHTTPRequestCtx)
		return
	}

	if r.Method == "GET" {
		fmt.Println("GET")
		ppsd.Get(rw, r, beardata.Masker, currentHTTPRequestCtx)
		return
	}

	if r.Method == "POST" {
		fmt.Println("POST")
		ppsd.Post(rw, r, beardata.Masker, currentHTTPRequestCtx)
		return
	}

	http.NotFound(rw, r)
}

func (pss ProviderServerSide) parpareforconnection(beardata *proto.HttpHeaderHolder) (*ProviderConnServerSide, context.Context, bool) {
	ppsd := &ProviderConnServerSide{}

	copy(ppsd.ID[:], beardata.ConnID[:])

	ppsd.RxChan = make(chan []byte, 8)
	ppsd.TxChan = make(chan []byte, 8)
	ppsd.pss = &pss
	ppsd.noReplayChecker = antiReplayWindow.NewAntiReplayWindow(119)
	ppsd.boostConnectionGSRV = &interfaces.ExtraOptionsBoostConnectionGracefulShutdownRequestValue{ShouldClose: make(chan interface{})}

	a, ok := pss.clientSet.LoadOrStore(synthid(beardata), ppsd)
	if ok {
		ppsd = a.(*ProviderConnServerSide)
	} else {
		connid := ppsd.ID[:]
		connctx := context.WithValue(pss.ctx, interfaces.ExtraOptionsConnID, connid)
		connctx = context.WithValue(connctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())

		univ := &interfaces.ExtraOptionsUniConnAttribValue{
			ID:   ppsd.ID[:],
			Rand: beardata.Rand[:],
			Iter: beardata.ConnIter,
		}

		connctx = context.WithValue(connctx, interfaces.ExtraOptionsUniConnAttrib, univ)

		connctx = pss.Uplistener.Connection(adp.NewRxTxToConn(ppsd.TxChan, ppsd.RxChan, ppsd, connctx), connctx)

		//Should use connctx

		connctx2, cancel := context.WithCancel(connctx)
		ppsd.drop = cancel
		ppsd.ctx = connctx2

		go ppsd.BoostingListener(connctx2)
	}

	if !ppsd.noReplayChecker.Check(beardata.Rand[:]) {
		return nil, nil, true
	}

	currentHTTPRequestCtx := pss.ctx

	currentHTTPRequestCtx = context.WithValue(currentHTTPRequestCtx,
		interfaces.ExtraOptionsBoostConnectionGracefulShutdownRequest,
		ppsd.boostConnectionGSRV)

	if beardata.Flags&proto.HttpHeaderFlag_BoostConnection != 0 {
		currentHTTPRequestCtx = context.WithValue(currentHTTPRequestCtx,
			interfaces.ExtraOptionsHTTPTransportConnIsBoostConnection,
			true)
		mbus := ibus.ConnectionMessageBusFromContext(ppsd.ctx)

		connstr := connidutil.ConnIDToString(ppsd.ctx)

		Boostchannel := ibusTopic.ConnBoostMode(connstr)

		w := ibusInterface.ConnBoostMode{
			Enable:         true,
			TimeoutAtLeast: 60,
		}
		_, erremit := mbus.Emit(ppsd.ctx, Boostchannel, w)
		if erremit != nil {
			fmt.Println(erremit.Error())
		}
	}
	return ppsd, currentHTTPRequestCtx, false
}

func (pss *ProviderServerSide) Start() {
	err := http.ListenAndServe(pss.addr, pss)
	if err != nil {
		fmt.Println(err)
	}
}

func (pss *ProviderServerSide) StartAbs(AbsListener interfaces.AbstractListener) {

	for {
		conn, bearerc, err := AbsListener.Accept(pss.ctx)
		if err != nil {
			fmt.Println(err.Error())
		}

		beardata := pss.hh.Open(bearerc)

		if beardata == nil {
			conn.Close()
			continue
		}
		timediff := math.Abs(float64(beardata.Time - time.Now().Unix()))
		//fmt.Println(timediff)
		if timediff > 120 {
			conn.Close()
			continue
		}

		ppsd, currentHTTPRequestCtx, done := pss.parpareforconnection(beardata)
		if done {
			conn.Close()
			continue
		}

		go wrapper.ReceivePacketOverReader(beardata.Masker, conn, ppsd.RxChan, currentHTTPRequestCtx)
		go wrapper.SendPacketOverWriter(beardata.Masker, conn, ppsd.TxChan, 0, currentHTTPRequestCtx)

	}

}

type ProviderConnServerSide struct {
	TxChan chan []byte
	RxChan chan []byte
	ID     [24]byte

	//TODO WARNING this is actually a copy!
	pss *ProviderServerSide

	noReplayChecker *antiReplayWindow.AntiReplayWindow

	boostConnectionGSRV *interfaces.ExtraOptionsBoostConnectionGracefulShutdownRequestValue

	drop context.CancelFunc
	ctx  context.Context
}

func (pcn *ProviderConnServerSide) Close() error {
	//TODO notify client
	pcn.pss.clientSet.Delete(pcn.ID)
	pcn.drop()
	return nil
}

func (pcn *ProviderConnServerSide) Post(rw http.ResponseWriter, r *http.Request, masker int64, ctx context.Context) {
	wrapper.ReceivePacketOverReader(masker, r.Body, pcn.RxChan, ctx)
	r.Body.Close()
}

var WriteBufferSize = 0

func (pcn *ProviderConnServerSide) Get(rw http.ResponseWriter, r *http.Request, masker int64, ctx context.Context) {
	rw.Header().Add("X-Accel-Buffering", "no")
	NetworkBuffering := 0

	bufferingCfg := ctx.Value(interfaces.ExtraOptionsHTTPNetworkBufferSize)
	if bufferingCfg != nil {
		NetworkBuffering = bufferingCfg.(*interfaces.ExtraOptionsHTTPNetworkBufferSizeValue).NetworkBufferSize
	}
	wrapper.SendPacketOverWriter(masker, rw, pcn.TxChan, NetworkBuffering, ctx)
}
