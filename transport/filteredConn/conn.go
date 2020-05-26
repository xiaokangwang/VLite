package filteredConn

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"github.com/xiaokangwang/VLite/proto"
	"io/ioutil"
	"log"
	"net"
	"sync/atomic"
	"time"
)

func NewFilteredConn(conn net.Conn,
	TxDataChannel chan interfaces.TrafficWithChannelTag,
	RxChannel chan interfaces.TrafficWithChannelTag,
	ctx context.Context) *FilteredConn {
	fc := &FilteredConn{
		conn:          conn,
		TxDataChannel: TxDataChannel,
		RxChannel:     RxChannel,
		ctx:           ctx,
	}
	fc.ctx, fc.ctxCancel = context.WithCancel(fc.ctx)
	go fc.WriteC()
	go fc.connectionBoostRequester()
	return fc
}

type FilteredConn struct {
	conn          net.Conn
	TxDataChannel chan interfaces.TrafficWithChannelTag
	RxChannel     chan interfaces.TrafficWithChannelTag

	ctx context.Context

	packetSent uint64
	packetRecv uint64

	ctxCancel context.CancelFunc
}

func (fc *FilteredConn) GetPacketStatus() (uint64, uint64) {
	sent := atomic.LoadUint64(&fc.packetSent)
	recv := atomic.LoadUint64(&fc.packetRecv)

	return sent, recv
}

//BLOCKING!!!
func (fc *FilteredConn) connectionBoostRequester() {
	sent, recv := fc.GetPacketStatus()

	msgbus := ibus.ConnectionMessageBusFromContext(fc.ctx)

	connstr := connidutil.ConnIDToString(fc.ctx)

	Boostchannel := ibusTopic.ConnBoostMode(connstr)

	msgbus.RegisterTopics(Boostchannel)

	ticker := time.NewTicker(time.Second)

	HiTrafficCounter := 0

	for {
		select {
		case <-fc.ctx.Done():
			ticker.Stop()
			fmt.Println("FilteredConn Done")
			return
		case <-ticker.C:
			sentn, recvn := fc.GetPacketStatus()
			//fmt.Println("FilteredConn", sentn, recvn)

			lastSecondTraffic := sent + recv

			thisSecondTraffic := sentn + recvn

			trafficFlow := thisSecondTraffic - lastSecondTraffic

			//fmt.Println("traffic flow:", trafficFlow)

			sent, recv = sentn, recvn

			if trafficFlow > 4 {
				HiTrafficCounter++
				if trafficFlow > 8 {
					HiTrafficCounter++
				}
				if trafficFlow > 16 {
					HiTrafficCounter++
				}

				if HiTrafficCounter > 3 {
					w := ibusInterface.ConnBoostMode{
						Enable:         true,
						TimeoutAtLeast: 60,
					}
					_, erremit := msgbus.Emit(fc.ctx, Boostchannel, w)
					if erremit != nil {
						fmt.Println(erremit.Error())
					}
				}
			} else {
				HiTrafficCounter = HiTrafficCounter / 2
				if HiTrafficCounter < 0 {
					HiTrafficCounter = 0
				}
			}
			//fmt.Println("HiTrafficCounter:", HiTrafficCounter)
		}
	}

}

func (fc *FilteredConn) LocalAddr() net.Addr {
	return fc.conn.LocalAddr()
}

func (fc *FilteredConn) RemoteAddr() net.Addr {
	return fc.conn.RemoteAddr()
}

func (fc *FilteredConn) SetDeadline(t time.Time) error {
	return fc.conn.SetDeadline(t)
}

func (fc *FilteredConn) SetReadDeadline(t time.Time) error {
	return fc.conn.SetReadDeadline(t)
}

func (fc *FilteredConn) SetWriteDeadline(t time.Time) error {
	return fc.conn.SetWriteDeadline(t)
}

func (fc *FilteredConn) Read(p []byte) (int, error) {
	var buffer [65536]byte
	for {
		n, err := fc.conn.Read(buffer[:])
		if err != nil {
			fmt.Println("Filter Conn Read: ", err.Error())
			fc.ctxCancel()
			return 0, err
		}

		atomic.AddUint64(&fc.packetRecv, 1)

		data := buffer[:n]

		datas := bytes.NewReader(data)

		dh := &proto.DataHeader{}
		err = struc.Unpack(datas, dh)
		if err != nil {
			log.Println(err)
		}
		s, _ := ioutil.ReadAll(datas)
		if dh.Channel == 0 {
			copy(p, s)
			return len(s), nil
		}
		RxData := interfaces.TrafficWithChannelTag{}
		RxData.Channel = dh.Channel
		RxData.Payload = s
		select {
		case fc.RxChannel <- RxData:
			break
		case <-fc.ctx.Done():
			fmt.Println("Filter Conn Ctx: ", fc.ctx.Err())
			return 0, fc.ctx.Err()
		}

	}

}

func (fc *FilteredConn) Write(p []byte) (int, error) {
	buf := bytes.NewBuffer(nil)
	datah := &proto.DataHeader{Channel: 0}

	err := struc.Pack(buf, datah)
	if err != nil {
		log.Println(err)
	}
	buf.Write(p)
	_, err2 := fc.conn.Write(buf.Bytes())
	atomic.AddUint64(&fc.packetSent, 1)
	if err2 == nil {
		return len(p), nil
	}
	return 0, err2
}

func (fc *FilteredConn) WriteC() {
	for {
		select {
		case data := <-fc.TxDataChannel:
			buf := bytes.NewBuffer(nil)
			datah := &proto.DataHeader{Channel: data.Channel}

			err := struc.Pack(buf, datah)
			if err != nil {
				log.Println(err)
			}
			_, err = buf.Write(data.Payload)
			if err != nil {
				log.Println(err)
			}

			_, err = fc.conn.Write(buf.Bytes())
			atomic.AddUint64(&fc.packetSent, 1)
			if err != nil {
				log.Println(err)
			}
			break
		case <-fc.ctx.Done():
			return
		}

	}

}

func (fc *FilteredConn) Close() error {
	return fc.conn.Close()
}
