package connect

import (
	"context"
	"net"
)

type contextKey string

const ipKey contextKey = "connect.TCPAddr"

func WithAddr(ctx context.Context, addr *net.TCPAddr) context.Context {
	return context.WithValue(ctx, ipKey, addr)
}

func Addr(ctx context.Context) *net.TCPAddr {
	if val := ctx.Value(ipKey); val != nil {
		if addr, ok := val.(*net.TCPAddr); ok {
			return addr
		}
	}
	return nil
}
