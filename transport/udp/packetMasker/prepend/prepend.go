package prepend

import (
	"bytes"
	"fmt"
	"io"
)

type PrependingMasker struct {
	PrependingPattern []byte
}

func (p PrependingMasker) Mask(input io.Reader, output io.Writer) error {
	output.Write(p.PrependingPattern)
	io.Copy(output, input)
	return nil
}

func (p PrependingMasker) UnMask(input io.Reader, output io.Writer) error {
	reader := io.LimitReader(input, int64(len(p.PrependingPattern)))
	var buf bytes.Buffer
	io.Copy(&buf, reader)
	if !bytes.Equal(buf.Bytes(), p.PrependingPattern) {
		fmt.Println("PrependingPattern not match")
	}
	io.Copy(output, input)
	return nil
}

func NewPrependingMasker(pattern []byte) *PrependingMasker {
	pm := &PrependingMasker{PrependingPattern: pattern}
	return pm
}
