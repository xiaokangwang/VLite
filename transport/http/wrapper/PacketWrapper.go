package wrapper

import (
	"bytes"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/proto"
	"io"
	"math/rand"
	"net/http"
	"time"
)

func ReceivePacketOverReader(lengthMask int64, reader io.Reader, sendingChan chan []byte) {
	src := rand.NewSource(lengthMask)
	lenmaskSource := rand.New(src)

	for {
		l := &proto.HTTPLenHeader{}
		err := struc.Unpack(reader, l)
		if err != nil {
			println(err.Error())
			return
		}

		l.Length ^= lenmaskSource.Int63()

		if l.Length > 1601 {
			println("Rx Message Too Long")
			return
		}

		if l.Length == 0 {
			//Ignore Zero Length Message, they are used for Flushing buffer
			continue
		}

		buf := make([]byte, l.Length)

		_, err = io.ReadFull(reader, buf)
		if err != nil {
			println(err.Error())
			return
		}

		sendingChan <- buf

	}

}
func SendPacketOverWriter(lengthMask int64, writer io.Writer, receivingChan chan []byte, networkBuffering int) {
	src := rand.NewSource(lengthMask)
	lenmaskSource := rand.New(src)
	fillInterval := time.Millisecond * 20
	timer := time.NewTimer(fillInterval)
	var bytesSendInThisInterval int
	var payloadSentInThisInterval bool

	for {
		select {
		case sending := <-receivingChan:
			if sendPayload(sending, lenmaskSource, writer) {
				return
			}
			bytesSendInThisInterval += len(sending)
			payloadSentInThisInterval = true
		case <-timer.C:
			if networkBuffering == 0 {
				continue
			}
			if payloadSentInThisInterval {
				done, overfill := sendFill(networkBuffering-(networkBuffering+bytesSendInThisInterval%networkBuffering), lenmaskSource, writer)
				if done {
					return
				}
				bytesSendInThisInterval = overfill
				payloadSentInThisInterval = false
			}
			timer.Reset(fillInterval)
		}

	}

}
func sendFill(fillLength int, lenmaskSource *rand.Rand, writer io.Writer) (bool, int) {
	var filled = 0
	for fillLength < filled {
		l := &proto.HTTPLenHeader{}
		l.Length = 0
		l.Length ^= lenmaskSource.Int63()

		err := struc.Pack(writer, l)

		if err != nil {
			println(err.Error())
			return true, 0
		}
	}
	overfill := filled - fillLength

	if f, ok := writer.(http.Flusher); ok {
		_ = f
		f.Flush()
	} else {
		//log.Println("Cannot flush writer")
	}

	return false, overfill
}
func sendPayload(sending []byte, lenmaskSource *rand.Rand, writer io.Writer) bool {
	l := &proto.HTTPLenHeader{}
	l.Length = int64(len(sending))

	if len(sending) > 1601 {
		println("Tx Message Too Long")
		return true
	}

	l.Length ^= lenmaskSource.Int63()

	err := struc.Pack(writer, l)

	if err != nil {
		println(err.Error())
		return true
	}

	_, err = io.Copy(writer, bytes.NewReader(sending))

	if f, ok := writer.(http.Flusher); ok {
		_ = f
		f.Flush()
	} else {
		//log.Println("Cannot flush writer")
	}

	if err != nil {
		println(err.Error())
		return true
	}
	return false
}
