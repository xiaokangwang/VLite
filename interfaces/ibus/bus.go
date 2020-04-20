package ibus

import (
	"github.com/mustafaturan/bus"
	"github.com/mustafaturan/monoton"
	"github.com/mustafaturan/monoton/sequencer"
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
