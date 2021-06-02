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

package begin

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type entryContextKey struct{}

func withEntry(parent context.Context, t *EventOriginator) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, entryContextKey{}, t)
}

// FromContext - returns a begin.EventOriginator from the ctx if one is present, otherwise nil
func FromContext(ctx context.Context) *EventOriginator {
	if rv, ok := ctx.Value(entryContextKey{}).(*EventOriginator); ok {
		return rv
	}
	return nil
}

type requestMutatorContextKey struct{}

// RequestMutatorFunc - function to mutate a networkservice.NetworkServiceRequest
type RequestMutatorFunc func(request *networkservice.NetworkServiceRequest)

// WithRequestMutator - add a RequestMutatorFunc to the list of RequestMutatorFunc's for the context.Context
func WithRequestMutator(parent context.Context, mutators ...RequestMutatorFunc) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestMutatorContextKey{}, append(requestMutators(parent), mutators...))
}

func requestMutators(ctx context.Context) []RequestMutatorFunc {
	if rv, ok := ctx.Value(requestMutatorContextKey{}).([]RequestMutatorFunc); ok {
		return rv
	}
	return nil
}
