// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
)

// NewFakeResolver returns a fake Resolver that can have records added
// to it using AddSRVEntry.
func NewFakeResolver() dnsresolve.Resolver {
	return &fakeResolver{
		ports:     map[string]string{},
		addresses: map[string]string{},
	}
}

// AddSRVEntry adds a DNS record to r using name and service as the
// key, and the host and port in u as the values. r must be a Resolver
// that was created by NewFakeDNSResolver.
func AddSRVEntry(r dnsresolve.Resolver, name, service string, u *url.URL) (err error) {
	f := r.(*fakeResolver)
	f.Lock()
	defer f.Unlock()

	key := fmt.Sprintf("%v.%v", service, name)
	f.addresses[key], f.ports[key], err = net.SplitHostPort(u.Host)

	return
}

// fakeResolver implements the dnsresolve.Resolver interface and can
// be used for logic DNS testing.
type fakeResolver struct {
	sync.Mutex
	addresses map[string]string
	ports     map[string]string
}

// LookupSRV looks up a DNS SRV record.
func (f *fakeResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	f.Lock()
	defer f.Unlock()

	if v, ok := f.ports[name]; ok {
		port, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("_%v._%v.%v", service, proto, name), []*net.SRV{{
			Port:   uint16(port),
			Target: name,
		}}, nil
	}
	return "", nil, errors.New("not found")
}

// LookupIPAddr looks up an IP address by host.
func (f *fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	f.Lock()
	defer f.Unlock()

	if address, ok := f.addresses[host]; ok {
		return []net.IPAddr{{
			IP: net.ParseIP(address),
		}}, nil
	}
	return nil, errors.New("not found")
}

// Ensure that FakeDNSResolver is a valid Resolver.
var _ dnsresolve.Resolver = (*fakeResolver)(nil)
