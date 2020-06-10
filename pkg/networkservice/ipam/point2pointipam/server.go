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

// Package point2pointipam provides a simple ipam appropriate for point2pointipam.
// point2pointipam assigns two ip addresses out of a pool of prefixes. The IP
// addresses assigned are not reassigned until they are released. All
// IP addresses assigned are assigned a CIDR mask of /32
package point2pointipam

import (
	"context"
	"encoding/binary"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/RoaringBitmap/roaring"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

type pointToPointServer struct {
	mutex    *sync.Mutex
	prefixes []*net.IPNet
	freeIPs  *roaring.Bitmap
	once     sync.Once
	initErr  error
}

func (srv *pointToPointServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	srv.once.Do(srv.init)
	if srv.initErr != nil {
		return nil, srv.initErr
	}
	srv.mutex.Lock()
	defer srv.mutex.Unlock()

	if srv.freeIPs.IsEmpty() {
		return nil, errors.New("ipam allocation pool depleted")
	}

	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{}
	}
	conn := request.GetConnection()

	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	connContext := conn.GetContext()

	if connContext.GetIpContext() == nil {
		connContext.IpContext = &networkservice.IPContext{}
	}
	ipContext := connContext.GetIpContext()

	exclude := roaring.New()
	excludePrefixes := ipContext.GetExcludedPrefixes()
	for _, prefix := range excludePrefixes {
		_, ipnet, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}
		low := binary.BigEndian.Uint32(cidr.NetworkAddress(ipnet).To4())
		high := binary.BigEndian.Uint32(cidr.BroadcastAddress(ipnet).To4()) + 1

		exclude.AddRange(uint64(low), uint64(high))
	}

	available := roaring.And(roaring.Xor(srv.freeIPs, exclude), srv.freeIPs)
	if available.IsEmpty() {
		return nil, errors.New("available IP addresses excluded by request")
	}
	dstInt := available.Minimum()
	available.Remove(dstInt)

	// explicitly check again, panics if empty on minimum and remove calls
	if available.IsEmpty() {
		return nil, errors.New("available IP addresses excluded by request")
	}
	srcInt := available.Minimum()
	available.Remove(srcInt)

	srv.freeIPs.Remove(dstInt)
	srv.freeIPs.Remove(srcInt)

	dstIP := make(net.IP, 4)
	srcIP := make(net.IP, 4)

	binary.BigEndian.PutUint32(dstIP, dstInt)
	binary.BigEndian.PutUint32(srcIP, srcInt)

	conn.Context.IpContext.DstIpAddr = dstIP.String() + "/32"
	conn.Context.IpContext.SrcIpAddr = srcIP.String() + "/32"

	return next.Server(ctx).Request(ctx, request)
}

func (srv *pointToPointServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	srv.once.Do(srv.init)
	if srv.initErr != nil {
		return nil, srv.initErr
	}
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

func (srv *pointToPointServer) init() {
	if len(srv.prefixes) == 0 {
		srv.initErr = errors.New("required one or more prefixes")
		return
	}

	for _, prefix := range srv.prefixes {
		if prefix == nil {
			srv.initErr = errors.Errorf("prefix must not be nil: %+v", srv.prefixes)
			return
		}
	}

	freeIPs := roaring.New()
	for _, ipnet := range srv.prefixes {
		networkAddress := cidr.NetworkAddress(ipnet)
		broadcastAddress := cidr.BroadcastAddress(ipnet)

		low := binary.BigEndian.Uint32(networkAddress)
		high := binary.BigEndian.Uint32(broadcastAddress) + 1
		freeIPs.AddRange(uint64(low), uint64(high))

		// TODO should we remove first and last (network, broadcast) addresses?
		// freeIPs.Remove(low)
		// freeIPs.Remove(high)
	}
	srv.freeIPs = freeIPs
}

// NewServer - creates a NetworkServiceServer that requests a kernel interface and populates the netns inode
func NewServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return &pointToPointServer{
		mutex:    &sync.Mutex{},
		prefixes: prefixes,
	}
}
