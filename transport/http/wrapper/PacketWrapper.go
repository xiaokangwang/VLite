package wrapper

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/proto"
	"io"
	"math/rand"
	"net/http"
	"time"
)

func ReceivePacketOverReader(lengthMask int64, reader io.Reader, sendingChan chan []byte, ctx context.Context) {
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

		if ctx.Err() != nil {
			fmt.Println("Connnection Closing,", ctx.Err().Error())

			//End Transmit

			if f, ok := reader.(*io.PipeReader); ok {
				_ = f
				f.Close()
			} else {
				//log.Println("Cannot flush writer")
			}

			if f, ok := reader.(io.ReadCloser); ok {
				_ = f
				f.Close()
			} else {
				//log.Println("Cannot flush writer")
			}

			if f, ok := reader.(io.Closer); ok {
				_ = f
				f.Close()
			} else {
				//log.Println("Cannot flush writer")
			}

			return
		}

	}

}
func SendPacketOverWriter(lengthMask int64, writer io.Writer, receivingChan chan []byte, networkBuffering int, ctx context.Context) {
	src := rand.NewSource(lengthMask)
	lenmaskSource := rand.New(src)
	fillInterval := time.Millisecond * 80
	timer := time.NewTimer(fillInterval)
	var bytesSendInThisInterval int
	var payloadSentInThisInterval bool

	done1, overfill1 := sendFill(8, lenmaskSource, writer)
	bytesSendInThisInterval = overfill1
	if done1 {
		return
	}

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
		case <-ctx.Done():
			fmt.Println("Connnection Closing,", ctx.Err().Error())
			return
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

func TestBufferSizePayload(sendingSize int, lenmaskSource *rand.Rand, writer io.Writer) {
	buf1 := make([]byte, sendingSize-1)
	buf2 := make([]byte, 1)

	io.ReadFull(lenmaskSource, buf1)
	io.ReadFull(lenmaskSource, buf2)

	//First Send

	io.Copy(writer, bytes.NewReader(buf1))

	if f, ok := writer.(http.Flusher); ok {
		_ = f
		f.Flush()
	} else {
		//log.Println("Cannot flush writer")
	}

	time.Sleep(4 * time.Second)

	io.Copy(writer, bytes.NewReader(buf2))

	if f, ok := writer.(http.Flusher); ok {
		_ = f
		f.Flush()
	} else {
		//log.Println("Cannot flush writer")
	}

	time.Sleep(6 * time.Second)

	if f, ok := writer.(*io.PipeWriter); ok {
		_ = f
		f.Close()
	} else {
		//log.Println("Cannot flush writer")
	}

	return
}

func TestBufferSizePayloadClient(sendingSize int, reader io.Reader, reqtime time.Time) int {
	var buf [65536]byte
	readen := 0

	n, err := reader.Read(buf[:])
	if err != nil && err != io.EOF {
		return -2
	}
	readen += n
	readentime := time.Now()

	if n < sendingSize {
		remainSize := sendingSize - readen
		_, err2 := io.ReadFull(reader, buf[:remainSize])
		if err2 != nil && err2 != io.EOF {
			return -2
		}
	}

	FullreadTime := time.Now()

	t1 := readentime.Sub(reqtime).Seconds()

	t2 := FullreadTime.Sub(readentime).Seconds()

	t3 := FullreadTime.Sub(reqtime).Seconds()

	if t1 < 3 {
		return ShorterThanExpected //Buffer Shorter than expected
	}

	if t2 <= 5 && t2 >= 3 {
		return Exact //Exact Buffer Size
	}

	if t3 > 9 {
		return LongerThanExpected //Buffer Longer than expected
	}

	return -1
}

const (
	ShorterThanExpected = 3
	Exact               = 1
	LongerThanExpected  = 2
)
