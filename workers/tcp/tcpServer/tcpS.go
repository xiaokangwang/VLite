package tcpServer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/workers"
	"io"
	"log"
)

type TCPServer struct {
}

func (ts *TCPServer) RelayStream(conn io.ReadWriteCloser, ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	//fmt.Println("Relaying Stream")
	if conn == nil {
		log.Println("conn is nil, why?")
		return
	}
	s, done := ParseHeader(conn)
	if done {
		conn.Close()
		return
	}
	//fmt.Println("Relaying Stream2")
	netconn, err2 := workers.Dialer.Dial("tcp", s.DestDomain, s.DestPort, ctx)
	if err2 != nil {
		log.Println(err2)
		conn.Close()
		return
	}
	//fmt.Println("Relaying Stream3")
	go io.Copy(netconn, conn)
	io.Copy(conn, netconn)
	conn.Close()
	//fmt.Println("Relaying Stream4")

}

func ParseHeader(conn io.Reader) (*proto.StreamConnectDomainHeader, bool) {

	if conn == nil {
		log.Println("conn is nil, why??")
		return nil, true
	}

	var w [65536]byte
	l, err := conn.Read(w[:2])
	if err != nil {
		log.Println(err)
		return nil, true
	}

	swl := bytes.NewReader(w[:l])
	Headerl := &proto.StreamConnectDomainHeaderLen{}

	err = struc.Unpack(swl, Headerl)
	if err != nil {
		log.Println(err)
		return nil, true
	}

	datalen := Headerl.Length
	//Then, we will read exact that amount of data

	l, err = io.ReadFull(conn, w[:datalen])

	if err != nil {
		log.Println(err)
		return nil, true
	}

	sw := bytes.NewReader(w[:l])
	Header := &proto.CommandHeader{}

	err = struc.Unpack(sw, Header)
	if err != nil {
		log.Println(err)
		return nil, true
	}

	s := &proto.StreamConnectDomainHeader{}
	err = struc.Unpack(sw, s)
	if err != nil {
		log.Println(err)
		return nil, true
	}
	return s, false
}
