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

package dnscontext_test

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

func TestDNSClient_ReceivesUpdateEvent(t *testing.T) {
	defer func() {
		_ = os.Remove("corefile")
	}()
	resolveConfigPath := path.Join(os.TempDir(), "resolv.conf")
	err := ioutil.WriteFile(resolveConfigPath, []byte(`
nameserver 8.8.4.4
search example.com`), os.ModePerm)
	require.Nil(t, err)
	require.Nil(t, err)
	eventCh := make(chan *networkservice.ConnectionEvent, 2)
	client := chain.NewNetworkServiceClient(dnscontext.NewClient(context.Background(), "corefile", resolveConfigPath, eventchannel.NewMonitorConnectionClient(eventCh), func() []grpc.CallOption {
		return []grpc.CallOption{grpc.WaitForReady(true)}
	}))
	_, err = client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.Nil(t, err)
	eventCh <- &networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			"1": {
				Context: &networkservice.ConnectionContext{
					DnsContext: &networkservice.DNSContext{
						Configs: []*networkservice.DNSConfig{
							{
								DnsServerIps: []string{"8.8.8.8"},
							},
						},
					},
				},
			},
		},
	}
	<-time.After(time.Second)
	b, err := ioutil.ReadFile("corefile")
	require.Nil(t, err)
	actual := string(b)
	require.NotContains(t, actual, "forward")
	require.Contains(t, actual, "fanout")
	require.Contains(t, actual, "8.8.8.8")
	require.Contains(t, actual, "8.8.4.4")
	_, err = client.Close(context.Background(), &networkservice.Connection{})
	require.Nil(t, err)
}

func TestDNSClient_ReceivesDeleteEvent(t *testing.T) {
	defer func() {
		_ = os.Remove("corefile")
	}()
	const expected = `. {
	forward . 8.8.4.4
	log
	reload
}`
	resolveConfigPath := path.Join(os.TempDir(), "resolv.conf")
	err := ioutil.WriteFile(resolveConfigPath, []byte(`
nameserver 8.8.4.4
search example.com`), os.ModePerm)
	require.Nil(t, err)
	require.Nil(t, err)
	eventCh := make(chan *networkservice.ConnectionEvent, 2)
	client := chain.NewNetworkServiceClient(dnscontext.NewClient(context.Background(), "corefile", resolveConfigPath, eventchannel.NewMonitorConnectionClient(eventCh), func() []grpc.CallOption {
		return []grpc.CallOption{grpc.WaitForReady(true)}
	}))
	_, err = client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.Nil(t, err)
	eventCh <- &networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			"1": {
				Context: &networkservice.ConnectionContext{
					DnsContext: &networkservice.DNSContext{
						Configs: []*networkservice.DNSConfig{
							{
								SearchDomains: []string{"example.com"},
								DnsServerIps:  []string{"8.8.8.8"},
							},
						},
					},
				},
			},
		},
	}
	eventCh <- &networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{
			"1": {},
		},
	}
	<-time.After(time.Second)
	actual, err := ioutil.ReadFile("corefile")
	require.Nil(t, err)
	require.Equal(t, expected, string(actual))
	_, err = client.Close(context.Background(), &networkservice.Connection{})
	require.Nil(t, err)
}
