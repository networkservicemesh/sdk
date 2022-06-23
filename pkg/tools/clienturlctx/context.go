// Copyright (c) 2020-2022 Cisco Systems, Inc.
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

// Package clienturlctx allows the setting of a client url in the context of the request
package clienturlctx

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	clientURLKey     contextKeyType = "ClientURL"
	dnsServerURLSKey contextKeyType = "dnsServerURLs"
	// UnixURLScheme - scheme for unix urls
	UnixURLScheme = "unix"
)

type contextKeyType string

// WithClientURL -
//    Wraps 'parent' in a new Context that has the ClientURL
func WithClientURL(parent context.Context, clientURL *url.URL) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	log.FromContext(parent).Debugf("passed clientURL: %v", clientURL)
	return context.WithValue(parent, clientURLKey, clientURL)
}

// ClientURL -
//   Returns the ClientURL
func ClientURL(ctx context.Context) *url.URL {
	if rv, ok := ctx.Value(clientURLKey).(*url.URL); ok {
		return rv
	}
	return nil
}

// WithDNSServerURLs -
//    Wraps 'parent' in a new Context that has the DNS server URLs
func WithDNSServerURLs(parent context.Context, urls []url.URL) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	log.FromContext(parent).Debugf("passed DNS server urls: %v", urls)
	return context.WithValue(parent, dnsServerURLSKey, urls)
}

// DNSServerURLs -
//    Returns  the DNS server URLs
func DNSServerURLs(ctx context.Context) []url.URL {
	if rv, ok := ctx.Value(dnsServerURLSKey).([]url.URL); ok {
		return rv
	}
	return nil
}
