package httpClient

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/httpconsts"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"io"
	mrand "math/rand"
	"net"
	"net/http"
	"time"
)

func NewProviderClientCreator(HttpRequestEndpoint string,
	MaxTxConnection int,
	MaxRxConnection int,
	password string) *ProviderClientCreator {
	return &ProviderClientCreator{
		HttpRequestEndpoint: HttpRequestEndpoint,
		MaxRxConnection:     MaxRxConnection,
		MaxTxConnection:     MaxTxConnection,
		password:            password,
	}
}

type ProviderClientCreator struct {
	HttpRequestEndpoint string
	MaxTxConnection     int
	MaxRxConnection     int
	password            string
}

func (p ProviderClientCreator) Connect() (net.Conn, error) {
	return NewProviderClient(p.HttpRequestEndpoint,
		p.MaxTxConnection,
		p.MaxRxConnection,
		p.password).AsConn(), nil
}

func NewProviderClient(HttpRequestEndpoint string,
	MaxTxConnection int,
	MaxRxConnection int,
	password string) *ProviderClient {
	prc := &ProviderClient{HttpRequestEndpoint: HttpRequestEndpoint,
		MaxRxConnection: MaxRxConnection,
		MaxTxConnection: MaxTxConnection,
		RxChan:          make(chan []byte, 8),
		TxChan:          make(chan []byte, 8),
		authlocation:    httpconsts.Authlocation_Path}
	id := make([]byte, 24)
	io.ReadFull(rand.Reader, id)
	prc.ID = id
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
		//<-time.NewTimer(time.Second).C
		if i < pc.MaxTxConnection {
			go pc.DialTxConnectionD()
			more = true
		}
		i++
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

	authlocation int
}

func (pc *ProviderClient) Close() error {
	pc.closed = true
	return nil
}

func (pc *ProviderClient) DialTxConnection() {

	hc := pc.getHttpClient()

	masking, preader, pwriter := pc.prepareHTTP()

	go wrapper.SendPacketOverWriter(masking, pwriter, pc.TxChan, 0)

	req, err := http.NewRequest("POST", pc.HttpRequestEndpoint, preader)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pc.reqprepare(req, masking)

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err4.Error())
		return
	}

	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}

	resp.Body.Close()
}

func (pc *ProviderClient) reqprepare(req *http.Request, masking int64) {
	req.Header.Set("User-Agent", "")

	ph := proto.HttpHeaderHolder{
		Masker: masking,
	}
	mrand.Read(ph.Rand[:])
	copy(ph.ConnID[:], pc.ID)
	BearToken := pc.hh.Seal(ph)

	switch pc.authlocation {
	case httpconsts.Authlocation_Header:
		req.Header.Set("Authorization", "Bearer "+BearToken)
		break
	case httpconsts.Authlocation_Path:
		req.URL.Path = "/" + BearToken
	}
}

func (pc *ProviderClient) reqprepareTest(req *http.Request, masking int64) {
	req.Header.Set("User-Agent", "")

	ph := proto.HttpHeaderHolder{
		Masker: masking,
	}
	BearToken := pc.hh.Seal(ph)

	switch pc.authlocation {
	case httpconsts.Authlocation_Header:
		req.Header.Set("Authorization", "Bearer "+BearToken)
		break
	case httpconsts.Authlocation_Path:
		req.URL.Path = "/" + BearToken
	}
}

func (pc *ProviderClient) prepareHTTP() (int64, *io.PipeReader, *io.PipeWriter) {
	masking := mrand.Int63()

	preader, pwriter := io.Pipe()
	return masking, preader, pwriter
}

func (pc *ProviderClient) getHttpClient() http.Client {
	transport := &http.Transport{Proxy: nil,
		DialContext: (&net.Dialer{Timeout: 30 * time.Second,
			KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second}

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
		fmt.Println(err4.Error())
		return
	}
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}
	io.Copy(pwriter, resp.Body)
}

func (pc *ProviderClient) AsConn() net.Conn {
	return adp.NewRxTxToConn(pc.TxChan, pc.RxChan, pc)
}

func (pc *ProviderClient) DialRxTestConnection(TestSize int) int {
	hc := pc.getHttpClient()

	req, err := http.NewRequest("GET", pc.HttpRequestEndpoint, nil)
	if err != nil {
		fmt.Println(err.Error())
		return -3
	}

	pc.reqprepareTest(req, int64(TestSize))

	timenow := time.Now()

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err4.Error())
		return -3
	}
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}

	result := wrapper.TestBufferSizePayloadClient(TestSize, resp.Body, timenow)

	resp.Body.Close()

	return result
}

var WriteBufferSize = 0

func (pc *ProviderClient) DialTxConnectionTest(TestSize int) int {

	hc := pc.getHttpClient()

	_, preader, pwriter := pc.prepareHTTP()
	if WriteBufferSize == 0 {
		go wrapper.TestBufferSizePayload(TestSize, mrand.New(mrand.NewSource(time.Now().UnixNano())), pwriter)
	} else {
		bufw := bufio.NewWriterSize(pwriter, WriteBufferSize)
		go wrapper.TestBufferSizePayload(TestSize, mrand.New(mrand.NewSource(time.Now().UnixNano())), bufw)
		go func() {
			time.Sleep(time.Second * 12)
			bufw.Flush()
			pwriter.Close()
		}()
	}

	req, err := http.NewRequest("POST", pc.HttpRequestEndpoint, preader)
	if err != nil {
		fmt.Println(err.Error())
		return -3
	}

	pc.reqprepareTest(req, int64(TestSize))

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err4.Error())
		return -3
	}

	result := resp.StatusCode - 500
	resp.Body.Close()
	return result
}
