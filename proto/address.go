package proto

import "net"

func IPv4ByteToAddr(in [4]byte)net.IP{
	return net.IP(in[:])
}

func IPv6ByteToAddr(in [16]byte)net.IP{
	return net.IP(in[:])
}

func IPv4AddrToByte(in net.IP)[4]byte{
	var addr [4]byte
	copy(addr[:],in.To4())
	return addr
}

func IPv6AddrToByte(in net.IP)[16]byte{
	var addr [16]byte
	copy(addr[:],in.To16())
	return addr
}