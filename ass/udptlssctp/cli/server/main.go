package main

import (
	"context"
	"flag"
	"github.com/xiaokangwang/VLite/ass/licenseroll"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
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

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")

	flag.IntVar(&rateLimitMax, "rateLimitMax", 0, "")
	flag.IntVar(&rateLimitInit, "rateLimitInit", 0, "")
	flag.IntVar(&rateLimitSpeed, "rateLimitSpeed", 0, "")
	flag.BoolVar(&LicenseRollOnly, "LicenseRollOnly", false, "Show License and Credit")

	flag.Parse()

	if LicenseRollOnly {
		licenseroll.PrintLicense()
		os.Exit(0)
	}

	us := udptlssctp.NewUdptlsSctpServer(address, password, context.Background())

	us.Up()

	us.RateLimitTcpServerWrite(rateLimitSpeed, rateLimitMax, rateLimitInit)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
