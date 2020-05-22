package client

import (
	"fmt"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"time"
)

func (ucc *UDPClientContext) boostingReceiver() {
	ConnIDString := connidutil.ConnIDToString(ucc.context)
	BusTopic := ibusTopic.ConnBoostMode(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(ucc.context)

	mbus.RegisterTopics(BusTopic)

	boostModeOptChan := make(chan ibusInterface.ConnBoostMode, 8)

	mbus.RegisterHandler(BusTopic+"ClientWorker", &bus.Handler{
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
	go ucc.boostWorker(boostModeOptChan)
}

func (ucc *UDPClientContext) boostWorker(info chan ibusInterface.ConnBoostMode) {
	boostEndTimer := time.NewTimer(time.Microsecond)

	//Discard First Timer signal
	<-boostEndTimer.C

	currentlyBoosting := false

	for {
		select {
		case infoi := <-info:
			if infoi.Enable {
				Boosttime := infoi.TimeoutAtLeast
				boostEndTimer.Reset(time.Second * time.Duration(Boosttime))
				currentlyBoosting = true
			}
		case <-boostEndTimer.C:
			currentlyBoosting = false
		case <-ucc.context.Done():
			return
		}
		ucc.isBoosted = currentlyBoosting
	}
}

func (ucc *UDPClientContext) AggressivePingBegin() {
	ucc.isAggressivePingRequested = true
}
