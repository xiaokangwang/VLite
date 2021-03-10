package workers

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"net"
	"strings"
)

const UDPReadingTimeout = 1200

var Dialer interfaces.SurrogateDialer

type systemDialer struct {
}

func (s systemDialer) Dial(network, address string, port uint16, ctx context.Context) (net.Conn, error) {
	dialer := net.Dialer{}

	extraOptionsLocalUDPBindPortR := ctx.Value(interfaces.ExtraOptionsLocalUDPBindPort)
	if extraOptionsLocalUDPBindPortR != nil {
		extraOptionsLocalUDPBindPort := extraOptionsLocalUDPBindPortR.(interfaces.ExtraOptionsLocalUDPBindPortValue)

		l, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(extraOptionsLocalUDPBindPort.LocalPort)})
		if err != nil {
			// If we can't listen on that port, try another port as well
			l2, err := net.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				return nil, err
			}
			return l2, err
		}
		return l, err
	}

	if strings.Contains(address, ":") {
		address = "[" + address + "]"
	}

	uconn, err := dialer.DialContext(ctx, network, fmt.Sprintf("%v:%v", address, port))
	if err != nil {
		return nil, err
	}
	return uconn, nil
}

func (s systemDialer) NotifyMeltdown(reason error) {
	panic("implement me")
}

func init() {
	Dialer = &systemDialer{}
}
