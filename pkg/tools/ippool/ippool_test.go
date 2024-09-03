// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
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

package ippool

import (
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

func TestIPPoolTool_Add(t *testing.T) {
	ipPool := New(net.IPv4len)

	ipPool.AddString("192.168.1.255")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddString("192.168.3.0")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddString("192.168.2.0")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddString("192.168.2.255")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_AddRange(t *testing.T) {
	ipPool := New(net.IPv4len)

	ipPool.AddNetString("192.168.1.0/31")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddNetString("192.168.10.0/24")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("192.168.1.0/24")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("192.168.11.0/24")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestGlobalCIDR(t *testing.T) {
	ipPool := NewWithNetString("192.168.0.0/16")
	require.True(t, ipPool.ContainsNetString("192.168.0.1/32"))
	require.False(t, ipPool.ContainsNetString("193.169.0.1/32"))
}

func TestIsItWorkCorrect(t *testing.T) {
	ipPool := NewWithNetString("192.168.0.0/16")
	ipPool.ExcludeString("192.168.0.0/30")
	p, err := ipPool.Pull()

	require.NoError(t, err)

	prefix := p.String() + "/24"
	fmt.Println(prefix)

	pool2 := NewWithNetString(prefix)
	pool2.ExcludeString("192.168.0.0/30")
	require.False(t, pool2.ContainsString("192.168.0.1"))
}

func TestIPPool_ExcludeRange(t *testing.T) {
	ipPool := NewWithNetString("192.168.0.0/16")

	for i := 0; i < 3; i++ {
		p, err := ipPool.Pull()

		require.NoError(t, err)

		prefix := p.String() + "/24"

		ipPool.ExcludeString(prefix)

		require.Equal(t, fmt.Sprintf("192.168.%v.0/24", i), prefix)
	}
}

func Test_IPPoolContains(t *testing.T) {
	ipPool := NewWithNetString("10.10.0.0/16")

	for i := 1; i <= 255; i++ {
		if i == 10 {
			continue
		}
		for j := 1; j <= 255; j++ {
			if j == 10 {
				continue
			}
			require.False(t, ipPool.ContainsNetString(fmt.Sprintf("%v.%v.0.0/24", i, j)))
		}
	}
	for i := 16; i < 32; i++ {
		ipNet := "10.10.0.0/" + fmt.Sprint(i)
		require.True(t, ipPool.ContainsNetString(ipNet), ipNet)
	}
	for i := 15; i > 0; i-- {
		ipNet := "10.10.0.0/" + fmt.Sprint(i)
		require.False(t, ipPool.ContainsNetString(ipNet), ipNet)
	}
}

func TestIPPoolTool_Contains(t *testing.T) {
	ipPool := NewWithNetString("192.168.0.0/16")

	require.True(t, ipPool.ContainsString("192.168.0.1"))
	require.True(t, ipPool.ContainsString("::ffff:192.168.0.1"))
	require.False(t, ipPool.ContainsString("192.167.0.1"))
}

func TestIPPoolTool_Exclude(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(t, err)
	ipPool := NewWithNet(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("192.255.0.0/16")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("192.0.1.0/24")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(2))

	_, ipNet, err = net.ParseCIDR("192.0.0.0/16")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("::ffff:192.2.0.0/112")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(2))
}


func TestIPPoolTool_Pull(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	ip, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.0")

	ipPool.ExcludeString("192.0.0.0/24")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.1.0")

	ipPool.AddNetString("192.0.0.6/31")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.6")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.7")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.1.1")

	ipPool.ExcludeString("::ffff:192.0.1.0/120")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.2.0")

	ipPool.ExcludeString("192.0.0.0/8")
	_, err = ipPool.Pull()
	require.Error(t, err)
}


func TestIPPoolTool_PullIP(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	ip, err := ipPool.PullIPString("192.0.0.10/32")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.10/32")

	ipPool.ExcludeString("192.0.0.0/24")
	ip, err = ipPool.PullIPString("192.0.1.10/32")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.1.10/32")

	ipPool.ExcludeString("192.0.2.0/24")
	_, err = ipPool.PullIPString("192.0.2.10/32")
	require.Error(t, err)
}


func TestIPPoolTool_GetPrefixes(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/16")
	require.NotNil(t, ipPool)

	prefixes := ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "192.0.0.0/16")

	ipPool.AddNetString("192.1.0.0/16")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "192.0.0.0/15")

	ipPool.AddNetString("192.2.0.0/31")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 2)
	require.Equal(t, prefixes[0], "192.0.0.0/15")
	require.Equal(t, prefixes[1], "192.2.0.0/31")

	ipPool.AddString("192.15.0.0")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 3)
	require.Equal(t, prefixes[0], "192.0.0.0/15")
	require.Equal(t, prefixes[1], "192.2.0.0/31")
	require.Equal(t, prefixes[2], "192.15.0.0/32")
}


func TestIPPoolTool_PullP2PAddrs(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	excludedPool := NewWithNetString("192.0.0.0/32")
	excluded2Pool := NewWithNetString("192.0.0.2/32")
	srcIPNet, dstIPNet, err := ipPool.PullP2PAddrs(excludedPool, excluded2Pool)
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "192.0.0.1/32")
	require.Equal(t, dstIPNet.String(), "192.0.0.3/32")

	srcIPNet, dstIPNet, err = ipPool.PullP2PAddrs()
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "192.0.0.0/32")
	require.Equal(t, dstIPNet.String(), "192.0.0.2/32")

	srcIP, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, srcIP.String(), "192.0.0.4")

	excludedPool = NewWithNetString("::ffff:192.0.0.0/120")
	excluded2Pool = NewWithNetString("192.0.1.1/32")
	srcIPNet, dstIPNet, err = ipPool.PullP2PAddrs(excludedPool, excluded2Pool)
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "192.0.1.0/32")
	require.Equal(t, dstIPNet.String(), "192.0.1.2/32")
}

func TestIPPoolTool_IPv6Add(t *testing.T) {
	ipPool := New(net.IPv6len)

	ipPool.Add(net.ParseIP("::1:ffff"))
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.Add(net.ParseIP("::1:0"))
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.Add(net.ParseIP("::2:0"))
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.Add(net.ParseIP("::0:ffff"))
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_IPv6AddRange(t *testing.T) {
	ipPool := New(net.IPv6len)

	ipPool.AddNetString("::1:0/127")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddNetString("::10:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("::1:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("::11:0/112")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_IPv6Contains(t *testing.T) {
	ipPool := NewWithNetString("::/64")

	require.True(t, ipPool.ContainsString("::0:1"))
	require.False(t, ipPool.ContainsString("0:1::0:1"))
}

func TestIPPoolTool_IPv6Exclude(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.ExcludeString("0:0:ffff:ffff::/64")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.ExcludeString("::1:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.ExcludeString("::/64")
	require.Equal(t, ipPool.size, uint64(1))
}


func TestIPPoolTool_IPv6Pull(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.NotNil(t, ipPool)

	ip, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::")

	ipPool.ExcludeString("::/112")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:0")

	ipPool.AddNetString("::6/127")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::6")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::7")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:1")

	ipPool.ExcludeString("::/32")
	_, err = ipPool.Pull()
	require.Error(t, err)
}


func TestIPPoolTool_IPv6PullIP(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.NotNil(t, ipPool)

	ip, err := ipPool.PullIPString("::10/128")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::10/128")

	ipPool.ExcludeString("::/120")
	ip, err = ipPool.PullIPString("::1010/128")
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1010/128")

	ipPool.ExcludeString("::2000/120")
	_, err = ipPool.PullIPString("::2010/128")
	require.Error(t, err)
}


func TestIPPoolTool_IPv6PullP2PAddrs(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.NotNil(t, ipPool)

	excludedPool := NewWithNetString("::/128")
	excluded2Pool := NewWithNetString("::2/128")
	srcIPNet, dstIPNet, err := ipPool.PullP2PAddrs(excludedPool, excluded2Pool)
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "::1/128")
	require.Equal(t, dstIPNet.String(), "::3/128")

	srcIPNet, dstIPNet, err = ipPool.PullP2PAddrs()
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "::/128")
	require.Equal(t, dstIPNet.String(), "::2/128")

	srcIP, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, srcIP.String(), "::4")
}


func TestIPPoolTool_IPv6GetPrefixes(t *testing.T) {
	ipPool := NewWithNetString("fe80::/112")
	require.NotNil(t, ipPool)

	prefixes := ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "fe80::/112")

	ipPool.AddNetString("fe80::1:0/112")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "fe80::/111")

	ipPool.AddNetString("fe80::3:0/112")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 2)
	require.Equal(t, prefixes[0], "fe80::/111")
	require.Equal(t, prefixes[1], "fe80::3:0/112")

	ipPool.AddString("fe80::15:0")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 3)
	require.Equal(t, prefixes[0], "fe80::/111")
	require.Equal(t, prefixes[1], "fe80::3:0/112")
	require.Equal(t, prefixes[2], "fe80::15:0/128")

	ipPool.AddNetString("fe80::/64")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "fe80::/64")

	ipPool.AddNetString("fe80:0:0:1::/64")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "fe80::/63")

	ipPool.AddNetString("fe80:0:0:2::/112")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 2)
	require.Equal(t, prefixes[0], "fe80::/63")
	require.Equal(t, prefixes[1], "fe80:0:0:2::/112")

	ipPool.AddNetString("fe7f:ffff:ffff:ffff:ffff:ffff:ffff:0/112")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 3)
	require.Equal(t, prefixes[0], "fe7f:ffff:ffff:ffff:ffff:ffff:ffff:0/112")
	require.Equal(t, prefixes[1], "fe80::/63")
	require.Equal(t, prefixes[2], "fe80:0:0:2::/112")
	ipPool.ExcludeString("fe7f:ffff:ffff:ffff:ffff:ffff:ffff:0/112")

	ipPool.AddString("fe80:0:0:15::")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 3)
	require.Equal(t, prefixes[0], "fe80::/63")
	require.Equal(t, prefixes[1], "fe80:0:0:2::/112")
	require.Equal(t, prefixes[2], "fe80:0:0:15::/128")

	ipPool.AddNetString("::/0")
	prefixes = ipPool.GetPrefixes()
	require.Equal(t, len(prefixes), 1)
	require.Equal(t, prefixes[0], "::/0")
}

func BenchmarkIPPool(b *testing.B) {
	b.Run("IPPool", func(b *testing.B) {
		benchmarkIPPool(b, b.N, runtime.GOMAXPROCS(0), 1000)
	})
	b.Run("PrefixPool", func(b *testing.B) {
		benchmarkPrefixPool(b, b.N, runtime.GOMAXPROCS(0), 1000)
	})
}

func benchmarkIPPool(b *testing.B, operations, threads, prefixes int) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(b, err)
	pool := NewWithNet(ipNet)
	ones, bits := ipNet.Mask.Size()

	wg := new(sync.WaitGroup)
	wg.Add(threads)
	var mtx sync.RWMutex
	mtx.Lock()

	f := func(t, operations int) {
		//nolint:gosec // Predictable random number generator is OK for testing purposes.
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
		ops := operations / threads
		if i < operations%threads {
			ops++
		}
		go f(i, ops)
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

	f := func(t, operations int) {
		//nolint:gosec // Predictable random number generator is OK for testing purposes.
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
			_, err = pool.ExcludePrefixes(*prefixes)
			require.NoError(b, err)
			_, _, _, err = pool.Extract("conn", networkservice.IpFamily_IPV4)
			require.NoError(b, err)
		}

		wg.Done()
	}
	for i := 0; i < threads; i++ {
		ops := operations / threads
		if i < operations%threads {
			ops++
		}
		go f(i, ops)
	}
	wg.Wait()
	wg.Add(threads)
	b.ResetTimer()
	mtx.Unlock()
	wg.Wait()
}

func generateSubnet(randSrc *rand.Rand, srcIP net.IP, ones, low, high int) *net.IPNet {
	length := low
	if high-low > 0 {
		length = randSrc.Intn(high-low+1) + low
	}
	ip := duplicateIP(srcIP)
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

func exclude(prefixes ...string) (ipv4exclude, ipv6exclude *IPPool) {
	ipv4exclude = New(net.IPv4len)
	ipv6exclude = New(net.IPv6len)
	for _, prefix := range prefixes {
		ipv4exclude.AddNetString(prefix)
		ipv6exclude.AddNetString(prefix)
	}
	return
}
