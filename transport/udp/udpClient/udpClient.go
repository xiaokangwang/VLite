package udpClient

import (
	"net"
)

func NewUdpClient(addr string) *udpClient {
	return &udpClient{dest: addr}
}

type udpClient struct {
	masking string
	dest    string
}

func (u *udpClient) Connect() (net.Conn, error) {
	conn, err := net.Dial("udp", u.dest)
	if err != nil {
		return nil, err
	}
	usageConn := conn
	//usageConn := masker2conn.NewMaskerAdopter(prependandxor.GetPrependAndXorMask(string(u.masking), []byte{0x1f, 0x0d}), conn)
	return usageConn , nil
}
