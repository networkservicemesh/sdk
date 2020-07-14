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

package interdomain_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
)

var errNotFound = errors.New("not found")

type resource interface {
	Close() error
}

func newInterdomainTestingTool(t *testing.T) *interdomainTestingTool {
	return &interdomainTestingTool{
		ports: map[string]int{},
		T:     t,
	}
}

type interdomainTestingTool struct {
	ports     map[string]int
	resources []resource
	*testing.T
}

func (t *interdomainTestingTool) LookupSRV(_ context.Context, service, proto, domain string) (string, []*net.SRV, error) {
	if v, ok := t.ports[domain]; ok {
		return fmt.Sprintf("_%v._%v.%v", service, proto, domain), []*net.SRV{{
			Port:   uint16(v),
			Target: domain,
		}}, nil
	}
	return "", nil, errNotFound
}

func (t *interdomainTestingTool) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	if _, ok := t.ports[host]; ok {
		return []net.IPAddr{{
			IP: net.ParseIP("127.0.0.1"),
		}}, nil
	}
	return nil, errNotFound
}

func (t *interdomainTestingTool) dialDomain(domain string) *grpc.ClientConn {
	_, srvs, err := t.LookupSRV(context.Background(), "", "tcp", domain)
	require.Nil(t, err)
	require.NotEmpty(t, srvs)
	ips, err := t.LookupIPAddr(context.Background(), srvs[0].Target)
	require.Nil(t, err)
	require.NotEmpty(t, ips)
	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", ips[0].IP.String(), srvs[0].Port), grpc.WithInsecure())
	require.Nil(t, err)
	t.resources = append(t.resources, conn)
	return conn
}

func (t *interdomainTestingTool) startNetworkServiceRegistryServerAsync(domain string, server registry.NetworkServiceRegistryServer) *url.URL {
	return t.startServerAsync(domain, func(s *grpc.Server) {
		registry.RegisterNetworkServiceRegistryServer(s, server)
	})
}

func (t *interdomainTestingTool) startNetworkServiceEndpointRegistryServerAsync(domain string, server registry.NetworkServiceEndpointRegistryServer) *url.URL {
	return t.startServerAsync(domain, func(s *grpc.Server) {
		registry.RegisterNetworkServiceEndpointRegistryServer(s, server)
	})
}

func (t *interdomainTestingTool) startServerAsync(domain string, registerFunc func(server *grpc.Server)) *url.URL {
	s := grpc.NewServer()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	t.ports[domain] = l.Addr().(*net.TCPAddr).Port
	t.resources = append(t.resources, l)
	logrus.Infof("listening domain %v by %v", domain, l.Addr().String())
	registerFunc(s)
	go func() {
		_ = s.Serve(l)
	}()
	u, err := url.Parse("tcp://127.0.0.1:" + fmt.Sprint(t.ports[domain]))
	require.Nil(t, err)
	return u
}

func (t *interdomainTestingTool) cleanup() {
	for _, r := range t.resources {
		_ = r.Close()
	}
	t.resources = nil
}

// At this moment clienurl is using finalization and we should make sure that GC was called before checking leaks
func (t *interdomainTestingTool) verifyNoneLeaks() {
	require.Eventually(t, func() bool {
		runtime.GC()
		return goleak.Find() != nil
	}, time.Second, time.Microsecond*250)
}

var _ dnsresolve.Resolver = (*interdomainTestingTool)(nil)
