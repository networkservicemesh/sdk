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

package monitor

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type eventConsumerKey struct{}

// EventConsumer - entity to which a ConnectionEvent can be sent
type EventConsumer interface {
	Send(event *networkservice.ConnectionEvent) (err error)
}

// WithEventConsumer - Add EventConsumer to context
func WithEventConsumer(parent context.Context, eventConsumer EventConsumer) context.Context {
	valueRaw := parent.Value(eventConsumerKey{})
	if valueRaw == nil {
		return context.WithValue(parent, eventConsumerKey{}, &([]EventConsumer{eventConsumer}))
	}
	valuePtr, _ := valueRaw.(*[]EventConsumer)
	*valuePtr = append(*valuePtr, eventConsumer)
	return parent
}

// FromContext - All Event Consumers in the context
func FromContext(ctx context.Context) []EventConsumer {
	value, ok := ctx.Value(eventConsumerKey{}).(*[]EventConsumer)
	if ok {
		rv := *value
		return rv
	}
	return nil
}
