package uniserver

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/antiReplayWindow"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	"net"
)

func (uic *UnifiedConnectionTransportHub) onConnection(conn net.Conn, ctx context.Context) context.Context {
	//We needs to get detail from ctx

	Attrib := ctx.Value(interfaces.ExtraOptionsUniConnAttrib).(*interfaces.ExtraOptionsUniConnAttribValue)

	//Construct Default Connection structure
	uCT := &UnifiedConnectionTransport{}
	uCT.TxChan = make(chan []byte, 8)
	uCT.RxChan = make(chan []byte, 8)
	uCT.ConnID = Attrib.ID
	uCT.LastConnIter = Attrib.Iter
	uCT.Arw = antiReplayWindow.NewAntiReplayWindow(120)
	uCT.LastCOnnIterCancelFunc = make(map[string]context.CancelFunc)
	uCT.connctx = ctx

	act, ctl := uic.Conns.LoadOrStore(string(Attrib.ID), uCT)

	if ctl {
		uCT = act.(*UnifiedConnectionTransport)
	} else {
		//ConnID should have been set
		uic.Uplistener.Connection(adp.NewRxTxToConn(uCT.TxChan, uCT.RxChan, uCT), ctx)
	}

	if !uCT.Arw.Check(Attrib.Rand) {
		//This connection Shall be discarded
		conn.Close()
		return nil
	}

	//Check Iter

	if Attrib.Iter > uCT.LastConnIter {
		for _, v := range uCT.LastCOnnIterCancelFunc {
			v()
		}
		uCT.LastCOnnIterCancelFunc = make(map[string]context.CancelFunc)
		fmt.Println("Connection Reincarnation")
	}
	thisconnctx, cancel := context.WithCancel(uCT.connctx)
	uCT.LastCOnnIterCancelFunc[string(Attrib.Rand)] = cancel
	uCT.LastConnIter = Attrib.Iter

	go uCT.Rx(conn, thisconnctx)
	go uCT.Tx(conn, thisconnctx)

	return thisconnctx

}
