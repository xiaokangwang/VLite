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
	hp := &HttpHeaderHolderProcessor{password: password, salt: "HTTPHeaderSecret"}
	hp.prepare()
	return hp
}

func NewHttpHeaderHolderProcessor2(password string, salt string) *HttpHeaderHolderProcessor {
	hp := &HttpHeaderHolderProcessor{password: password, salt: salt}
	hp.prepare()
	return hp
}

type HttpHeaderHolderProcessor struct {
	password string
	salt     string
}

func (pc *HttpHeaderHolderProcessor) prepare() cipher.AEAD {

	hasher := sha3.NewCShake128(nil, []byte(pc.salt))
	hasher.Write([]byte(pc.password))

	keyin := make([]byte, 64)

	io.ReadFull(hasher, keyin[:])

	aeadBlock, err2 := siv.NewCMAC(keyin)

	if err2 != nil {
		fmt.Println(err2.Error())
	}
	return aeadBlock
}
func (pc *HttpHeaderHolderProcessor) Open(input string) *proto.HttpHeaderHolder {
	inputb := bytes.NewBufferString(input)
	decodeReader := base64.NewDecoder(base64.URLEncoding, inputb)
	cont, err := ioutil.ReadAll(decodeReader)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	opened, erro := pc.prepare().Open(nil, nil, cont, nil)
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
	err = struc.Pack(buf, &src)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	sealed := pc.prepare().Seal(nil, nil, buf.Bytes(), nil)

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
