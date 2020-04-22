package ibus

import (
	"context"
	"github.com/mustafaturan/bus"
	"github.com/mustafaturan/monoton"
	"github.com/mustafaturan/monoton/sequencer"
	"github.com/xiaokangwang/VLite/interfaces"
	"time"
)

func NewMessageBus() *bus.Bus {
	node := uint64(1)
	initialTime := time.Now().UnixNano()
	m, err := monoton.New(sequencer.NewMillisecond(), node, uint64(initialTime))
	if err != nil {
		panic(err)
	}

	// init an id generator
	var idGenerator bus.Next = (*m).Next

	// create a new ibus instance
	b, err := bus.NewBus(idGenerator)
	if err != nil {
		panic(err)
	}

	return b
}

func MessageBusFromContext(ctx context.Context) *bus.Bus {
	val := ctx.Value(interfaces.ExtraOptionsMessageBus)
	return val.(*bus.Bus)
}

func ConnectionMessageBusFromContext(ctx context.Context) *bus.Bus {
	val := ctx.Value(interfaces.ExtraOptionsMessageBusByConn)
	return val.(*bus.Bus)
}
