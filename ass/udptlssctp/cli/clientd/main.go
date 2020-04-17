package main

import (
	"context"
	"flag"
	"github.com/txthinking/socks5"
	"github.com/xiaokangwang/VLite/ass/socksinterface"
	"github.com/xiaokangwang/VLite/ass/udpconn2tun"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var password string
	var address string
	var addressL string

	//flag.StringVar(&password, "Password", "", "")
	//flag.StringVar(&address, "Address", "", "")
	flag.StringVar(&addressL, "AddressL", "127.0.0.1:1988", "")
	flag.Parse()
	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, context.TODO())
	uc.Up()

	connadp := udpconn2tun.NewUDPConn2Tun(uc.TunnelTxToTun, uc.TunnelRxFromTun)

	socks, err := socks5.NewClassicServer(addressL, "127.0.0.1", "", "", 0, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	socks.Handle = socksinterface.NewSocksHandler(uc,connadp)
	go socks.RunTCPServer()
	go socks.RunUDPServer()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

}

