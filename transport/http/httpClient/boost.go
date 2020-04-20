package httpClient

import (
	"context"
	"fmt"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"time"
)

func (pc *ProviderClient) BoostingListener() {
	ConnIDString := connidutil.ConnIDToString(pc.connctx)
	BusTopic := ibusTopic.ConnBoostMode(ConnIDString)

	mbus := ibus.MessageBusFromContext(pc.connctx)

	mbus.RegisterTopics(BusTopic)

	boostModeOptChan := make(chan ibusInterface.ConnBoostMode, 8)

	mbus.RegisterHandler(BusTopic, &bus.Handler{
		Handle: func(e *bus.Event) {
			d := e.Data.(ibusInterface.ConnBoostMode)
			select {
			case boostModeOptChan <- d:
			default:
				fmt.Println("WARNING: boost mode hint discarded")
			}

		},
		Matcher: BusTopic,
	})

	go pc.boostWorker(boostModeOptChan)
}

func (pc *ProviderClient) boostWorker(info chan ibusInterface.ConnBoostMode) {
	boostEndTimer := time.NewTimer(time.Microsecond)

	//Discard First Timer signal
	<-boostEndTimer.C

	currentlyBoosting := false

	BoostConnectionCtx, cancelFunc := context.WithCancel(pc.connctx)

	for {
		select {
		case infoi := <-info:
			if infoi.Enable {
				Boosttime := infoi.TimeoutAtLeast
				if !currentlyBoosting {
					fmt.Println("Boosting Started", Boosttime)
					currentlyBoosting = true
					BoostConnectionCtx, cancelFunc = context.WithCancel(pc.connctx)
					go pc.boostConnStarter(BoostConnectionCtx)
				}
				boostEndTimer.Reset(time.Second * time.Duration(Boosttime))
				fmt.Println("Boost Time Recharged ", Boosttime)

			} else {
				currentlyBoosting = false
				cancelFunc()
			}
		case <-boostEndTimer.C:
			currentlyBoosting = false
			cancelFunc()
		case <-pc.connctx.Done():
			return
		}
	}

}

func (pc *ProviderClient) boostConnStarter(boostingconnctx context.Context) {

	boostingconnctx = context.WithValue(boostingconnctx,
		interfaces.ExtraOptionsHTTPTransportConnIsBoostConnection, true)

	var i int
	more := true
	for more {
		more = false
		if i < pc.MaxBoostRxConnection {
			go pc.DialRxConnectionD(boostingconnctx)
			more = true
		}
		//<-time.NewTimer(time.Second).C
		if i < pc.MaxBoostTxConnection {
			go pc.DialTxConnectionD(boostingconnctx)
			more = true
		}
		i++
		<-time.NewTimer(time.Millisecond * 300).C
	}
}
