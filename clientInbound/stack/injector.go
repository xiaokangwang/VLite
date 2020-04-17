package stack

import (
	"bytes"
	"encoding/binary"
	"github.com/FlowerWrong/water"
	"net"
	"runtime"
)

//https://gist.github.com/chrisnc/0ff3d1c20cb6687454b0
//https://gist.github.com/xiaokangwang/a209ca4dd1acec0e2bd7c37f6a5a18ec

type iphdr struct {
	vhl   uint8
	tos   uint8
	iplen uint16
	id    uint16
	off   uint16
	ttl   uint8
	proto uint8
	csum  uint16
	src   [4]byte
	dst   [4]byte
}

type udphdr struct {
	src  uint16
	dst  uint16
	ulen uint16
	csum uint16
}

// pseudo header used for checksum calculation
type pseudohdr struct {
	ipsrc   [4]byte
	ipdst   [4]byte
	zero    uint8
	ipproto uint8
	plen    uint16
}

func checksum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func (h *iphdr) checksum() {
	h.csum = 0
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, h)
	h.csum = checksum(b.Bytes())
}

func (u *udphdr) checksum(ip *iphdr, payload []byte) {
	u.csum = 0
	phdr := pseudohdr{
		ipsrc:   ip.src,
		ipdst:   ip.dst,
		zero:    0,
		ipproto: ip.proto,
		plen:    u.ulen,
	}
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, &phdr)
	binary.Write(&b, binary.BigEndian, u)
	binary.Write(&b, binary.BigEndian, &payload)
	u.csum = checksum(b.Bytes())
}

func ForgeIPv4(data []byte,
	destaddr string,
	destport uint16,
	srcaddr string,
	srcport uint16) []byte {
	ip := iphdr{
		vhl: 0x45,
		tos: 0,
		id:  0xffff, // the kernel overwrites id if it is zero
		off: 0,
		ttl: 64,
		//proto: unix.IPPROTO_UDP,
		proto: 17,
	}
	ipsrc := net.ParseIP(srcaddr)
	ipdst := net.ParseIP(destaddr)
	copy(ip.src[:], ipsrc.To4())
	copy(ip.dst[:], ipdst.To4())
	udp := udphdr{
		src: uint16(srcport),
		dst: uint16(destport),
	}
	udplen := 8 + len(data)
	totalLen := 20 + udplen
	ip.iplen = uint16(totalLen)
	ip.checksum()
	udp.ulen = uint16(udplen)
	udp.checksum(&ip, data)
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, &ip)
	binary.Write(&b, binary.BigEndian, &udp)
	binary.Write(&b, binary.BigEndian, &data)
	bb := b.Bytes()
	if runtime.GOOS == "darwin" {
		bb[2], bb[3] = bb[3], bb[2]
	}
	return bb
}

func UDPInjector(ifce *water.Interface, inchan chan UDPPack) {
	for {
		pack := <-inchan
		res := ForgeIPv4(pack.payload, pack.addr, pack.port, pack.saddr, pack.sport)
		ifce.Write(res)
	}
}
