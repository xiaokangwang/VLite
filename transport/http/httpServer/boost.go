package httpServer

import (
	"context"
	"fmt"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"time"
)

func (pcn *ProviderConnServerSide) BoostingListener(connctx context.Context) {
	ConnIDString := connidutil.ConnIDToString(connctx)
	BusTopic := ibusTopic.ConnBoostMode(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(connctx)

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
	go pcn.boostingWorker(boostModeOptChan, connctx)
}

func (pcn *ProviderConnServerSide) boostingWorker(infoc chan ibusInterface.ConnBoostMode, connctx context.Context) {
	boostEndTimer := time.NewTimer(time.Microsecond)

	//Discard First Timer signal
	<-boostEndTimer.C

	for {
		select {
		case infov := <-infoc:
			if infov.Enable {
				//We assume Client have a graceful period of 60 second, we close it 20 second later
				//so that client will not try to redial
				boostEndTimer.Reset(time.Duration(infov.TimeoutAtLeast+20) * time.Second)
			} else {
			cast:
				for {
					select {
					case pcn.boostConnectionGSRV.ShouldClose <- 0:
					default:
						break cast
					}
				}

			}

		case <-connctx.Done():
			return
		case <-boostEndTimer.C:
		cast2:
			for {
				select {
				case pcn.boostConnectionGSRV.ShouldClose <- 0:
				default:
					break cast2
				}
			}
		}
	}

}
