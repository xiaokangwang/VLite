package uniserver

import (
	"context"
	"fmt"
	"net"
)

func (uct *UnifiedConnectionTransport) Rx(conn net.Conn, ctx context.Context) {
	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		data := buf[:n]
		uct.RxChan <- data
		if ctx.Err() != nil {
			return
		}
		if uct.connctx.Err() != nil {
			return
		}
	}
}
func (uct *UnifiedConnectionTransport) Tx(conn net.Conn, ctx context.Context) {
	for {
		select {
		case data := <-uct.TxChan:
			_, err := conn.Write(data)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case <-ctx.Done():
			return
		case <-uct.connctx.Done():
			return
		}
	}
}
