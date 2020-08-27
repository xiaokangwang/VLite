package main

import (
	"context"
	"flag"
	"github.com/xiaokangwang/VLite/ass/licenseroll"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"github.com/xiaokangwang/VLite/interfaces"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var password string
	var address string

	var rateLimitMax int
	var rateLimitInit int
	var rateLimitSpeed int
	var LicenseRollOnly bool

	var NetworkBuffering int

	var UseUDPPolyMasking bool

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")

	flag.IntVar(&rateLimitMax, "rateLimitMax", 0, "")
	flag.IntVar(&rateLimitInit, "rateLimitInit", 0, "")
	flag.IntVar(&rateLimitSpeed, "rateLimitSpeed", 0, "")
	flag.BoolVar(&LicenseRollOnly, "LicenseRollOnly", false, "Show License and Credit")

	flag.IntVar(&NetworkBuffering, "NetworkBuffering", 0, "HTTP Network Buffering Amount(apply to HTTP transport only)")

	flag.BoolVar(&UseUDPPolyMasking, "UseUDPPolyMasking", false, "Mask UDP packet to avoid matching")

	flag.Parse()

	if LicenseRollOnly {
		licenseroll.PrintLicense()
		os.Exit(0)
	}

	ctx := context.Background()
	if NetworkBuffering != 0 {
		ctxv := &interfaces.ExtraOptionsHTTPNetworkBufferSizeValue{NetworkBufferSize: NetworkBuffering}
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPNetworkBufferSize, ctxv)
	}

	if UseUDPPolyMasking {
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsUDPShouldMask, true)
	}

	us := udptlssctp.NewUdptlsSctpServer(address, password, ctx)

	us.Up()

	us.RateLimitTcpServerWrite(rateLimitSpeed, rateLimitMax, rateLimitInit)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
