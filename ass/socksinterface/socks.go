package socksinterface

import (
	"bytes"
	"fmt"
	"github.com/txthinking/socks5"
	"github.com/xiaokangwang/VLite/ass/udpconn2tun"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func NewSocksHandler(uc udptlssctp.TCPSocketDialer, udpa *udpconn2tun.UDPConn2Tun) *SocksHandler {
	return &SocksHandler{uc: uc, udpa: udpa, udpassoc: new(sync.Map)}
}

type SocksHandler struct {
	singleClientInstance bool

	uc udptlssctp.TCPSocketDialer

	udpa *udpconn2tun.UDPConn2Tun

	udpassoc *sync.Map
}

type UDPExchange struct {
	ClientAddr *net.UDPAddr
	RemoteConn net.PacketConn
	MsgCount   int64
}

func (so SocksHandler) TCPHandle(sr *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	s := &so
	sro := sr
	if r.Cmd == socks5.CmdUDP && s.udpa != nil {

		addr, reserr := net.ResolveUDPAddr("udp", r.Address())

		if reserr != nil {
			fmt.Println(reserr.Error())
			return socks5.ErrBadRequest
		}

		caddr := addr

		if addr.Port == 0 {
			fmt.Println("Client did not send its bind port, we need to open an new udp listener.")
			newServerd, err := socks5.NewClassicServer("0.0.0.0:0", sr.UDPAddr.IP.String(), "", "", 0, 0, 0, 0)
			if err != nil {
				fmt.Println(err.Error())
				return socks5.ErrBadRequest
			}

			hld := &SocksHandler{
				singleClientInstance: true,
				uc:                   s.uc,
				udpa:                 s.udpa,
				udpassoc:             new(sync.Map),
			}
			newServerd.Handle = hld

			go func() {
				newServerd.RunUDPServer()
			}()

			fmt.Println("Rewrite routine")

			sr = newServerd

			s = hld
		}
		dialaddr := *caddr
		if dialaddr.Port == 0 {
			dialaddr.Port = c.RemoteAddr().(*net.TCPAddr).Port
		}
		udpconn := s.udpa.DialUDP(dialaddr)

		uex := &UDPExchange{
			ClientAddr: caddr,
			RemoteConn: udpconn,
		}

		s.udpassoc.Store(caddr.Port, uex)

		if sr.UDPConn == nil {
			//not ready yet, wait for it so that it can become ready
			runtime.Gosched()

		}
		for sr.UDPConn == nil {
			//It still not ready yet, wait until it is ready
			<-time.NewTimer(time.Millisecond * 100).C
		}

		uladdr := sr.UDPConn.LocalAddr()
		uladdr.(*net.UDPAddr).IP = sro.UDPAddr.IP

		aR, addrR, portR, errR := socks5.ParseAddress(uladdr.String())

		if errR != nil {
			fmt.Println(errR.Error())
			return socks5.ErrBadRequest
		}



		p := socks5.NewReply(socks5.RepSuccess, aR, addrR, portR)

		var buf bytes.Buffer
		if _, err := p.WriteTo(&buf); err != nil {
			return socks5.ErrBadRequest
		}

		io.Copy(c, bytes.NewReader(buf.Bytes()))

		//Wait for connection to end
		io.Copy(ioutil.Discard,c)

		s.udpassoc.Delete(caddr.Port)
		udpconn.Close()

		if addr.Port == 0 {
			sr.Shutdown()
		}

		return nil
	}

	if r.Cmd == socks5.CmdConnect {
		p := socks5.NewReply(socks5.RepSuccess, socks5.ATYPIPv4, []byte{0x00, 0x00, 0x00, 0x00}, []byte{0x00, 0x00})
		p.WriteTo(c)

		hp, pp, err01 := net.SplitHostPort(r.Address())

		if err01 != nil {
			println(err01.Error())
			c.Close()
			return nil
		}

		conn, err0 := s.uc.DialDirect(hp, func() uint16 { i, _ := strconv.Atoi(pp); return uint16(i) }())

		if err0 != nil {
			println(err0.Error())
			c.Close()
			return nil
		}

		go io.Copy(conn, c)

		_, err := io.Copy(c, conn)

		if err != nil {
			println(err.Error())
		}

		err = c.Close()
		if err != nil {
			println(err.Error())
		}
		err = conn.Close()
		if err != nil {
			println(err.Error())
		}
		return nil
	}

	return socks5.ErrUnsupportCmd
}

func (s SocksHandler) UDPHandle(sr *socks5.Server, addr *net.UDPAddr, dg *socks5.Datagram) error {

	loadPort := addr.Port

	if s.singleClientInstance {
		loadPort = 0
	}

	v, ok := s.udpassoc.Load(loadPort)

	if !ok {
		fmt.Println("Orphan packet received")
		fmt.Println(addr.String())
		return nil
	}

	ue := v.(*UDPExchange)

	newval := atomic.AddInt64(&ue.MsgCount, 1)

	if newval == 1 {
		go s.udpRxLoop(ue.RemoteConn, addr, sr)
	}

	ue.ClientAddr = addr

	hp, pp, err01 := net.SplitHostPort(dg.Address())

	if err01 != nil {
		return nil
	}

	addr2 := net.UDPAddr{
		IP:   net.ParseIP(hp),
		Port: func() int { i, _ := strconv.Atoi(pp); return int(i) }(),
		Zone: "",
	}

	ue.RemoteConn.WriteTo(dg.Data, &addr2)

	return nil
}

func (s SocksHandler) udpRxLoop(mirror net.PacketConn, dest *net.UDPAddr, sr *socks5.Server) {
	var b [65536]byte
	for {
		n, addr, err := mirror.ReadFrom(b[:])
		if err != nil {
			sr.UDPReleasesTCP(dest)
			return
		}

		_ = addr

		payload := b[:n]

		a, raddr, rport, err2 := socks5.ParseAddress(addr.String())

		if err2 != nil {
			return
		}

		d1 := socks5.NewDatagram(a, raddr, rport, payload)

		_, err3 := sr.UDPConn.WriteTo(d1.Bytes(), dest)

		if err3 != nil {
			return
		}
	}
}
