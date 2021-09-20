package grpcutils

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc/test/bufconn"
)

func BufConContextDialer(l *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		default:
		}
		return l.Dial()
	}
}
