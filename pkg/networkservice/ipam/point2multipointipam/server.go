// Copyright (c) 2020-2021 Nordix and its affiliates.
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

package point2multipointipam

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type multipIpam struct {
	Map
	ipPools  []*ippool.IPPool
	prefixes []*net.IPNet
	myIPs    []string
	masks    []string
	once     sync.Once
	initErr  error
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
func NewServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return &multipIpam{
		prefixes: prefixes,
	}
}
func (mipam *multipIpam) init() {
	if len(mipam.prefixes) == 0 {
		mipam.initErr = errors.New("required one or more prefixes")
		return
	}
	for _, prefix := range mipam.prefixes {
		if prefix == nil {
			mipam.initErr = errors.Errorf("prefix must not be nil: %+v", mipam.prefixes)
			return
		}
		ones, _ := prefix.Mask.Size()
		mask := fmt.Sprintf("/%d", ones)
		mipam.masks = append(mipam.masks, mask)
		ipPool := ippool.NewWithNet(prefix)
		ipPool.ExcludeString(getFirstIP(prefix))
		ipPool.ExcludeString(getLastIP(prefix))
		mipam.ipPools = append(mipam.ipPools, ipPool)
	}
}

func (mipam *multipIpam) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mipam.once.Do(mipam.init)
	if mipam.initErr != nil {
		return nil, mipam.initErr
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

	connInfo, loaded := mipam.Load(conn.GetId())
	if loaded && (connInfo.shouldUpdate(excludeIP4) || connInfo.shouldUpdate(excludeIP6)) {
		// some of the existing addresses are excluded
		deleteAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)
		deleteAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
		mipam.free(connInfo)
		loaded = false
	}
	var err error
	if !loaded {
		if connInfo, err = mipam.getAddrs(excludeIP4, excludeIP6); err != nil {
			return nil, err
		}
		mipam.Store(conn.GetId(), connInfo)
	}

	addAddr(&ipContext.SrcIpAddrs, connInfo.srcAddr)
	addAddr(&ipContext.DstIpAddrs, connInfo.dstAddr)

	conn, err = next.Server(ctx).Request(ctx, request)
	if err != nil {
		if !loaded {
			mipam.free(connInfo)
		}
		return nil, err
	}

	return conn, nil
}

func (mipam *multipIpam) Close(
	ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	mipam.once.Do(mipam.init)
	if mipam.initErr != nil {
		return nil, mipam.initErr
	}

	if connInfo, ok := mipam.Load(conn.GetId()); ok {
		mipam.free(connInfo)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func getLastIP(ipNet *net.IPNet) string {
	out := make(net.IP, len(ipNet.IP))
	for i := 0; i < len(ipNet.IP); i++ {
		out[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}

	return addMaskIP(out)
}
func getFirstIP(ipNet *net.IPNet) string {
	return addMaskIP(ipNet.IP)
}

func addMaskIP(ip net.IP) string {
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}

func (mipam *multipIpam) free(connInfo *connectionInfo) {
	ipAddr, _, err := net.ParseCIDR(connInfo.dstAddr)
	if err == nil {
		connInfo.ipPool.AddNetString(addMaskIP(ipAddr))
	}
}

func (mipam *multipIpam) setMyIP(i int) error {
	myIP, err := mipam.ipPools[i].Pull()
	if err != nil {
		return err
	}
	if i >= len(mipam.myIPs) {
		mipam.myIPs = append(mipam.myIPs, myIP.String()+mipam.masks[i])
	} else {
		mipam.myIPs[i] = myIP.String() + mipam.masks[i]
	}
	return nil
}

func (mipam *multipIpam) getAddrs(excludeIP4, excludeIP6 *ippool.IPPool) (connInfo *connectionInfo, err error) {
	var dstAddr, srcAddr net.IP

	for i := 0; i < len(mipam.prefixes); i++ {
		// The NSE needs only one src address
		srcSet := false
		if i >= len(mipam.myIPs) {
			err = mipam.setMyIP(i)
			if err != nil {
				continue
			}
		}

		for {
			myIP, _, _ := net.ParseCIDR(mipam.myIPs[i])
			if !excludeIP4.ContainsString(myIP.String()) && !excludeIP6.ContainsString(myIP.String()) {
				srcAddr = myIP
				srcSet = true
				break
			}
			err = mipam.setMyIP(i)
			if err != nil {
				break
			}
		}
		for {
			if dstAddr, err = mipam.ipPools[i].Pull(); err != nil {
				break
			}
			if !excludeIP4.ContainsString(dstAddr.String()) && !excludeIP6.ContainsString(dstAddr.String()) {
				if srcSet {
					return &connectionInfo{
						ipPool:  mipam.ipPools[i],
						srcAddr: srcAddr.String() + mipam.masks[i],
						dstAddr: dstAddr.String() + mipam.masks[i],
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
