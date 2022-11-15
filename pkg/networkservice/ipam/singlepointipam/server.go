// Copyright (c) 2020-2024 Nordix and its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package singlepointipam defines a chain element that implements IPAM service
package singlepointipam

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type singlePIpam struct {
	genericsync.Map[string, *connectionInfo]
	ipPools      []*ippool.IPPool
	ipamNetworks []*IpamNet
	myIPs        []string
	masks        []string
	once         sync.Once
	initErr      error
}

type connectionInfo struct {
	ipPool  *ippool.IPPool
	srcAddr string
	dstAddr string
}

func (i *connectionInfo) shouldUpdate(exclude *ippool.IPPool) bool {
	srcIP, _, srcErr := net.ParseCIDR(i.srcAddr)
	dstIP, _, dstErr := net.ParseCIDR(i.dstAddr)

	return srcErr == nil && exclude.ContainsString(srcIP.String()) || dstErr == nil && exclude.ContainsString(dstIP.String())
}

// NewServer - creates a new NetworkServiceServer chain element that implements IPAM service.
func NewServer(networks ...*IpamNet) networkservice.NetworkServiceServer {
	return &singlePIpam{
		ipamNetworks: networks,
	}
}
func (sipam *singlePIpam) init() {
	if len(sipam.ipamNetworks) == 0 {
		sipam.initErr = errors.New("required one or more prefixes/ranges")
		return
	}
	for _, ipamNetwork := range sipam.ipamNetworks {
		if ipamNetwork.Network == nil {
			sipam.initErr = errors.Errorf("prefix must not be nil: %+v", sipam.ipamNetworks)
			return
		}
		prefix := ipamNetwork.Network
		ones, _ := prefix.Mask.Size()
		mask := fmt.Sprintf("/%d", ones)
		sipam.masks = append(sipam.masks, mask)
		ipPool := ipamNetwork.Pool

		if ipPool == nil { // simple cidr prefix
			ipPool = ippool.NewWithNet(prefix)
		}
		if prefix.IP.To4() != nil {
			// Remove the broadcast address from the pool
			_, _ = ipPool.PullIP(cidr.BroadcastAddress(prefix))
		}
		sipam.ipPools = append(sipam.ipPools, ipPool)
	}
}

func (sipam *singlePIpam) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	sipam.once.Do(sipam.init)
	if sipam.initErr != nil {
		return nil, errors.Wrap(sipam.initErr, "failed to init IPAM server during request")
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

	excludeIP4, excludeIP6 := exclude(ipContext.GetExcludedPrefixes()...)

	connInfo, loaded := sipam.Load(conn.GetId())
	if loaded && (connInfo.shouldUpdate(excludeIP4) || connInfo.shouldUpdate(excludeIP6)) {
		// some of the existing addresses are excluded
		deleteAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
		deleteAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
		sipam.free(connInfo)
		loaded = false
	}
	var err error
	if !loaded {
		if connInfo, err = sipam.getAddrs(excludeIP4, excludeIP6); err != nil {
			return nil, err
		}
		sipam.Store(conn.GetId(), connInfo)
	}

	addAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		if !loaded {
			sipam.free(connInfo)
		}
		return nil, err
	}

	return conn, nil
}

func (sipam *singlePIpam) Close(
	ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	sipam.once.Do(sipam.init)
	if sipam.initErr != nil {
		return nil, errors.Wrap(sipam.initErr, "failed to init IPAM server during close")
	}

	if connInfo, ok := sipam.Load(conn.GetId()); ok {
		sipam.free(connInfo)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func addMaskIP(ip net.IP) string {
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}

func (sipam *singlePIpam) free(connInfo *connectionInfo) {
	ipAddr, _, err := net.ParseCIDR(connInfo.srcAddr)
	if err == nil {
		connInfo.ipPool.AddNetString(addMaskIP(ipAddr))
	}
}

func (sipam *singlePIpam) setMyIP(i int) error {
	myIP, err := sipam.ipPools[i].Pull()
	if err != nil {
		return err
	}
	if i >= len(sipam.myIPs) {
		sipam.myIPs = append(sipam.myIPs, myIP.String()+sipam.masks[i])
	} else {
		sipam.myIPs[i] = myIP.String() + sipam.masks[i]
	}
	return nil
}

func (sipam *singlePIpam) getAddrs(excludeIP4, excludeIP6 *ippool.IPPool) (connInfo *connectionInfo, err error) {
	var dstAddr, srcAddr net.IP

	for i := 0; i < len(sipam.ipamNetworks); i++ {
		// The NSE needs only one src address
		dstSet := false
		if i >= len(sipam.myIPs) {
			err = sipam.setMyIP(i)
			if err != nil {
				continue
			}
		}

		for {
			myIP, _, _ := net.ParseCIDR(sipam.myIPs[i])
			if !excludeIP4.ContainsString(myIP.String()) && !excludeIP6.ContainsString(myIP.String()) {
				dstAddr = myIP
				dstSet = true
				break
			}
			err = sipam.setMyIP(i)
			if err != nil {
				break
			}
		}
		for {
			if srcAddr, err = sipam.ipPools[i].Pull(); err != nil {
				break
			}
			if !excludeIP4.ContainsString(srcAddr.String()) && !excludeIP6.ContainsString(srcAddr.String()) {
				if dstSet {
					return &connectionInfo{
						ipPool:  sipam.ipPools[i],
						srcAddr: srcAddr.String() + sipam.masks[i],
						dstAddr: dstAddr.String() + sipam.masks[i],
					}, nil
				}
			}
		}
	}
	return nil, err
}

//
// common with point2point
// ------------------------

func exclude(prefixes ...string) (ipv4exclude, ipv6exclude *ippool.IPPool) {
	ipv4exclude = ippool.New(net.IPv4len)
	ipv6exclude = ippool.New(net.IPv6len)
	for _, prefix := range prefixes {
		ipv4exclude.AddNetString(prefix)
		ipv6exclude.AddNetString(prefix)
	}
	return
}
func deleteAddr(addrs *[]string, addr string) {
	for i, a := range *addrs {
		if a == addr {
			*addrs = append((*addrs)[:i], (*addrs)[i+1:]...)
			return
		}
	}
}

func addAddr(addrs *[]string, addr string) {
	for _, a := range *addrs {
		if a == addr {
			return
		}
	}
	*addrs = append(*addrs, addr)
}
