package udpClient

import "net"

func NewUdpClient(addr string) *udpClient {
	return &udpClient{dest:addr}
}

type udpClient struct {
	dest string
}

func (u *udpClient) Connect() (net.Conn, error) {
	return net.Dial("udp", u.dest)
}
