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

// Package point2point provides a simple ipam appropriate for point2point.
// point2point assigns two ip addresses out of a pool of prefixes. The IP
// addresses assigned are not reassigned until they are released. All
// IP addresses assigned are assigned a CIDR mask of /32
package point2point

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

type linkLocalServer struct {
	mutex    *sync.Mutex
	prefixes []*net.IPNet
	freeIPs  *roaring.Bitmap
}

func (srv *linkLocalServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()

	dstInt := srv.freeIPs.Minimum()
	srv.freeIPs.Remove(dstInt)

	srcInt := srv.freeIPs.Minimum()
	srv.freeIPs.Remove(srcInt)

	conn := request.Connection

	dstIP := make(net.IP, 4)
	srcIP := make(net.IP, 4)

	binary.BigEndian.PutUint32(dstIP, dstInt)
	binary.BigEndian.PutUint32(srcIP, srcInt)

	conn.Context.IpContext.DstIpAddr = dstIP.String() + "/32"
	conn.Context.IpContext.SrcIpAddr = srcIP.String() + "/32"

	return next.Server(ctx).Request(ctx, request)
}

func (srv *linkLocalServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()

	dstIP, _, err := net.ParseCIDR(conn.Context.IpContext.DstIpAddr)
	if err != nil {
		return nil, err
	}

	srcIP, _, err := net.ParseCIDR(conn.Context.IpContext.SrcIpAddr)
	if err != nil {
		return nil, err
	}

	intip := binary.BigEndian.Uint32(dstIP.To4())
	srv.freeIPs.Add(intip)

	intip = binary.BigEndian.Uint32(srcIP.To4())
	srv.freeIPs.Add(intip)

	return next.Server(ctx).Close(ctx, conn)
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer(prefixes []*net.IPNet) (networkservice.NetworkServiceServer, error) {
	if len(prefixes) == 0 {
		return nil, errors.New("prefixes must not be nil")
	}

	for _, prefix := range prefixes {
		if prefix == nil {
			return nil, errors.New("prefix must not be nil")
		}
	}

	freeIPs := roaring.New()
	for _, ipnet := range prefixes {
		networkAddress := cidr.NetworkAddress(ipnet)
		broadcastAddress := cidr.BroadcastAddress(ipnet)

		low := binary.BigEndian.Uint32(networkAddress)
		high := binary.BigEndian.Uint32(broadcastAddress)
		freeIPs.AddRange(uint64(low), uint64(high))

		// TODO should we remove first and last (network, broadcast) addresses?
		// freeIPs.Remove(low)
		// freeIPs.Remove(high)
	}

	return &linkLocalServer{
		mutex:    &sync.Mutex{},
		prefixes: prefixes,
		freeIPs:  freeIPs,
	}, nil
}
