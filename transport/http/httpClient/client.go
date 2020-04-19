package httpClient

import (
	"crypto/rand"
	"fmt"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"io"
	mrand "math/rand"
	"net"
	"net/http"
	"time"
)

func NewProviderClient(HttpRequestEndpoint string,
	MaxTxConnection int,
	MaxRxConnection int,
	password string) *ProviderClient {
	prc := &ProviderClient{HttpRequestEndpoint: HttpRequestEndpoint,
		MaxRxConnection: MaxRxConnection,
		MaxTxConnection: MaxTxConnection,
		RxChan:          make(chan []byte, 8),
		TxChan:          make(chan []byte, 8)}
	id := make([]byte, 24)
	io.ReadFull(rand.Reader, id)
	prc.hh = headerHolder.NewHttpHeaderHolderProcessor(password)
	go prc.StartConnections()
	return prc
}
func (pc *ProviderClient) StartConnections() {
	var i int
	more := true
	for more {
		more = false
		if i < pc.MaxRxConnection {
			go pc.DialRxConnectionD()
			more = true
		}
		if i < pc.MaxRxConnection {
			go pc.DialRxConnectionD()
			more = true
		}
		<-time.NewTimer(time.Second).C
	}
}

func (pc *ProviderClient) DialTxConnectionD() {
	for !pc.closed {
		pc.DialTxConnection()
	}
}

func (pc *ProviderClient) DialRxConnectionD() {
	for !pc.closed {
		pc.DialRxConnection()
	}
}

type ProviderClient struct {
	HttpRequestEndpoint string
	MaxTxConnection     int
	MaxRxConnection     int

	TxChan chan []byte
	RxChan chan []byte

	ID []byte

	hh *headerHolder.HttpHeaderHolderProcessor

	closed bool
}

func (pc *ProviderClient) Close() error {
	pc.closed = true
}

func (pc *ProviderClient) DialTxConnection() {

	hc := pc.getHttpClient()

	masking, preader, pwriter := pc.prepareHTTP()

	go wrapper.SendPacketOverWriter(masking, pwriter, pc.TxChan)

	req, err := http.NewRequest("POST", pc.HttpRequestEndpoint, preader)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pc.reqprepare(req, masking)

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err.Error())
		return
	}

	resp.Body.Close()
}

func (pc *ProviderClient) reqprepare(req *http.Request, masking int64) {
	req.Header.Del("User-Agent")

	ph := proto.HttpHeaderHolder{
		Masker: masking,
		ConnID: pc.ID,
	}

	BearToken := pc.hh.Seal(ph)

	req.Header.Set("Authorization", "Bearer "+BearToken)
}

func (pc *ProviderClient) prepareHTTP() (int64, *io.PipeReader, *io.PipeWriter) {
	masking := mrand.Int63()

	preader, pwriter := io.Pipe()
	return masking, preader, pwriter
}

func (pc *ProviderClient) getHttpClient() http.Client {
	transport := &http.Transport{Proxy: nil,
		DialContext: (&net.Dialer{Timeout: 30 * time.Second,
			KeepAlive: 30 * time.Second,}).DialContext,
		MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,}

	hc := http.Client{
		Transport:     transport,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
	}
	return hc
}
func (pc *ProviderClient) DialRxConnection() {
	hc := pc.getHttpClient()

	masking, preader, pwriter := pc.prepareHTTP()

	go wrapper.ReceivePacketOverReader(masking, preader, pc.RxChan)

	req, err := http.NewRequest("GET", pc.HttpRequestEndpoint, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pc.reqprepare(req, masking)

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err.Error())
		return
	}

	io.Copy(pwriter, resp.Body)
}

func (pc *ProviderClient) AsConn() net.Conn {
	return adp.NewRxTxToConn(pc.TxChan, pc.RxChan, pc)
}
