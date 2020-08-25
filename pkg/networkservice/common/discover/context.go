// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package discover provides a NetworkServiceServer chain element that discovers possible NSEs that can provide
// the requested network service and add them to the context.Context where they can be retrieved by
// Candidates(ctx)
package discover

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

const (
	candidatesKey contextKeyType = "Candidates"
)

type contextKeyType string

// NetworkServiceCandidates contains candidates for network service
type NetworkServiceCandidates struct {
	NetworkService *registry.NetworkService
	Endpoints      []*registry.NetworkServiceEndpoint
}

// WithCandidates -
//    Wraps 'parent' in a new Context that has the Candidates
func WithCandidates(parent context.Context, candidates []*registry.NetworkServiceEndpoint, service *registry.NetworkService) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, candidatesKey, &NetworkServiceCandidates{
		NetworkService: service,
		Endpoints:      candidates,
	})
}

// Candidates -
//   Returns the Candidates
func Candidates(ctx context.Context) *NetworkServiceCandidates {
	if rv, ok := ctx.Value(candidatesKey).(*NetworkServiceCandidates); ok {
		return rv
	}
	return nil
}
