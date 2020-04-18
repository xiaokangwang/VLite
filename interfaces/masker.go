package interfaces

import "io"

type Masker interface {
	Mask(input io.Reader, output io.Writer) error
	UnMask(input io.Reader, output io.Writer) error
}
