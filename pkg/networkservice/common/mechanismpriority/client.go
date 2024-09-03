// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package mechanismpriority prioritize mechanisms according to the list
package mechanismpriority

import (
	"context"
	"sort"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type mechanismPriorityClient struct {
	priorities map[string]int
}

// NewClient - returns a new client chain element that prioritize mechanisms according to the list.
func NewClient(priorities ...string) networkservice.NetworkServiceClient {
	c := &mechanismPriorityClient{
		priorities: map[string]int{},
	}
	for i, p := range priorities {
		c.priorities[strings.ToUpper(p)] = i
	}

	return c
}

func (m *mechanismPriorityClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.MechanismPreferences = prioritizeMechanismsByType(request.GetMechanismPreferences(), m.priorities)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (m *mechanismPriorityClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// At the top of the list are the mechanisms to be sorted by priority. The tail contains mechanisms without priorities.
func prioritizeMechanismsByType(mechanisms []*networkservice.Mechanism, priorities map[string]int) []*networkservice.Mechanism {
	tailIndex := 0
	passedElements := 0
	for ; passedElements < len(mechanisms); passedElements++ {
		if _, ok := priorities[mechanisms[tailIndex].GetType()]; ok {
			tailIndex++
		} else {
			value := mechanisms[tailIndex]
			mechanisms = append(append(mechanisms[:tailIndex], mechanisms[tailIndex+1:]...), value)
		}
	}

	// Sort by priority
	sort.SliceStable(mechanisms[:tailIndex], func(i, j int) bool {
		return priorities[mechanisms[i].GetType()] < priorities[mechanisms[j].GetType()]
	})

	return mechanisms
}
