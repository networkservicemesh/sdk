package monitor_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEventConsumer struct{}

func (t *testEventConsumer) Send(event *networkservice.ConnectionEvent) (err error) {
	return nil
}

var _ monitor.EventConsumer = &testEventConsumer{}

func TestWithEventConsumer(t *testing.T) {
	ctx := context.Background()
	eventConsumers := monitor.FromContext(ctx)
	assert.Nil(t, eventConsumers)
	ec1 := &testEventConsumer{}
	ec2 := &testEventConsumer{}
	ctx1 := monitor.WithEventConsumer(ctx, ec1)
	eventConsumers = monitor.FromContext(ctx1)
	ctx2 := monitor.WithEventConsumer(ctx1, ec2)
	assert.Equal(t, ctx1, ctx2)
	eventConsumers = monitor.FromContext(ctx2)
	require.Len(t, eventConsumers, 2)
	assert.Equal(t, eventConsumers[0], ec1)
	assert.Equal(t, eventConsumers[1], ec2)
}
