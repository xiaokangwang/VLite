package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
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

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")

	flag.IntVar(&rateLimitMax, "rateLimitMax", 0, "")
	flag.IntVar(&rateLimitInit, "rateLimitInit", 0, "")
	flag.IntVar(&rateLimitSpeed, "rateLimitSpeed", 0, "")

	flag.Parse()



	checksum := sha256.New()
	for i:=0 ; i<=3 ; i++{
		checksum.Write([]byte(password))

		checksum.Write([]byte("D2OxX6DDcB2pZ7WlyRHrZcZ1DfAUdrldhji1A"))

		checksum.Write([]byte(address))

		checksum.Write([]byte("D2OxX6DDcB2pZ7WlyRHrZcZ1DfAUdrldhji1B"))
	}


	bytse:=checksum.Sum([]byte(nil))

	out := fmt.Sprint(bytse)

	const Checking = ""

	if out != Checking{
		println(out)

		if Checking != "" {
			println("<?This is a testing software and cannot be run on unexpected environment. Please contact developer.?>")
			os.Exit(1)
		}
	}

	us := udptlssctp.NewUdptlsSctpServer(address, password, context.TODO())

	us.Up()

	us.RateLimitTcpServerWrite(rateLimitSpeed, rateLimitMax, rateLimitInit)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
