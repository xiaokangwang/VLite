package stackstub

import (
	"io"
	"io/ioutil"
)

type StackStub struct {
}

func (r *StackStub) HostWR(writer io.ReadWriter) {
	go func() {
		io.Copy(ioutil.Discard, writer)
	}()
}
