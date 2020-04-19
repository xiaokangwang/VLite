package httpServer

import (
	"fmt"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/httpconsts"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"net/http"
	"strings"
	"sync"
)

func NewProviderServerSide(listenaddr string, password string, Uplistener transport.UnderlayTransportListener) *ProviderServerSide {
	pss := &ProviderServerSide{
		clientSet:        new(sync.Map),
		addr:             listenaddr,
		hh:               headerHolder.NewHttpHeaderHolderProcessor(password),
		Uplistener:       Uplistener,
		authlocation:     httpconsts.Authlocation_Path,
		networkbuffering: 0,
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

	ppsd := &ProviderConnServerSide{}

	copy(ppsd.ID[:], beardata.ConnID[:])

	ppsd.RxChan = make(chan []byte, 8)
	ppsd.TxChan = make(chan []byte, 8)
	ppsd.pss = &pss

	a, ok := pss.clientSet.LoadOrStore(beardata.ConnID, ppsd)

	if ok {
		ppsd = a.(*ProviderConnServerSide)
	} else {
		go pss.Uplistener.Connection(adp.NewRxTxToConn(ppsd.TxChan, ppsd.RxChan, ppsd))
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
}

func (pcn *ProviderConnServerSide) Close() error {
	//TODO notify client
	pcn.pss.clientSet.Delete(pcn.ID)
	return nil
}

func (pcn *ProviderConnServerSide) Post(rw http.ResponseWriter, r *http.Request, masker int64) {
	wrapper.ReceivePacketOverReader(masker, r.Body, pcn.RxChan)
	r.Body.Close()
}
func (pcn *ProviderConnServerSide) Get(rw http.ResponseWriter, r *http.Request, masker int64) {
	wrapper.SendPacketOverWriter(masker, rw, pcn.TxChan, pcn.pss.networkbuffering)
}
