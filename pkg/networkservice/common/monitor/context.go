package monitor

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type eventConsumerKey struct{}

type EventConsumer interface {
	Send(event *networkservice.ConnectionEvent) (err error)
}

func WithEventConsumer(parent context.Context, eventConsumer EventConsumer) context.Context {
	valueRaw := parent.Value(eventConsumerKey{})
	if valueRaw == nil {
		return context.WithValue(parent, eventConsumerKey{}, &([]EventConsumer{eventConsumer}))
	}
	valuePtr, _ := valueRaw.(*[]EventConsumer)
	*valuePtr = append(*valuePtr, eventConsumer)
	return parent
}

func FromContext(ctx context.Context) []EventConsumer {
	value, ok := ctx.Value(eventConsumerKey{}).(*[]EventConsumer)
	if ok {
		rv := *value
		fmt.Printf("FromContext ptr %p\n", rv)
		return rv
	}
	return nil
}
