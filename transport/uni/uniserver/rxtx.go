package uniserver

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/transport/packetuni/puniCommon"
	"net"
	"time"
)

func (uct *UnifiedConnectionTransport) Rx(conn net.Conn, ctx context.Context) {
	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err.Error())
			return
		} else {
			uct.timeout.Reset(time.Second * 600)
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
			fmt.Println("Uni connTx Ended")
			return
		case <-uct.connctx.Done():
			return
		}
	}
}

func (uct *UnifiedConnectionTransport) Rehandshake() {
	puniCommon.ReHandshake2(uct.connctx, true)
}

func (uct *UnifiedConnectionTransport) timeoutWatcher() {
	<-uct.timeout.C
	uct.connCancel()
}
