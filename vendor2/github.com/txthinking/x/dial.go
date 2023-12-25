package x

import (
	"net"
	"time"
)

// Dialer is a common interface for dialing
type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

// DefaultDial is the default dialer which dial with tcp network
type Dial struct {
	Timeout time.Duration
}

var DefaultDial = &Dial{
	Timeout: 10 * time.Second,
}

// Dial a remote address
func (d *Dial) Dial(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, d.Timeout)
}
