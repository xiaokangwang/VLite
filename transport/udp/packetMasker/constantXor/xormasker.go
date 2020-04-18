package constantXor

import (
	"bytes"
	"golang.org/x/crypto/sha3"
	"io"
)

type XorMasker struct {
	MaskingPattern []byte
	MaskingSeed    string
}

func NewXorMasker(masker string) *XorMasker {
	xm := &XorMasker{
		MaskingSeed: masker,
	}
	xm.prepare()
	return xm
}

func (xm *XorMasker) prepare() {
	Seeder := sha3.NewCShake128(nil, []byte("XORMaskingSeed"))

	Seeder.Write([]byte(xm.MaskingSeed))
	var maskerpattern bytes.Buffer
	io.Copy(&maskerpattern, io.LimitReader(Seeder, 65536))

	xm.MaskingPattern = maskerpattern.Bytes()
}
func (xm *XorMasker) maskInternal(input io.Reader, output io.Writer) error {
	var counter = 0
	var buf [1]byte
	for {
		_, err := input.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		buf[0] ^= xm.MaskingPattern[counter]
		_, err = output.Write(buf[:])

		if err != nil {
			return err
		}
	}
}

func (xm *XorMasker) Mask(input io.Reader, output io.Writer) error {
	return xm.maskInternal(input, output)
}
func (xm *XorMasker) UnMask(input io.Reader, output io.Writer) error {
	return xm.maskInternal(input, output)
}
