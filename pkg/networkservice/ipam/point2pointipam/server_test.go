// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package point2pointipam_test

import (
	"context"
	"encoding/binary"
	"github.com/RoaringBitmap/roaring"
	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
	"sync"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func newIpamServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		metadata.NewServer(),
		point2pointipam.NewServer(prefixes...),
	)
}

func newRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}
}

func validateConn(t *testing.T, conn *networkservice.Connection, dst, src string) {
	require.Equal(t, conn.Context.IpContext.DstIpAddr, dst)
	require.Equal(t, conn.Context.IpContext.DstRoutes, []*networkservice.Route{
		{
			Prefix: src,
		},
	})

	require.Equal(t, conn.Context.IpContext.SrcIpAddr, src)
	require.Equal(t, conn.Context.IpContext.SrcRoutes, []*networkservice.Route{
		{
			Prefix: dst,
		},
	})
}

func TestServer(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/32", "192.168.0.1/32")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.2/32", "192.168.0.3/32")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.0/32", "192.168.0.1/32")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.4/32", "192.168.0.5/32")
}

func TestNilPrefixes(t *testing.T) {
	srv := newIpamServer()
	_, err := srv.Request(context.Background(), newRequest())
	require.Error(t, err)

	_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
	require.NoError(t, err)

	srv = newIpamServer(nil, ipNet, nil)
	_, err = srv.Request(context.Background(), newRequest())
	require.Error(t, err)
}

func TestExclude32Prefix(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.1.0/32", "192.168.1.2/32")

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn2, err := srv.Request(context.Background(), req2)
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.1.4/32", "192.168.1.5/32")

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn3, err := srv.Request(context.Background(), req3)
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.1.7/32", "192.168.1.8/32")
}

func TestOutOfIPs(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.1.2/32", "192.168.1.3/32")

	req2 := newRequest()
	_, err = srv.Request(context.Background(), req2)
	require.Error(t, err)
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.2/31"}
	conn1, err := srv.Request(context.Background(), req1)
	require.Nil(t, conn1)
	require.Error(t, err)
}

func TestRefreshRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.0/32", "192.168.0.2/32")

	req = newRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.0/32", "192.168.0.2/32")

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.4/32", "192.168.0.5/32")
}

func BenchmarkIPPool(b *testing.B) {
	b.Run("IPPool/1Thread", func(b *testing.B) {
		benchmarkIPPool(b, b.N, 1, 500000)
	})
	b.Run("IPPool/2Threads", func(b *testing.B) {
		benchmarkIPPool(b, b.N, 2, 5000)
	})
	b.Run("IPPool/4Threads", func(b *testing.B) {
		benchmarkIPPool(b, b.N, 4, 2500)
	})
	b.Run("RoaringBitmap/1Thread", func(b *testing.B) {
		benchmarkRoaringBitmap(b, b.N, 1, 500000)
	})
	b.Run("RoaringBitmap/2Threads", func(b *testing.B) {
		benchmarkRoaringBitmap(b, b.N, 2, 5000)
	})
	b.Run("RoaringBitmap/4Threads", func(b *testing.B) {
		benchmarkRoaringBitmap(b, b.N, 4, 2500)
	})
	b.Run("PrefixPool/1Thread", func(b *testing.B) {
		benchmarkPrefixPool(b, b.N, 1, 10000)
	})
	b.Run("PrefixPool/2Threads", func(b *testing.B) {
		benchmarkPrefixPool(b, b.N, 2, 5000)
	})
	b.Run("PrefixPool/4Threads", func(b *testing.B) {
		benchmarkPrefixPool(b, b.N, 4, 2500)
	})
}

func benchmarkIPPool(b *testing.B, operations, threads, prefixes int) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(b, err)
	pool := ippool.NewWithNet(ipNet)
	ones, bits := ipNet.Mask.Size()

	wg := new(sync.WaitGroup)
	wg.Add(threads)
	var mtx sync.RWMutex
	mtx.Lock()

	f := func(t int) {
		randSrc := rand.New(rand.NewSource(int64(t)))

		var excludePrefxes []*[]string
		for op := 0; op < operations; op++ {
			var ex []string
			for i := 0; i < prefixes; i++ {
				excludeSubnet := generateSubnet(randSrc, ipNet.IP, ones, bits-8, bits)
				ex = append(ex, excludeSubnet.String())
			}
			excludePrefxes = append(excludePrefxes, &ex)
		}

		wg.Done()
		mtx.RLock()
		defer mtx.RUnlock()

		for _, iPrefixes := range excludePrefxes {
			excludeIP4, _ := exclude(*iPrefixes...)
			_, _, err := pool.PullP2PAddrs(excludeIP4)
			require.NoError(b, err)
		}

		wg.Done()
	}
	for i := 0; i < threads; i++ {
		go f(i)
	}
	wg.Wait()
	wg.Add(threads)
	b.ResetTimer()
	mtx.Unlock()
	wg.Wait()
}

func benchmarkRoaringBitmap(b *testing.B, operations, threads, prefixes int) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(b, err)
	pool := point2pointipam.NewIPPool(ipNet)
	ones, bits := ipNet.Mask.Size()

	wg := new(sync.WaitGroup)
	wg.Add(threads)
	var mtx sync.RWMutex
	mtx.Lock()

	f := func(t int) {
		randSrc := rand.New(rand.NewSource(int64(t)))

		var excludePrefxes []*[]string
		for op := 0; op < operations; op++ {
			var ex []string
			for i := 0; i < prefixes; i++ {
				excludeSubnet := generateSubnet(randSrc, ipNet.IP, ones, bits-8, bits)
				ex = append(ex, excludeSubnet.String())
			}
			excludePrefxes = append(excludePrefxes, &ex)
		}

		wg.Done()
		mtx.RLock()
		defer mtx.RUnlock()

		for _, prefixes := range excludePrefxes {
			exclude, _ := excludeBitmap(*prefixes...)
			_, _, err := pool.GetP2PAddrs(exclude)
			require.NoError(b, err)
		}

		//println(src.String(), dst.String())
		wg.Done()
	}
	for i := 0; i < threads; i++ {
		go f(i)
	}
	wg.Wait()
	wg.Add(threads)
	b.ResetTimer()
	mtx.Unlock()
	wg.Wait()
}

func benchmarkPrefixPool(b *testing.B, operations, threads, prefixes int) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(b, err)
	pool, err := prefixpool.New("192.0.0.0/8")
	require.NoError(b, err)
	ones, bits := ipNet.Mask.Size()

	wg := new(sync.WaitGroup)
	wg.Add(threads)
	var mtx sync.RWMutex
	mtx.Lock()

	f := func(t int) {
		randSrc := rand.New(rand.NewSource(int64(t)))

		var excludePrefxes []*[]string
		for op := 0; op < operations; op++ {
			var ex []string
			for i := 0; i < prefixes; i++ {
				excludeSubnet := generateSubnet(randSrc, ipNet.IP, ones, bits-8, bits)
				ex = append(ex, excludeSubnet.String())
			}
			excludePrefxes = append(excludePrefxes, &ex)
		}

		wg.Done()
		mtx.RLock()
		defer mtx.RUnlock()

		for _, prefixes := range excludePrefxes {
			pool.ExcludePrefixes(*prefixes)
			_, _, _, err = pool.Extract("conn", networkservice.IpFamily_IPV4)
			require.NoError(b, err)
		}

		//println(src.String(), dst.String())
		wg.Done()
	}
	for i := 0; i < threads; i++ {
		go f(i)
	}
	wg.Wait()
	wg.Add(threads)
	b.ResetTimer()
	mtx.Unlock()
	wg.Wait()

	/*randSrc := rand.New(rand.NewSource(0))

	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(t, err)
	pool, err := prefixpool.New("192.0.0.0/8")
	require.NoError(t, err)
	ones, bits := ipNet.Mask.Size()

	var excludePrefxes []string
	for i := 0; i < 1000000; i++ {
		excludeSubnet := generateSubnet(randSrc, ipNet.IP, ones, bits-8, bits)
		excludePrefxes = append(excludePrefxes, excludeSubnet.String())
	}
	now := time.Now()
	for i := 0; i < len(excludePrefxes); i++ {
		pool.ExcludePrefixes([]string{excludePrefxes[i]})
		if time.Now().Sub(now) > time.Second {
			break
		}
	}

	src, dst, _, err := pool.Extract("conn", networkservice.IpFamily_IPV4)
	//pool.ReleaseExcludedPrefixes(excludePrefxes)
	require.NoError(t, err)
	println(time.Now().Sub(now).String())
	println(src, dst)*/
}

func generateSubnet(randSrc *rand.Rand, srcIp net.IP, ones, low, high int) *net.IPNet {
	length := low
	if high-low > 0 {
		length = randSrc.Intn(high-low+1) + low
	}
	ip := duplicateIP(srcIp)
	for i := ones; i < length; i++ {
		ip[i/8] >>= 8 - i%8
		ip[i/8] <<= 1
		ip[i/8] += byte(randSrc.Intn(2))
		ip[i/8] <<= 8 - i%8 - 1
	}
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(length, len(ip)*8),
	}
}

func duplicateIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func exclude(prefixes ...string) (ipv4exclude *ippool.IPPool, ipv6exclude *ippool.IPPool) {
	ipv4exclude = ippool.New(net.IPv4len)
	ipv6exclude = ippool.New(net.IPv6len)
	for _, prefix := range prefixes {
		ipv4exclude.AddNetString(prefix)
		ipv6exclude.AddNetString(prefix)
	}
	return
}

func excludeBitmap(prefixes ...string) (*roaring.Bitmap, error) {
	exclude := roaring.New()
	for _, prefix := range prefixes {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}
		low := binary.BigEndian.Uint32(cidr.NetworkAddress(ipNet).To4())
		high := binary.BigEndian.Uint32(cidr.BroadcastAddress(ipNet).To4()) + 1
		exclude.AddRange(uint64(low), uint64(high))
	}
	return exclude, nil
}

func addrToInt(addr string) uint32 {
	ip, _, _ := net.ParseCIDR(addr)
	return binary.BigEndian.Uint32(ip.To4())
}
