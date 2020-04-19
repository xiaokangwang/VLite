package httpServer

import (
	"fmt"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"net/http"
	"strings"
	"sync"
)

func NewProviderServerSide(listenaddr string, password string, Uplistener transport.UnderlayTransportListener) *ProviderServerSide {
	pss := &ProviderServerSide{
		clientSet:  new(sync.Map),
		addr:       listenaddr,
		hh:         headerHolder.NewHttpHeaderHolderProcessor(password),
		Uplistener: Uplistener,
	}
	go pss.Start()
	return pss
}

type ProviderServerSide struct {
	clientSet  *sync.Map
	addr       string
	hh         *headerHolder.HttpHeaderHolderProcessor
	Uplistener transport.UnderlayTransportListener
}

func (pss ProviderServerSide) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	res := r.Header.Get("Authorization")
	head := "Bearer "
	if !strings.HasPrefix(res, head) {
		http.NotFound(rw, r)
		return
	}

	bearerc := head[len(head):]

	beardata := pss.hh.Open(bearerc)

	if beardata == nil {
		http.NotFound(rw, r)
		return
	}

	ppsd := &ProviderConnServerSide{}

	ppsd.ID = beardata.ConnID

	ppsd.RxChan = make(chan []byte, 8)
	ppsd.TxChan = make(chan []byte, 8)
	ppsd.pss = &pss

	a, ok := pss.clientSet.LoadOrStore(beardata.ConnID, ppsd)

	if ok {
		ppsd = a.(*ProviderConnServerSide)
	} else {
		pss.Uplistener.Connection(adp.NewRxTxToConn(ppsd.TxChan, ppsd.RxChan, ppsd))
	}

	if r.Method == "GET" {
		ppsd.Get(rw, r, beardata.Masker)
		return
	}

	if r.Method == "POST" {
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
	ID     []byte

	//TODO WARNING this is actually a copy!
	pss *ProviderServerSide
}

func (pcn *ProviderConnServerSide) Close() error {
	//TODO notify client
	pcn.pss.clientSet.Delete(pcn.ID)
}

func (pcn *ProviderConnServerSide) Post(rw http.ResponseWriter, r *http.Request, masker int64) {
	wrapper.ReceivePacketOverReader(masker, r.Body, pcn.RxChan)
	r.Body.Close()
	rw.WriteHeader(200)
}
func (pcn *ProviderConnServerSide) Get(rw http.ResponseWriter, r *http.Request, masker int64) {
	wrapper.SendPacketOverWriter(masker, rw, pcn.TxChan)
}
