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

// Package prioritymechanisms prioritize mechanisms according to the list
package prioritymechanisms

import (
	"context"
	"sort"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type priorityMechanismsClient struct {
	priorities map[string]int
}

// NewClient - returns a new client chain element that prioritize mechanisms according to the list
func NewClient(priorities []string) networkservice.NetworkServiceClient {
	c := &priorityMechanismsClient{
		priorities: map[string]int{},
	}
	for i, p := range priorities {
		c.priorities[strings.ToUpper(p)] = i
	}

	return c
}

func (p *priorityMechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.MechanismPreferences = prioritizeMechanismsByType(request.GetMechanismPreferences(), p.priorities)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (p *priorityMechanismsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func prioritizeMechanismsByType(mechanisms []*networkservice.Mechanism, priorities map[string]int) []*networkservice.Mechanism {
	var head, tail []*networkservice.Mechanism
	for _, mechanism := range mechanisms {
		if _, ok := priorities[mechanism.GetType()]; ok {
			head = append(head, mechanism)
		} else {
			tail = append(tail, mechanism)
		}
	}
	sort.Slice(head, func(i, j int) bool {
		return priorities[head[i].GetType()] < priorities[head[j].GetType()]
	})

	return append(head, tail...)
}
