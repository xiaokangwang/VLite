package polyxor

import (
	"bytes"
	"golang.org/x/crypto/sha3"
	"hash/crc64"
	"io"
	"math/rand"
)

type XorMasker struct {
	MaskingPattern []byte
	MaskingSeed    string

	MaskingPatternPoly uint64
}

func NewPolyXorMasker(masker string) *XorMasker {
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
	io.Copy(&maskerpattern, io.LimitReader(Seeder, 16))

	xm.MaskingPattern = maskerpattern.Bytes()

	xm.MaskingPatternPoly = crc64.Checksum(xm.MaskingPattern, crctable)
}

var crctable = crc64.MakeTable(crc64.ECMA)

func (xm *XorMasker) maskInternal(input io.Reader, output io.Writer) error {
	var counter = 0
	var buf [1600]byte
	var polymask [1600]byte
	for {
		n, err := input.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		maskingseed := buf[n-16 : n]

		randsource := rand.New(rand.NewSource(int64(xm.MaskingPatternPoly ^ crc64.Checksum(maskingseed, crctable))))

		randsource.Read(polymask[:n])

		for i := 0; i < n-16; i++ {

			buf[counter] ^= polymask[counter]

			counter++
		}
		_, err = output.Write(buf[:n])

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
