package httpServer

import (
	"bufio"
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/antiReplayWindow"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/httpconsts"
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
		clientSet:        new(sync.Map),
		addr:             listenaddr,
		hh:               headerHolder.NewHttpHeaderHolderProcessor(password),
		Uplistener:       Uplistener,
		authlocation:     httpconsts.Authlocation_Path,
		networkbuffering: 0,
		ctx:              ctx,
	}
	go pss.Start()
	return pss
}

type ProviderServerSide struct {
	clientSet        *sync.Map
	addr             string
	hh               *headerHolder.HttpHeaderHolderProcessor
	Uplistener       transport.UnderlayTransportListener
	authlocation     int
	networkbuffering int
	ctx              context.Context
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

	ppsd := &ProviderConnServerSide{}

	copy(ppsd.ID[:], beardata.ConnID[:])

	ppsd.RxChan = make(chan []byte, 8)
	ppsd.TxChan = make(chan []byte, 8)
	ppsd.pss = &pss
	ppsd.noReplayChecker = antiReplayWindow.NewAntiReplayWindow(119)

	a, ok := pss.clientSet.LoadOrStore(beardata.ConnID, ppsd)

	if ok {
		ppsd = a.(*ProviderConnServerSide)
	} else {
		connid := ppsd.ID[:]
		connctx := context.WithValue(pss.ctx, interfaces.ExtraOptionsConnID, connid)
		go pss.Uplistener.Connection(adp.NewRxTxToConn(ppsd.TxChan, ppsd.RxChan, ppsd), connctx)
	}

	if !ppsd.noReplayChecker.Check(beardata.Rand[:]) {
		http.NotFound(rw, r)
		return
	}

	if r.Method == "GET" {
		fmt.Println("GET")
		ppsd.Get(rw, r, beardata.Masker)
		return
	}

	if r.Method == "POST" {
		fmt.Println("POST")
		ppsd.Post(rw, r, beardata.Masker)
		return
	}

	http.NotFound(rw, r)
}

func (pss *ProviderServerSide) Start() {
	err := http.ListenAndServe(pss.addr, pss)
	if err != nil {
		fmt.Println(err)
	}
}

type ProviderConnServerSide struct {
	TxChan chan []byte
	RxChan chan []byte
	ID     [24]byte

	//TODO WARNING this is actually a copy!
	pss *ProviderServerSide

	noReplayChecker *antiReplayWindow.AntiReplayWindow
}

func (pcn *ProviderConnServerSide) Close() error {
	//TODO notify client
	pcn.pss.clientSet.Delete(pcn.ID)
	return nil
}

func (pcn *ProviderConnServerSide) Post(rw http.ResponseWriter, r *http.Request, masker int64) {
	wrapper.ReceivePacketOverReader(masker, r.Body, pcn.RxChan, pcn.pss.ctx)
	r.Body.Close()
}

var WriteBufferSize = 0

func (pcn *ProviderConnServerSide) Get(rw http.ResponseWriter, r *http.Request, masker int64) {
	rw.Header().Add("X-Accel-Buffering", "no")

	wrapper.SendPacketOverWriter(masker, rw, pcn.TxChan, pcn.pss.networkbuffering, pcn.pss.ctx)
}
