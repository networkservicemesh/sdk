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

// Package externaldnscontext gets dnscontext from the remote side.
package externaldnscontext

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type externaldnscontextServer struct {
	serverLabels map[string]string
	clientQueue  chan *dns.DNSRequest
	configs      atomic.Value
}

// NewServer  gets dnscontext from the remote side
func NewServer(ctx context.Context, serverLabels map[string]string, remoteURL *url.URL, opts ...grpc.DialOption) networkservice.NetworkServiceServer {
	var r = &externaldnscontextServer{
		serverLabels: serverLabels,
		clientQueue:  make(chan *dns.DNSRequest, 100),
	}
	var logger = log.FromContext(ctx).WithField("externaldnscontextServer", "managePrefixes go-routine")
	r.configs.Store((*dns.Configs)(nil))
	go func() {
		defer close(r.clientQueue)
		<-ctx.Done()
	}()
	go func() {
		for ; ctx.Err() == nil; time.Sleep(time.Millisecond * 100) {
			cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(remoteURL), opts...)
			if err != nil {
				logger.Errorf("cant dial: %v", err.Error())
				continue
			}
			defer func() { _ = cc.Close() }()

			c := dns.NewDNSClient(cc)
			resp, err := c.FetchConfigs(ctx, new(empty.Empty))

			if err != nil {
				logger.Errorf("cant fetch configs: %v")
			}

			r.configs.Store(resp)

			stream, err := c.ManageNames(ctx)
			if err != nil {
				logger.Errorf("cant open stream: %v", err.Error())
				continue
			}

			for req := range r.clientQueue {
				err = stream.Send(req)
				if err != nil {
					logger.Errorf("cant send msg: %v", err.Error())
					break
				}
				_, err = stream.Recv()
				if err != nil {
					logger.Errorf("cant recv msg: %v", err.Error())
					break
				}
			}
			r.configs.Store((*dns.Configs)(nil))
			_ = cc.Close()
		}
	}()
	return r
}

func (e *externaldnscontextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	var configs []*networkservice.DNSConfig

	if v, ok := e.configs.Load().(*dns.Configs); ok && v != nil {
		configs = v.Configs
	} else {
		return nil, errors.New("dns service is not ready yet")
	}

	if request.GetConnection() == nil {
		request.Connection = new(networkservice.Connection)
	}
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = new(networkservice.ConnectionContext)
	}
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.GetConnection().GetContext().DnsContext = new(networkservice.DNSContext)
	}

	e.enqueueDNSRequest(dns.Type_ASSIGN, e.serverLabels, request.GetConnection().GetContext().GetIpContext().GetDstIPNets())
	e.enqueueDNSRequest(dns.Type_ASSIGN, request.GetConnection().GetLabels(), request.GetConnection().GetContext().GetIpContext().GetSrcIPNets())

	request.GetConnection().GetContext().GetDnsContext().Configs = append(request.GetConnection().GetContext().GetDnsContext().Configs, configs...)

	resp, err := next.Server(ctx).Request(ctx, request)

	if err != nil {
		return nil, err
	}

	return resp, err
}

func (e *externaldnscontextServer) enqueueDNSRequest(t dns.Type, labels map[string]string, ipNets []*net.IPNet) {
	var ips []string
	for _, ipNet := range ipNets {
		ips = append(ips, ipNet.IP.String())
	}

	select {
	case e.clientQueue <- &dns.DNSRequest{Type: t, Ips: ips, Labels: labels}:
	default:
	}
}

func (e *externaldnscontextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	e.enqueueDNSRequest(dns.Type_UNASSIGN, e.serverLabels, conn.GetContext().GetIpContext().GetDstIPNets())
	e.enqueueDNSRequest(dns.Type_UNASSIGN, conn.GetLabels(), conn.GetContext().GetIpContext().GetSrcIPNets())

	return next.Server(ctx).Close(ctx, conn)
}
