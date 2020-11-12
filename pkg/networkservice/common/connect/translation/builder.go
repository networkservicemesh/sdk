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

package translation

import "github.com/networkservicemesh/api/pkg/api/networkservice"

// Builder is a builder for translation client
type Builder struct {
	requestOpts []RequestOption
	connOpts    []ConnectionOption
}

// WithRequestOptions appends request options to the builder
func (b *Builder) WithRequestOptions(requestOpts ...RequestOption) *Builder {
	if b == nil {
		b = new(Builder)
	}

	b.requestOpts = append(b.requestOpts, requestOpts...)

	return b
}

// WithConnectionOptions appends connection options to the builder
func (b *Builder) WithConnectionOptions(connOpts ...ConnectionOption) *Builder {
	if b == nil {
		b = new(Builder)
	}

	b.connOpts = append(b.connOpts, connOpts...)

	return b
}

// Build returns a new translation client with set request, connection options
func (b *Builder) Build() networkservice.NetworkServiceClient {
	return &translationClient{
		requestOpts: b.requestOpts,
		connOpts:    b.connOpts,
	}
}

// RequestOption is a request translation option
type RequestOption func(request *networkservice.NetworkServiceRequest, clientConn *networkservice.Connection)

// ReplaceMechanism replaces request.Connection.Mechanism with clientConn.Mechanism
func ReplaceMechanism() RequestOption {
	return func(request *networkservice.NetworkServiceRequest, clientConn *networkservice.Connection) {
		request.GetConnection().Mechanism = clientConn.GetMechanism()
	}
}

// ReplacePath replaces request.Connection.Path with clientConn.Path
func ReplacePath() RequestOption {
	return func(request *networkservice.NetworkServiceRequest, clientConn *networkservice.Connection) {
		if path := clientConn.GetPath(); path != nil {
			request.GetConnection().Path = clientConn.GetPath()
		}
	}
}

// ReplaceMechanismPreferences clears request.MechanismPreferences
func ReplaceMechanismPreferences() RequestOption {
	return func(request *networkservice.NetworkServiceRequest, _ *networkservice.Connection) {
		request.MechanismPreferences = nil
	}
}

// ConnectionOption is a connection translation option
type ConnectionOption func(conn *networkservice.Connection, clientConn *networkservice.Connection)

// WithMechanism copies clientConn.Mechanism to conn
func WithMechanism() ConnectionOption {
	return func(conn *networkservice.Connection, clientConn *networkservice.Connection) {
		conn.Mechanism = clientConn.GetMechanism()
	}
}

// WithContext copies clientConn.Context to conn
func WithContext() ConnectionOption {
	return func(conn *networkservice.Connection, clientConn *networkservice.Connection) {
		conn.Context = clientConn.GetContext()
	}
}

// WithPath copies clientConn.Path to conn
func WithPath() ConnectionOption {
	return func(conn *networkservice.Connection, clientConn *networkservice.Connection) {
		conn.Path = clientConn.GetPath()
	}
}
