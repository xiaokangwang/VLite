package interfaces

import (
	"context"
	"io"
)

type StreamRelayer interface {
	RelayStream(conn io.ReadWriteCloser, ctx context.Context)
}
