package main

import (
	"context"
	"flag"
	"github.com/xiaokangwang/VLite/transport/udp/errorCorrection/lossPattern"
	"github.com/xiaokangwang/VLite/transport/udp/udpClient"
	"github.com/xiaokangwang/VLite/transport/udp/udpServer"
	"net"
	"time"
)

type sr struct {
}

func (s sr) Connection(conn net.Conn, ctx context.Context) context.Context {
	server := lossPattern.NewLossPatternServer(conn)
	time.Sleep(20 * time.Minute)
	_ = server
	return ctx
}

func main() {
	var remoteAddress string
	var Listen bool

	flag.StringVar(&remoteAddress, "Address", "", "")
	flag.BoolVar(&Listen, "Listen", false, "Listen")
	flag.Parse()

	if Listen {
		udpServer.NewUDPServer(remoteAddress, context.TODO(), &sr{})
	} else {
		c, err, _ := udpClient.NewUdpClient(remoteAddress, context.TODO()).Connect(context.Background())
		if err != nil {
			panic(err)
		}
		lossPattern.NewLossPatternClient(c).Receive()
	}

	time.Sleep(40 * time.Minute)
}
