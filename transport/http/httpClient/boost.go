package httpClient

import (
	"context"
	"fmt"
	"github.com/golang-collections/go-datastructures/queue"
	"github.com/mustafaturan/bus"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
	"sync/atomic"
	"time"
)

func (pc *ProviderClient) BoostingListener() {
	ConnIDString := connidutil.ConnIDToString(pc.connctx)
	BusTopic := ibusTopic.ConnBoostMode(ConnIDString)

	mbus := ibus.ConnectionMessageBusFromContext(pc.connctx)

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

	TxBoostExpectedSize := make(chan int, 8)
	RxBoostExpectedSize := make(chan int, 8)

	go pc.boostConnScaleMgr(pc.connctx, RxBoostExpectedSize, false)
	go pc.boostConnScaleMgr(pc.connctx, TxBoostExpectedSize, true)

	TxBoostCurrentSize := 0
	RxBoostCurrentSize := 0

	for {
		select {
		case infoi := <-info:
			if infoi.Enable {
				Boosttime := infoi.TimeoutAtLeast
				if !currentlyBoosting {
					fmt.Println("Boosting Started", Boosttime)
					currentlyBoosting = true
					TxBoostExpectedSize <- pc.MaxBoostTxConnection / 8
					TxBoostCurrentSize = pc.MaxBoostTxConnection / 8

					RxBoostExpectedSize <- pc.MaxBoostRxConnection / 8
					RxBoostCurrentSize = pc.MaxBoostRxConnection / 8
				}
				boostEndTimer.Reset(time.Second * time.Duration(Boosttime))
				fmt.Println("Boost Time Recharged ", Boosttime)

				if pc.MaxBoostTxConnection > TxBoostCurrentSize {
					TxBoostCurrentSize += 1
					TxBoostExpectedSize <- TxBoostCurrentSize
				}

				if pc.MaxBoostRxConnection > RxBoostCurrentSize {
					RxBoostCurrentSize += 1
					RxBoostExpectedSize <- RxBoostCurrentSize
				}

			} else {

				currentlyBoosting = false
				TxBoostExpectedSize <- 0
				RxBoostExpectedSize <- 0
			}
		case <-boostEndTimer.C:
			currentlyBoosting = false
			TxBoostExpectedSize <- 0
			RxBoostExpectedSize <- 0
		case <-pc.connctx.Done():
			return
		}
	}

}
func (pc *ProviderClient) boostConnScaleMgr(boostingconnctx context.Context, expectedSize chan int, isTx bool) {
	boostingconnctx = context.WithValue(boostingconnctx,
		interfaces.ExtraOptionsHTTPTransportConnIsBoostConnection, true)

	cancelQueue := queue.NewRingBuffer(uint32(pc.MaxBoostRxConnection + pc.MaxTxConnection + 10))
	downscaleTimer := time.NewTicker(time.Second * time.Duration(15))
	upscaleTimer := time.NewTicker(time.Second * time.Duration(1))
	expectedSizeLast := 0

	//we assume server will close OUR Rx to client connection with 20 seconds of delay
	//and we have a 60 seconds of grace period
	//in which we don't redial, but keep connection opened
	//so that server can close it gracefully
	RxGracePeriodCharges := 4

	RedialValue := &interfaces.ExtraOptionsBoostConnectionShouldNotRedialValue{ShouldNotReDial: 1}

	for {
		select {
		case <-pc.connctx.Done():
			return
		case <-downscaleTimer.C:
			if cancelQueue.Len() > uint32(expectedSizeLast) {
				atomic.StoreInt32(&RedialValue.ShouldNotReDial, 1)
				if RxGracePeriodCharges <= 0 || isTx {
					s, _ := cancelQueue.Get()
					s.(context.CancelFunc)()
				} else {
					RxGracePeriodCharges--
				}
			}
		case currentExpected := <-expectedSize:
			expectedSizeLast = currentExpected
		case <-upscaleTimer.C:
			if uint32(expectedSizeLast) > cancelQueue.Len() {
				atomic.StoreInt32(&RedialValue.ShouldNotReDial, 0)
				RxGracePeriodCharges = 4
				thisctx, cancel := context.WithCancel(boostingconnctx)
				thisctx = context.WithValue(thisctx,
					interfaces.ExtraOptionsBoostConnectionShouldNotRedial,
					RedialValue)
				if pc.useWebsocket {
					go pc.DialWsConnectionD(thisctx)
				} else {
					if isTx {
						go pc.DialTxConnectionD(thisctx)
					} else {
						go pc.DialRxConnectionD(thisctx)
					}
				}
				cancelQueue.Put(cancel)
			}
		}
	}
}
