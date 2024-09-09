// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package vl3dns

import (
	"context"
	"fmt"
	"text/template"

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

// Option configures vl3DNSServer.
type Option func(*vl3DNSServer)

// WithConfigs sets initial list to fanout queries.
func WithConfigs(m *genericsync.Map[string, []*networkservice.DNSConfig]) Option {
	return func(vd *vl3DNSServer) {
		vd.dnsConfigs = m
	}
}

// WithDNSServerHandler replaces default dns handler to specific one.
func WithDNSServerHandler(handler dnsutils.Handler) Option {
	return func(vd *vl3DNSServer) {
		vd.dnsServer = handler
	}
}

// WithDomainSchemes sets domain schemes for vl3 dns server. Schemes are using to get dns names for clients.
func WithDomainSchemes(domainSchemes ...string) Option {
	return func(vd *vl3DNSServer) {
		vd.domainSchemeTemplates = nil
		for i, domainScheme := range domainSchemes {
			vd.domainSchemeTemplates = append(vd.domainSchemeTemplates,
				template.Must(template.New(fmt.Sprintf("dnsScheme%d", i)).
					Funcs(template.FuncMap{
						"target": interdomain.Target,
						"domain": interdomain.Domain,
					}).
					Parse(domainScheme)))
		}
	}
}

// WithDNSListenAndServeFunc replaces default listen and serve behavior for inner dns server.
func WithDNSListenAndServeFunc(listenAndServeDNS func(ctx context.Context, handler dnsutils.Handler, listenOn string)) Option {
	if listenAndServeDNS == nil {
		panic("listenAndServeDNS cannot be nil")
	}
	return func(vd *vl3DNSServer) {
		vd.listenAndServeDNS = listenAndServeDNS
	}
}

// WithDNSPort replaces default dns port for the inner dns server.
func WithDNSPort(dnsPort int) Option {
	return func(vd *vl3DNSServer) {
		vd.dnsPort = dnsPort
	}
}
