package interfaces

import (
	"context"
	"net"
)

type SurrogateDialer interface {
	Dial(network, address string, port uint16, ctx context.Context) (net.Conn, error)
	NotifyMeltdown(reason error)
}

const ExtraOptionsLocalUDPBindPort = "ExtraOptionsLocalUDPBindPort"

type ExtraOptionsLocalUDPBindPortValue struct {
	LocalPort uint16
	LocalIP   *net.IP
}

type UDPPacket struct {
	Source  *net.UDPAddr
	Dest    *net.UDPAddr
	Payload []byte
}
