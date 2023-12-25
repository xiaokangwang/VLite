package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/txthinking/socks5"
	"github.com/xiaokangwang/VLite/ass/licenseroll"
	"github.com/xiaokangwang/VLite/ass/socksinterface"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"github.com/xiaokangwang/VLite/interfaces"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var password string
	var address string
	var addressL string

	var tunName string

	var LicenseRollOnly bool

	var UseSystemHTTPProxy bool
	var UseSystemSocksProxy bool
	var UseUDPPolyMasking bool

	var NetworkBuffering int

	var HTTPDialAddr string

	var TCPAlternativeChannelAddr string

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")
	flag.StringVar(&addressL, "AddressL", "", "")
	flag.StringVar(&tunName, "TunName", "tunvlite", "")
	flag.BoolVar(&LicenseRollOnly, "LicenseRollOnly", false, "Show License and Credit")
	flag.BoolVar(&UseSystemHTTPProxy, "UseSystemHTTPProxy", false, "Respect System HTTP Proxy Environment Var  HTTP_PROXY HTTPS_PROXY (apply to HTTP transport only)")
	flag.BoolVar(&UseSystemSocksProxy, "UseSystemSocksProxy", false, "Respect System Socks Proxy Environment Var ALL_PROXY (apply to HTTP transport only)")
	flag.IntVar(&NetworkBuffering, "NetworkBuffering", 0, "HTTP Network Buffering Amount(apply to HTTP transport only)")
	flag.StringVar(&HTTPDialAddr, "HTTPDialAddr", "", "If set, HTTP Connections will dial this address instead of requesting DNS result(apply to HTTP transport only)")
	flag.StringVar(&TCPAlternativeChannelAddr, "TCPAlternativeChannelAddr", "", "If set, TCP connection will dial this address over ws instead of over Datagram connection")
	flag.BoolVar(&UseUDPPolyMasking, "UseUDPPolyMasking", false, "Mask UDP packet to avoid matching")

	flag.Parse()

	if LicenseRollOnly {
		licenseroll.PrintLicense()
		os.Exit(0)
	}

	ctx := context.Background()

	if UseSystemHTTPProxy {
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPUseSystemHTTPProxy, true)
	}

	if UseSystemSocksProxy {
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPUseSystemSocksProxy, true)
	}

	if UseUDPPolyMasking {
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsUDPShouldMask, true)
	}

	if HTTPDialAddr != "" {
		ox := &interfaces.ExtraOptionsHTTPDialAddrValue{}
		ox.Addr = HTTPDialAddr
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPDialAddr, ox)
	}

	if NetworkBuffering != 0 {
		ctxv := &interfaces.ExtraOptionsHTTPNetworkBufferSizeValue{NetworkBufferSize: NetworkBuffering}
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPNetworkBufferSize, ctxv)
	}

	uc := udptlssctp.NewUdptlsSctpClient(address, password, tunName, ctx)

	if TCPAlternativeChannelAddr != "" {
		uc.AlternativeChannel(TCPAlternativeChannelAddr)
	}

	socks, err := socks5.NewClassicServer(addressL, "0.0.0.0", "", "", 0, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	socks.Handle = socksinterface.NewSocksHandler(uc, nil)
	go func() {
		fmt.Println(socks.RunTCPServer().Error())
	}()
	uc.Up()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

}
