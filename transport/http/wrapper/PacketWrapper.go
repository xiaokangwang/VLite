package wrapper

import (
	"bytes"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/proto"
	"io"
	"math/rand"
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

		buf := make([]byte, l.Length)

		_, err = io.ReadFull(reader, buf)
		if err != nil {
			println(err.Error())
			return
		}

		sendingChan <- buf

	}

}
func SendPacketOverWriter(lengthMask int64, writer io.Writer, receivingChan chan []byte) {
	src := rand.NewSource(lengthMask)
	lenmaskSource := rand.New(src)

	for {
		sending := <-receivingChan
		l := &proto.HTTPLenHeader{}
		l.Length = int64(len(sending))

		if len(sending) > 1601 {
			println("Tx Message Too Long")
			return
		}

		l.Length ^= lenmaskSource.Int63()

		err := struc.Pack(writer, l)

		if err != nil {
			println(err.Error())
			return
		}

		_, err = io.Copy(writer, bytes.NewReader(sending))

		if err != nil {
			println(err.Error())
			return
		}
	}

}
