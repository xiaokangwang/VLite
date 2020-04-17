package stack

import (
	"github.com/FlowerWrong/netstack/tcpip"
	"github.com/FlowerWrong/netstack/waiter"
	"net"
)

type NSTunnel struct {
	endpoint tcpip.Endpoint
	wq       *waiter.Queue
	mirror   net.Conn
}

func NewNSTunnel(endpoint tcpip.Endpoint,
	wq *waiter.Queue,
	mirror net.Conn) *NSTunnel {
	return &NSTunnel{endpoint: endpoint, wq: wq, mirror: mirror}
}

func (nt *NSTunnel) Start() {
	go nt.Rx()
	go nt.Tx()
}
func (nt *NSTunnel) Rx() {
	waitEntry, notifyCh := waiter.NewChannelEntry(nil)
	nt.wq.EventRegister(&waitEntry, waiter.EventIn)
	defer nt.wq.EventUnregister(&waitEntry)
	for {
		data, _, err := nt.endpoint.Read(nil)
		if err != nil {
			if err == tcpip.ErrWouldBlock {
				select {
				case <-notifyCh:
					continue
				}
			}
			nt.mirror.Close()
			nt.endpoint.Close()
			return
		}
	writeToMirror:
		for {
			n, erro := nt.mirror.Write(data)
			if erro != nil {
				nt.mirror.Close()
				nt.endpoint.Close()
				return
			}
			if n < len(data) {
				data = data[n:]
				continue writeToMirror
			}
			break writeToMirror
		}

	}
}
func (nt *NSTunnel) Tx() {

	for {
		buf := make([]byte, 1600)
		n, err := nt.mirror.Read(buf)
		if err != nil {
			nt.mirror.Close()
			nt.endpoint.Close()
			return
		}
		if n > 0 {
			chunk := buf[0:n]
		WriteIntoNS:
			for {
				if len(chunk) <= 0 {
					break WriteIntoNS
				}
				var m uintptr
				var err *tcpip.Error
				m, err = nt.endpoint.Write(tcpip.SlicePayload(chunk), tcpip.WriteOptions{})
				n := int(m)
				if err != nil {
					if err == tcpip.ErrWouldBlock {
						if n < len(chunk) {
							chunk = chunk[n:]
							continue WriteIntoNS
						}
					}
					return
				} else if n < len(chunk) {
					chunk = chunk[n:]
					continue WriteIntoNS
				} else {
					break WriteIntoNS
				}
			}
		}

	}
}