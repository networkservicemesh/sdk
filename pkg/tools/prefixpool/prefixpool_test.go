// Copyright (c) 2020 Cisco Systems, Inc.
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

package prefixpool

import (
	"net"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

func TestPrefixPoolSubnet1(t *testing.T) {
	prefixes := []string{"10.10.1.0/24"}
	logrus.Printf("Address count: %d", AddressCount(prefixes...))
	require.Equal(t, uint64(256), AddressCount(prefixes...))

	_, snet1, _ := net.ParseCIDR("10.10.1.0/24")
	sn1, err := subnet(snet1, 0)
	require.Nil(t, err)
	logrus.Printf(sn1.String())
	require.Equal(t, "10.10.1.0/25", sn1.String())
	s, e := AddressRange(sn1)
	require.Equal(t, "10.10.1.0", s.String())
	require.Equal(t, "10.10.1.127", e.String())
	require.Equal(t, uint64(128), addressCount(sn1.String()))

	lastIP := s
	for i := uint64(0); i < addressCount(sn1.String())-1; i++ {
		ip, incr_err := IncrementIP(lastIP, sn1)
		require.Nil(t, incr_err)
		lastIP = ip
	}

	_, err = IncrementIP(lastIP, sn1)
	require.Equal(t, "Overflowed CIDR while incrementing IP", err.Error())

	sn2, err := subnet(snet1, 1)
	require.Nil(t, err)
	logrus.Printf(sn2.String())
	require.Equal(t, "10.10.1.128/25", sn2.String())
	s, e = AddressRange(sn2)
	require.Equal(t, "10.10.1.128", s.String())
	require.Equal(t, "10.10.1.255", e.String())
	require.Equal(t, uint64(128), addressCount(sn2.String()))
}

func TestNetExtractIPv4(t *testing.T) {
	testNetExtract(t, "10.10.1.0/24", "10.10.1.1/30", "10.10.1.2/30", networkservice.IpFamily_IPV4)
}

func TestNetExtractIPv6(t *testing.T) {
	testNetExtract(t, "100::/64", "100::1/126", "100::2/126", networkservice.IpFamily_IPV6)
}

func testNetExtract(t *testing.T, inPool, srcDesired, dstDesired string, family networkservice.IpFamily_Family) {
	pool, err := NewPrefixPool(inPool)
	require.Nil(t, err)

	srcIP, dstIP, requested, err := pool.Extract("c1", family)
	require.Nil(t, err)
	require.Nil(t, requested)

	require.Equal(t, srcDesired, srcIP.String())
	require.Equal(t, dstDesired, dstIP.String())

	err = pool.Release("c1")
	require.Nil(t, err)
}

func TestExtract1(t *testing.T) {
	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.0/25"}, "10.10.1.127/25")
	require.Nil(t, err)
	require.Equal(t, []string{"10.10.1.0/24"}, newPrefixes)
	logrus.Printf("%v", newPrefixes)
}

func TestExtractPrefixes_1_ipv4(t *testing.T) {
	newPrefixes, prefixes, err := ExtractPrefixes([]string{"10.10.1.0/24"},
		&networkservice.ExtraPrefixRequest{
			AddrFamily:      &networkservice.IpFamily{Family: networkservice.IpFamily_IPV4},
			RequiredNumber:  10,
			RequestedNumber: 20,
			PrefixLen:       31,
		},
	)
	require.Nil(t, err)
	require.Equal(t, 20, len(newPrefixes))
	require.Equal(t, 4, len(prefixes))
	logrus.Printf("%v", newPrefixes)
}

func TestExtractPrefixes_1_ipv6(t *testing.T) {
	newPrefixes, prefixes, err := ExtractPrefixes([]string{"100::/64"},
		&networkservice.ExtraPrefixRequest{
			AddrFamily:      &networkservice.IpFamily{Family: networkservice.IpFamily_IPV6},
			RequiredNumber:  100,
			RequestedNumber: 200,
			PrefixLen:       128,
		},
	)
	require.Nil(t, err)
	require.Equal(t, 200, len(newPrefixes))
	require.Equal(t, 59, len(prefixes))
	logrus.Printf("%v", newPrefixes)
}

func TestExtract2(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 24)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.0/24", prefix)
	require.Equal(t, 0, len(prefixes))
}

func TestExtract2_ipv6(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"100::/64"}, 65)
	require.Nil(t, err)
	require.Equal(t, "100::/65", prefix)
	require.Equal(t, 1, len(prefixes))
	require.Equal(t, "100::8000:0:0:0/65", prefixes[0])
}

func TestExtract3_ipv6(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"100::/64"}, 128)
	require.Nil(t, err)
	require.Equal(t, "100::/128", prefix)
	require.Equal(t, 64, len(prefixes))
}

func TestRelease_ipv6(t *testing.T) {
	prefixes := []string{
		"100::1/128",
		"100::2/127",
		"100::4/126",
		"100::8/125",
		"100::10/124",
		"100::20/123",
		"100::40/122",
		"100::80/121",
		"100::100/120",
		"100::200/119",
		"100::400/118",
		"100::800/117",
		"100::1000/116",
		"100::2000/115",
		"100::4000/114",
		"100::8000/113",
		"100::1:0/112",
		"100::2:0/111",
		"100::4:0/110",
		"100::8:0/109",
		"100::10:0/108",
		"100::20:0/107",
		"100::40:0/106",
		"100::80:0/105",
		"100::100:0/104",
		"100::200:0/103",
		"100::400:0/102",
		"100::800:0/101",
		"100::1000:0/100",
		"100::2000:0/99",
		"100::4000:0/98",
		"100::8000:0/97",
		"100::1:0:0/96",
		"100::2:0:0/95",
		"100::4:0:0/94",
		"100::8:0:0/93",
		"100::10:0:0/92",
		"100::20:0:0/91",
		"100::40:0:0/90",
		"100::80:0:0/89",
		"100::100:0:0/88",
		"100::200:0:0/87",
		"100::400:0:0/86",
		"100::800:0:0/85",
		"100::1000:0:0/84",
		"100::2000:0:0/83",
		"100::4000:0:0/82",
		"100::8000:0:0/81",
		"100::1:0:0:0/80",
		"100::2:0:0:0/79",
		"100::4:0:0:0/78",
		"100::8:0:0:0/77",
		"100::10:0:0:0/76",
		"100::20:0:0:0/75",
		"100::40:0:0:0/74",
		"100::80:0:0:0/73",
		"100::100:0:0:0/72",
		"100::200:0:0:0/71",
		"100::400:0:0:0/70",
		"100::800:0:0:0/69",
		"100::1000:0:0:0/68",
		"100::2000:0:0:0/67",
		"100::4000:0:0:0/66",
		"100::8000:0:0:0/65",
	}
	released, err := ReleasePrefixes(prefixes, "100::/128")
	require.Nil(t, err)
	require.Equal(t, 1, len(released))
	require.Equal(t, "100::/64", released[0])
}

func TestExtract3(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 23)
	require.Equal(t, "Failed to find room to have prefix len 23 at [10.10.1.0/24]", err.Error())
	require.Equal(t, "", prefix)
	require.Equal(t, 1, len(prefixes))
}

func TestExtract4(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 25)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.0/25", prefix)
	require.Equal(t, []string{"10.10.1.128/25"}, prefixes)
}

func TestExtract5(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 26)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.0/26", prefix)
	require.Equal(t, []string{"10.10.1.64/26", "10.10.1.128/25"}, prefixes)
}

func TestExtract6(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.0/24"}, 32)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.0/32", prefix)
	require.Equal(t, []string{"10.10.1.1/32", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}, prefixes)
}

func TestExtract7(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.1/32", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}, 31)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.2/31", prefix)
	require.Equal(t, []string{"10.10.1.1/32", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26", "10.10.1.128/25"}, prefixes)
}
func TestExtract8(t *testing.T) {
	prefix, prefixes, err := ExtractPrefix([]string{"10.10.1.128/25", "10.10.1.2/31", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26"}, 32)
	require.Nil(t, err)
	require.Equal(t, "10.10.1.2/32", prefix)
	require.Equal(t, []string{"10.10.1.128/25", "10.10.1.3/32", "10.10.1.4/30", "10.10.1.8/29", "10.10.1.16/28", "10.10.1.32/27", "10.10.1.64/26"}, prefixes)
}

func TestRelease1(t *testing.T) {
	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.0/25"}, "10.10.1.127/25")
	require.Nil(t, err)
	require.Equal(t, []string{"10.10.1.0/24"}, newPrefixes)
}

func TestRelease2(t *testing.T) {
	_, snet, _ := net.ParseCIDR("10.10.1.0/25")
	sn1, _ := subnet(snet, 0)
	sn2, _ := subnet(snet, 1)
	logrus.Printf("%v %v", sn1.String(), sn2.String())

	sn10 := clearNetIndexInIP(sn1.IP, 26)
	sn11 := clearNetIndexInIP(sn1.IP, 26)
	logrus.Printf("%v %v", sn10.String(), sn11.String())

	sn20 := clearNetIndexInIP(sn2.IP, 26)
	sn21 := clearNetIndexInIP(sn2.IP, 26)
	logrus.Printf("%v %v", sn20.String(), sn21.String())

	newPrefixes, err := ReleasePrefixes([]string{"10.10.1.64/26", "10.10.1.128/25"}, "10.10.1.0/26")
	require.Nil(t, err)
	require.Equal(t, []string{"10.10.1.0/24"}, newPrefixes)
}

func TestIntersect1(t *testing.T) {
	pp, err := NewPrefixPool("10.10.1.0/24")
	require.Nil(t, err)

	x, _ := pp.Intersect("10.10.1.0/28")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.1.10/28")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.1.0/10")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.0.0/10")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.0.0/24")
	require.False(t, x)
	x, _ = pp.Intersect("10.10.1.0/24")
	require.True(t, x)
}

func TestIntersect2(t *testing.T) {
	pp, err := NewPrefixPool("10.10.1.0/24", "10.32.1.0/16")
	require.Nil(t, err)

	x, _ := pp.Intersect("10.10.1.0/28")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.1.10/28")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.1.0/10")
	require.True(t, x)
	x, _ = pp.Intersect("10.10.0.0/10")
	require.True(t, x)
	x, _ = pp.Intersect("10.32.0.0/10")
	require.True(t, x)
	x, _ = pp.Intersect("10.32.0.0/24")
	require.True(t, x)
	x, _ = pp.Intersect("10.2.0.0/16")
	require.False(t, x)
}

func TestReleaseExcludePrefixes(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23"}, pool.GetPrefixes())

	err = pool.ReleaseExcludedPrefixes(excluded)
	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())
}

func TestReleaseExcludePrefixesNoOverlap(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"10.32.0.0/16"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())

	err = pool.ReleaseExcludedPrefixes(excluded)
	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())
}

func TestReleaseExcludePrefixesFullOverlap(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16", "2.20.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"2.20.0.0/8"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())

	err = pool.ReleaseExcludedPrefixes(excluded)
	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16", "2.20.0.0/16"}, pool.GetPrefixes())
}

func TestExcludePrefixesPartialOverlap(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16", "10.32.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23", "10.32.0.0/16"}, pool.GetPrefixes())
}

func TestExcludePrefixesPartialOverlapSmallNetworks(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"10.20.1.0/30", "10.20.10.0/30", "10.20.20.0/30", "10.20.20.20/30", "10.20.40.20/30"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.32.0/21", "10.20.40.0/28", "10.20.40.16/30", "10.20.48.0/20", "10.20.44.0/22", "10.20.42.0/23", "10.20.41.0/24", "10.20.40.128/25", "10.20.40.64/26", "10.20.40.32/27", "10.20.40.24/29", "10.20.20.16/30", "10.20.20.24/29", "10.20.16.0/22", "10.20.24.0/21", "10.20.22.0/23", "10.20.21.0/24", "10.20.20.128/25", "10.20.20.64/26", "10.20.20.32/27", "10.20.20.8/29", "10.20.20.4/30", "10.20.8.0/23", "10.20.12.0/22", "10.20.11.0/24", "10.20.10.128/25", "10.20.10.64/26", "10.20.10.32/27", "10.20.10.16/28", "10.20.10.8/29", "10.20.10.4/30", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.4.0/22", "10.20.2.0/23", "10.20.1.128/25", "10.20.1.64/26", "10.20.1.32/27", "10.20.1.16/28", "10.20.1.8/29", "10.20.1.4/30"}, pool.GetPrefixes())
}

func TestExcludePrefixesNoOverlap(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/16")
	require.Nil(t, err)
	excludedPrefix := []string{"10.32.1.0/16"}

	_, _ = pool.ExcludePrefixes(excludedPrefix)

	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())
}

func TestExcludePrefixesFullOverlap(t *testing.T) {
	pool, err := NewPrefixPool("10.20.0.0/24")
	require.Nil(t, err)
	excludedPrefix := []string{"10.20.1.0/16"}

	_, _ = pool.ExcludePrefixes(excludedPrefix)

	require.Equal(t, "IPAM: The available address pool is empty, probably intersected by excludedPrefix", err.Error())
}

func TestNewPrefixPoolReader(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}
	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")

	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	f, err := os.Create(configPath)
	require.Nil(t, err)
	defer os.Remove(configPath)
	_, err = f.WriteString(testConfig)
	require.Nil(t, err)
	err = f.Close()
	require.Nil(t, err)

	prefixPool := NewPrefixPoolReader(configPath)

	require.Equal(t, prefixPool.GetPrefixes(), prefixes)
}
