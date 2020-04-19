package headerHolder

import (
	"bytes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/secure-io/siv-go"
	"github.com/xiaokangwang/VLite/proto"
	"golang.org/x/crypto/sha3"
	"io"
	"io/ioutil"
)

func NewHttpHeaderHolderProcessor(password string) *HttpHeaderHolderProcessor {
	hp := &HttpHeaderHolderProcessor{password: password}
	hp.prepare()
	return hp
}

type HttpHeaderHolderProcessor struct {
	password  string
	aeadBlock cipher.AEAD
}

func (pc *HttpHeaderHolderProcessor) prepare() {
	var err error

	hasher := sha3.NewCShake128(nil, []byte("HTTPHeaderSecret"))
	hasher.Write([]byte(pc.password))

	keyin := make([]byte, 64)

	io.ReadFull(hasher, keyin[:])

	pc.aeadBlock, err = siv.NewCMAC(keyin)

	if err != nil {
		fmt.Println(err.Error())
	}
}
func (pc *HttpHeaderHolderProcessor) Open(input string) *proto.HttpHeaderHolder {
	inputb := bytes.NewBufferString(input)
	decodeReader := base64.NewDecoder(base64.URLEncoding, inputb)
	cont, err := ioutil.ReadAll(decodeReader)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	opened, erro := pc.aeadBlock.Open(nil, nil, cont, nil)
	if erro != nil {
		fmt.Println(err)
		return nil
	}
	openedReader := bytes.NewReader(opened)

	HeaderHolder := &proto.HttpHeaderHolder{}
	err = struc.Unpack(openedReader, HeaderHolder)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return HeaderHolder
}
func (pc *HttpHeaderHolderProcessor) Seal(src proto.HttpHeaderHolder) string {
	var err error
	buf := bytes.NewBuffer(nil)
	err = struc.Pack(buf, src)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	sealed := pc.aeadBlock.Seal(nil, nil, buf.Bytes(), nil)

	obuf := bytes.NewBuffer(nil)
	decodeReader := base64.NewEncoder(base64.URLEncoding, obuf)
	_, err = decodeReader.Write(sealed)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	err = decodeReader.Close()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return obuf.String()
}
