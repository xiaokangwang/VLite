package tcpClient

import (
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/proto"
	"io"
	"log"
)

func WriteTcpDialHeader(w io.Writer, destHost string, destPort uint16) {
	Header := &proto.CommandHeader{}
	Header.CommandByte = proto.CommandByte_Stream_ConnectDomain
	s := &proto.StreamConnectDomainHeader{}
	s.DestDomain = destHost
	s.DestPort = destPort
	err := struc.Pack(w, Header)
	if err != nil {
		log.Println(err)
	}
	err = struc.Pack(w, s)
	if err != nil {
		log.Println(err)
	}
}
