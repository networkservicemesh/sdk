// Copyright (c) 2023-2024 Doc.ai and/or its affiliates.
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

// Package testutil has few util functions for testing
package testutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// NewConnection - create connection for testing
func NewConnection() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpRequired: true,
				},
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: "KERNEL",
				Cls:  cls.LOCAL,
			},
			{
				Type: "KERNEL",
				Cls:  cls.LOCAL,
				Parameters: map[string]string{
					"label": "v2",
				},
			},
		},
	}
}

// LabelChangerFirstServer - common server for testing
type LabelChangerFirstServer struct{}

// Request - test request
func (c *LabelChangerFirstServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{
		"Label": "A",
	}
	rv, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	rv.Labels = map[string]string{
		"Label": "D",
	}
	return rv, err
}

// Close - test request
func (c *LabelChangerFirstServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	connection.Labels = map[string]string{
		"Label": "W",
	}
	r, err := next.Server(ctx).Close(ctx, connection)
	connection.Labels = map[string]string{
		"Label": "Z",
	}
	return r, err
}

// LabelChangerSecondServer - common server for testing
type LabelChangerSecondServer struct{}

// Request - test request
func (c *LabelChangerSecondServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{
		"Label": "B",
	}
	rv, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	rv.Labels = map[string]string{
		"Label": "C",
	}
	return rv, err
}

// Close - test request
func (c *LabelChangerSecondServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	connection.Labels = map[string]string{
		"Label": "X",
	}
	r, err := next.Server(ctx).Close(ctx, connection)
	connection.Labels = map[string]string{
		"Label": "Y",
	}
	return r, err
}

// CustomError - custom error for testing
type CustomError struct{}

// Error - error message example
func (*CustomError) Error() string {
	return `Error returned from api/pkg/api/networkservice/networkServiceClient.Close
github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close
	/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85
github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close
	/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65
github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close
	/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65
github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close
	/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106
github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close
	/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65`
}

// StackTrace - testing errors utility
func (*CustomError) StackTrace() errors.StackTrace {
	return []errors.Frame{}
}

// ErrorServer - error server for testing
type ErrorServer struct{}

// Request - test request
func (c *ErrorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{
		"Label": "B",
	}

	_, _ = next.Server(ctx).Request(ctx, request)

	return nil, &CustomError{}
}

// Close - test request
func (c *ErrorServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

// Normalize normalizes test input of logs to make it usable for equals operations
func Normalize(in string) string {
	return strings.ReplaceAll(in, " ", "")
}

// TrimLogTime - to format logs
func TrimLogTime(buff fmt.Stringer) string {
	// Logger created by the trace chain element uses custom formatter, which prints date and time info in each line
	// To check if output matches our expectations, we need to somehow get rid of this info.
	// We have the following options:
	// 1. Configure formatter options on logger creation in trace element
	// 2. Use some global configuration (either set global default formatter
	// 	  instead of creating it in trace element or use global config for our formatter)
	// 3. Remove datetime information from the output
	// Since we are unlikely to need to remove date in any case except these tests,
	// it seems like the third option would be the most convenient.
	result := ""
	datetimeLength := 19
	for _, line := range strings.Split(buff.String(), "\n") {
		if len(line) > datetimeLength {
			result += line[datetimeLength:] + "\n"
		} else {
			result += line
		}
	}

	return result
}
