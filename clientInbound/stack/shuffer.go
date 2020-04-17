package stack

import (
	"context"
	"github.com/xiaokangwang/VLite/interfaces"
	"net"
	"sync"
)

type Shuffler struct {
	conn   sync.Map
	dialer interfaces.SurrogateDialer
	inchan chan UDPPack
}

func NewShuffler(dialer interfaces.SurrogateDialer,
	inchan chan UDPPack) *Shuffler {
	return &Shuffler{dialer: dialer, inchan: inchan}
}

type UDPPack struct {
	payload []byte
	addr    string
	port    uint16
	saddr   string
	sport   uint16
}

func (s *Shuffler) Progress(data []byte, destaddr string, destport uint16, srcaddr string, srcport uint16) {
	//Check if local port have corresponding socket
	//TODO also check if local addr
	obj, ok := s.conn.Load(srcport)
	//log.Print(destaddr, destport, srcaddr, srcport)
	var ct net.Conn
	if !ok {
		//Dial one since there is none
		conn, err := s.dialer.Dial("udp", destaddr, destport, context.TODO())
		if err != nil {
			return
		}
		ct = conn
		s.conn.Store(srcport, conn)
		go s.ConnTrackerRx(conn, s.inchan, destaddr, destport, srcaddr, srcport)
	} else {
		ct = obj.(net.Conn)
	}
	s.ConnTrackerTx(ct, data)

}

func (s *Shuffler) ConnTrackerRx(conn net.Conn, inchan chan (UDPPack), destaddr string, destport uint16, srcaddr string, srcport uint16) {
	for {
		buf := make([]byte, 1800)
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			s.conn.Delete(srcport)
			return
		}
		p := UDPPack{payload: buf[0:n], saddr: destaddr, sport: destport, addr: srcaddr, port: srcport}
		inchan <- p
	}
}

func (s *Shuffler) ConnTrackerTx(conn net.Conn, data []byte) {
	go conn.Write(data)
}
