// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package initjaeger provides chain element, that initializes jaeger tracer.
package initjaeger

import (
	"context"
	"io"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
)

type jaegerInitServer struct {
	once         sync.Once
	tracerCloser io.Closer
	serviceName  string
}

// NewServer creates new jaegerInitServer chain element
func NewServer(serviceName string) networkservice.NetworkServiceServer {
	return &jaegerInitServer{
		serviceName: serviceName,
	}
}

func (jis *jaegerInitServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	jis.once.Do(func() {
		if !opentracing.IsGlobalTracerRegistered() {
			jis.tracerCloser = jaeger.InitJaeger(jis.serviceName)
			go func() {
				<-ctx.Done()
				if err := jis.tracerCloser.Close(); err != nil {
					logrus.Error(err)
				}
			}()
			return
		}
		logrus.Warn("Global opentracer is already initialized")
	})
	return next.Server(ctx).Request(ctx, request)
}

func (jis *jaegerInitServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	e, err := next.Server(ctx).Close(ctx, connection)
	if jis.tracerCloser != nil {
		if tracerErr := jis.tracerCloser.Close(); tracerErr != nil {
			return e, errors.Wrapf(err, "error closing tracer: %v", tracerErr)
		}
	}

	return e, err
}
