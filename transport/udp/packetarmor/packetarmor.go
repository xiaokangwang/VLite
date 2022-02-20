package packetarmor

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/lunixbochs/struc"
	"github.com/secure-io/siv-go"
	"golang.org/x/crypto/sha3"
)

func NewPacketArmor(password, salt string, isClient bool) *PacketArmor {
	return &PacketArmor{password: password, salt: salt, isClient: isClient}
}

type PacketArmor struct {
	password string
	salt     string
	isClient bool
}

func (a *PacketArmor) getKey(isUp bool) (cipher.AEAD, error) {
	hasher := sha3.NewCShake128(nil, []byte(a.salt))
	hasher.Write([]byte(a.password))

	if isUp == a.isClient {
		hasher.Write([]byte("U"))
	} else {
		hasher.Write([]byte("D"))
	}

	keyin := make([]byte, 64)

	io.ReadFull(hasher, keyin[:])

	aeadBlock, err2 := siv.NewCMAC(keyin)
	if err2 != nil {
		return nil, err2
	}
	return aeadBlock, err2
}

var errLengthTooShort = errors.New("pack length too short")

func (a *PacketArmor) Pack(input []byte, length int) ([]byte, error) {
	key, err := a.getKey(true)
	if err != nil {
		return nil, err
	}
	padding := length - key.Overhead() - key.NonceSize() - len(input) - 4
	if padding < 0 {
		return nil, err
	}
	var paddingData [1600]byte
	data := &armoredPacket{
		Data:    input,
		DataLen: len(input),
		Padding: paddingData[:padding],
	}
	outdata := bytes.NewBuffer(nil)
	if err := struc.Pack(outdata, data); err != nil {
		return nil, err
	}
	nonce := make([]byte, key.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	encryptedData := key.Seal(nil, nonce, outdata.Bytes(), nil)
	encryptedOutData := bytes.NewBuffer(nil)

	if _, err := io.Copy(encryptedOutData, bytes.NewReader(nonce)); err != nil {
		return nil, err
	}
	if _, err := io.Copy(encryptedOutData, bytes.NewReader(encryptedData)); err != nil {
		return nil, err
	}
	return encryptedOutData.Bytes(), nil
}

func (a *PacketArmor) Unpack(encryptedInput []byte) ([]byte, error) {
	key, err := a.getKey(false)
	if err != nil {
		return nil, err
	}
	if len(encryptedInput) <= 17 {
		return nil, errLengthTooShort
	}
	decryptedData, err := key.Open(nil, encryptedInput[:16], encryptedInput[16:], nil)
	if err != nil {
		return nil, err
	}
	container := &armoredPacket{}
	if err := struc.Unpack(bytes.NewReader(decryptedData), container); err != nil {
		return nil, err
	}
	return container.Data, nil
}

type armoredPacket struct {
	DataLen    int    `struc:"uint16,sizeof=Data"`
	Data       []byte `struc:"[]byte"`
	PaddingLen int    `struc:"uint16,sizeof=Padding"`
	Padding    []byte `struc:"[]byte"`
}
