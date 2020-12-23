package interfaces

import (
	"context"
	"net"
)

type AbstractDialer interface {
	Dial(ctx context.Context, token string) (net.Conn, error)
}

type AbstractListener interface {
	Accept(ctx context.Context) (net.Conn, string, error)
}
