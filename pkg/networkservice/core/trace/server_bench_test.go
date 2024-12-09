// Copyright (c) 2024 Cisco Systems, Inc.
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

package trace_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/testutil"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"

	"github.com/sirupsen/logrus"
)

func newTestRequest(size int) *networkservice.NetworkServiceRequest {
	res := testutil.NewConnection()

	res.GetConnection().Path = &networkservice.Path{}
	res.GetConnection().Labels = make(map[string]string)

	for i := 0; i < size; i++ {
		res.GetConnection().GetPath().PathSegments = append(res.GetConnection().GetPath().PathSegments,
			&networkservice.PathSegment{
				Name:  nanoid.MustGenerateString(15),
				Token: nanoid.MustGenerateString(25),
				Id:    nanoid.MustGenerateString(20),
			})
	}

	res.GetConnection().Id = res.GetConnection().GetPath().GetPathSegments()[0].Id

	return res
}

func newTestServerChain(size int) networkservice.NetworkServiceServer {
	var servers []networkservice.NetworkServiceServer

	for i := 0; i < size; i++ {
		id := i
		servers = append(servers, checkrequest.NewServer(nil, func(_ *testing.T, nsr *networkservice.NetworkServiceRequest) {
			nsr.GetConnection().GetLabels()["common-label"] = fmt.Sprint(id)
		}))
	}

	return chain.NewNetworkServiceServer(servers...)
}

func newTestStaticServerChain(size int) networkservice.NetworkServiceServer {
	var servers []networkservice.NetworkServiceServer

	for i := 0; i < size; i++ {
		servers = append(servers, null.NewServer())
	}

	return chain.NewNetworkServiceServer(servers...)
}

func Benchmark_ShortRequest_Info(b *testing.B) {
	var s = newTestServerChain(10)
	var request = newTestRequest(5)
	var ctx = context.Background()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}

func Benchmark_LongRequest_Info(b *testing.B) {
	var s = newTestServerChain(100)
	var request = newTestRequest(100)
	var ctx = context.Background()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}

func Benchmark_ShortRequest_Trace(b *testing.B) {
	var s = newTestServerChain(10)
	var request = newTestRequest(5)
	var ctx = context.Background()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}
func Benchmark_LongRequest_Trace(b *testing.B) {
	var s = newTestServerChain(100)
	var request = newTestRequest(100)
	var ctx = context.Background()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}

func Benchmark_LongRequest_Trace_NoDiff(b *testing.B) {
	var s = newTestStaticServerChain(100)
	var request = newTestRequest(100)
	var ctx = context.Background()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}

func Benchmark_LongRequest_Diff_Warn(b *testing.B) {
	var s = newTestStaticServerChain(100)
	var request = newTestRequest(100)
	var ctx = context.Background()

	logrus.SetLevel(logrus.WarnLevel)
	logrus.SetOutput(io.Discard)
	log.EnableTracing(true)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Request(ctx, request)
	}
}
