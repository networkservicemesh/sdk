// Copyright (c) 2021 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
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
	require.Len(t, eventConsumers, 1)
	assert.Equal(t, eventConsumers[0], ec1)
	ctx2 := monitor.WithEventConsumer(ctx1, ec2)
	assert.Equal(t, ctx1, ctx2)
	eventConsumers = monitor.FromContext(ctx2)
	require.Len(t, eventConsumers, 2)
	assert.Equal(t, eventConsumers[0], ec1)
	assert.Equal(t, eventConsumers[1], ec2)
}
