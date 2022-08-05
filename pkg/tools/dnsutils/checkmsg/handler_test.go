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

package checkmsg_test

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/cache"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/checkmsg"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type responseWriter struct {
	dns.ResponseWriter
	Response *dns.Msg
}

func (r *responseWriter) WriteMsg(m *dns.Msg) error {
	r.Response = m
	return nil
}

func TestCheckMsgHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100000*time.Second)
	defer cancel()

	logrus.SetLevel(logrus.TraceLevel)
	log.EnableTracing(true)

	dnsConfigsMap := new(dnsconfig.Map)
	dnsConfigs := []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{"8.8.8.8"},
			SearchDomains: []string{"com"},
		},
	}
	dnsConfigsMap.Store("a", dnsConfigs)

	handler := chain.NewDNSHandler(
		checkmsg.NewDNSHandler(),
		dnsconfigs.NewDNSHandler(dnsConfigsMap),
		searches.NewDNSHandler(),
		noloop.NewDNSHandler(),
		norecursion.NewDNSHandler(),
		cache.NewDNSHandler(),
		fanout.NewDNSHandler(),
	)

	rw := &responseWriter{}
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	handler.ServeDNS(ctx, rw, m)
}
