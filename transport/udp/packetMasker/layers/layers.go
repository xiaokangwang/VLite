package layers

import (
	"bytes"
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"io/ioutil"
)

type SyntheticLayerMasker struct {
	layers []interfaces.Masker
}

func (s SyntheticLayerMasker) Mask(input io.Reader, output io.Writer) error {
	Layerslen := len(s.layers)
	Content, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	for i := 0; i < Layerslen; i++ {
		reader := bytes.NewReader(Content)
		writer := bytes.NewBuffer(nil)
		err = s.layers[i].Mask(reader, writer)
		if err != nil {
			return err
		}
		Content = writer.Bytes()
	}

	_, err = io.Copy(output, bytes.NewReader(Content))

	if err != nil {
		return err
	}

	return nil
}

func (s SyntheticLayerMasker) UnMask(input io.Reader, output io.Writer) error {
	Layerslen := len(s.layers)
	Content, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	for io := 0; io < Layerslen; io++ {
		i := Layerslen - io - 1
		reader := bytes.NewReader(Content)
		writer := bytes.NewBuffer(nil)
		err = s.layers[i].UnMask(reader, writer)
		if err != nil {
			return err
		}
		Content = writer.Bytes()
	}
	_, err = io.Copy(output, bytes.NewReader(Content))
	if err != nil {
		return err
	}

	return nil
}

func NewSyntheticLayerMasker(layers []interfaces.Masker) *SyntheticLayerMasker {
	slm := &SyntheticLayerMasker{layers: layers}
	return slm
}
