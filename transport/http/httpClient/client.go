package httpClient

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/http/httpconsts"
	"github.com/xiaokangwang/VLite/transport/http/wrapper"
	"golang.org/x/net/proxy"
	"io"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

func NewProviderClientCreator(HttpRequestEndpoint string,
	MaxTxConnection int,
	MaxRxConnection int,
	password string, ctx context.Context) *ProviderClientCreator {
	return &ProviderClientCreator{
		HttpRequestEndpoint: HttpRequestEndpoint,
		MaxRxConnection:     MaxRxConnection,
		MaxTxConnection:     MaxTxConnection,
		password:            password,
		ctx:                 ctx,
	}
}

type ProviderClientCreator struct {
	HttpRequestEndpoint string
	MaxTxConnection     int
	MaxRxConnection     int
	password            string
	ctx                 context.Context
}

func (p ProviderClientCreator) Connect(ctx context.Context) (net.Conn, error, context.Context) {

	pc := NewProviderClient(p.HttpRequestEndpoint,
		p.MaxTxConnection,
		p.MaxRxConnection,
		p.password, ctx)

	return pc.AsConn(), nil, pc.connctx
}

func NewProviderClient(HttpRequestEndpoint string,
	MaxTxConnection int,
	MaxRxConnection int,
	password string, ctx context.Context) *ProviderClient {
	prc := &ProviderClient{HttpRequestEndpoint: HttpRequestEndpoint,
		MaxRxConnection: MaxRxConnection,
		MaxTxConnection: MaxTxConnection,
		RxChan:          make(chan []byte, 8),
		TxChan:          make(chan []byte, 8),
		authlocation:    httpconsts.Authlocation_Path,
		ctx:             ctx}
	id := make([]byte, 24)
	io.ReadFull(rand.Reader, id)

	unival := ctx.Value(interfaces.ExtraOptionsUniConnAttrib)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		copy(id, uniAtt.ID)
	}

	prc.ID = id
	prc.hh = headerHolder.NewHttpHeaderHolderProcessor(password)

	prc.MaxBoostRxConnection = 16
	prc.MaxBoostTxConnection = 16

	boostvsl := ctx.Value(interfaces.ExtraOptionsBoostConnectionSettingsHTTPTransport)

	if boostvsl != nil {
		bcfg := boostvsl.(*interfaces.ExtraOptionsBoostConnectionSettingsHTTPTransportValue)
		prc.MaxBoostRxConnection = bcfg.MaxBoostRxConnection
		prc.MaxBoostTxConnection = bcfg.MaxBoostTxConnection

	}

	vctx := context.WithValue(ctx, interfaces.ExtraOptionsConnID, id)

	vctx = context.WithValue(vctx, interfaces.ExtraOptionsMessageBusByConn, ibus.NewMessageBus())

	prc.connctx, prc.closeCtx = context.WithCancel(vctx)
	go prc.StartConnections()
	go prc.BoostingListener()

	return prc
}
func (pc *ProviderClient) StartConnections() {
	var i int
	more := true
	for more {
		more = false
		if i < pc.MaxRxConnection {
			go pc.DialRxConnectionD(pc.connctx)
			more = true
		}
		//<-time.NewTimer(time.Second).C
		if i < pc.MaxTxConnection {
			go pc.DialTxConnectionD(pc.connctx)
			more = true
		}
		i++
		<-time.NewTimer(time.Second).C
	}
}

func (pc *ProviderClient) DialTxConnectionD(ctx context.Context) {
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
		pc.DialTxConnection(ctx)
		<-nobust.C
	}
}

func (pc *ProviderClient) DialRxConnectionD(ctx context.Context) {
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
		pc.DialRxConnection(ctx)
		<-nobust.C
	}
}

type ProviderClient struct {
	ctx context.Context

	connctx context.Context

	HttpRequestEndpoint string
	MaxTxConnection     int
	MaxRxConnection     int

	MaxBoostTxConnection int
	MaxBoostRxConnection int

	TxChan chan []byte
	RxChan chan []byte

	ID []byte

	hh *headerHolder.HttpHeaderHolderProcessor

	closeCtx context.CancelFunc

	authlocation int
}

func (pc *ProviderClient) GetConnCtx() context.Context {
	return pc.connctx
}

func (pc *ProviderClient) Close() error {
	pc.closeCtx()
	return nil
}

func (pc *ProviderClient) DialTxConnection(ctx context.Context) {

	hc := pc.getHttpClient()

	masking, preader, pwriter := pc.prepareHTTP()

	NetworkBuffering := 0

	bufferingCfg := ctx.Value(interfaces.ExtraOptionsHTTPNetworkBufferSize)
	if bufferingCfg != nil {
		NetworkBuffering = bufferingCfg.(*interfaces.ExtraOptionsHTTPNetworkBufferSizeValue).NetworkBufferSize
	}

	go wrapper.SendPacketOverWriter(masking, pwriter, pc.TxChan, NetworkBuffering, ctx)

	req, err := http.NewRequest("POST", pc.HttpRequestEndpoint, preader)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pc.reqprepare(req, masking, ctx)

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

func (pc *ProviderClient) reqprepare(req *http.Request, masking int64, ctx context.Context) {
	req.Header.Set("User-Agent", "")

	ph := proto.HttpHeaderHolder{
		Masker: masking,
	}
	mrand.Read(ph.Rand[:])
	copy(ph.ConnID[:], pc.ID)
	ph.Time = time.Now().Unix()

	boostConnection := false

	v := ctx.Value(interfaces.ExtraOptionsHTTPTransportConnIsBoostConnection)
	if v != nil {
		boostConnection = v.(bool)
	}

	if boostConnection {
		ph.Flags |= proto.HttpHeaderFlag_BoostConnection
		fmt.Println("Boost Mark Set")
	}

	unival := ctx.Value(interfaces.ExtraOptionsUniConnAttrib)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		ph.ConnIter = uniAtt.Iter
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

	UseSystemProxy := false

	sysp := pc.ctx.Value(interfaces.ExtraOptionsHTTPUseSystemHTTPProxy)

	if sysp != nil {
		UseSystemProxy = sysp.(bool)
	}

	useSystemSocksProxy := false

	socksvp := pc.ctx.Value(interfaces.ExtraOptionsHTTPUseSystemSocksProxy)
	if socksvp != nil {
		useSystemSocksProxy = socksvp.(bool)
	}

	HTTPDialAddr := ""

	httpdialaddrvp := pc.ctx.Value(interfaces.ExtraOptionsHTTPDialAddr)

	if httpdialaddrvp != nil {
		HTTPDialAddr = httpdialaddrvp.(*interfaces.ExtraOptionsHTTPDialAddrValue).Addr
	}

	origDialContext := (&net.Dialer{Timeout: 30 * time.Second,
		KeepAlive: 30 * time.Second}).DialContext

	transport := &http.Transport{Proxy: func(request *http.Request) (url *url.URL, err error) {
		if UseSystemProxy {
			return http.ProxyFromEnvironment(request)
		}
		return nil, nil
	},
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
			Dialaddr := addr
			if HTTPDialAddr != "" {
				Dialaddr = HTTPDialAddr
			}

			if useSystemSocksProxy {
				dialerProxy := proxy.FromEnvironment()
				return dialerProxy.Dial(network, Dialaddr)
			} else {
				return origDialContext(ctx, network, Dialaddr)
			}
		},
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

func (pc *ProviderClient) DialRxConnection(ctx context.Context) {
	hc := pc.getHttpClient()

	masking, preader, pwriter := pc.prepareHTTP()

	go wrapper.ReceivePacketOverReader(masking, preader, pc.RxChan, ctx)

	req, err := http.NewRequest("GET", pc.HttpRequestEndpoint, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pc.reqprepare(req, masking, ctx)

	resp, err4 := hc.Do(req)
	if err4 != nil {
		fmt.Println(err4.Error())
		return
	}
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
	}
	io.Copy(pwriter, resp.Body)
	resp.Body.Close()
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
