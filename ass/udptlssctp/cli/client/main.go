package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/txthinking/socks5"
	"github.com/xiaokangwang/VLite/ass/socksinterface"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var password string
	var address string
	var addressL string

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")
	flag.StringVar(&addressL, "AddressL", "", "")
	flag.Parse()
	uc := udptlssctp.NewUdptlsSctpClient(address, password, context.Background())
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
