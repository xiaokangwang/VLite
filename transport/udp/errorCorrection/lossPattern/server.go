package lossPattern

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/lunixbochs/struc"
	"io"
	"net"
	"time"
)

func NewLossPatternServer(conn net.Conn) *LossPatternServer {
	lps := &LossPatternServer{
		SendInterval: 10,
		Size:         112,
		SendSum:      90000,
		conn:         conn,
	}
	go lps.Send()
	go lps.Receive()
	return lps
}

type LossPatternServer struct {
	SendInterval int
	Size         int
	SendSum      int
	conn         net.Conn
}

func (lps *LossPatternServer) Send() {
	for i := 0; i <= lps.SendSum; i++ {
		buf := bytes.NewBuffer(nil)
		lpsp := &LossPatternSignal{}
		lpsp.Seq = int64(i)
		lpsp.Time = time.Now().UnixNano()
		struc.Pack(buf, lpsp)
		fillsize := lps.Size - buf.Len()
		io.CopyN(buf, rand.Reader, int64(fillsize))
		lps.conn.Write(buf.Bytes())
		time.Sleep(time.Millisecond * time.Duration(lps.SendInterval))
	}

}

type LossPatternSignal struct {
	Seq  int64
	Time int64
}

func (lpc *LossPatternServer) Receive() {
	for {
		var reavbuf [65536]byte
		n, err := lpc.conn.Read(reavbuf[:])
		if err != nil {
			panic(err)
		}
		data := reavbuf[:n]
		d := bytes.NewReader(data)
		lpsp := &LossPatternSignal{}
		struc.Unpack(d, lpsp)
		fmt.Printf("%v %v\n", lpsp.Seq, lpsp.Time)

	}

}
