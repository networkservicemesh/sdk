// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package prefixpool_test

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/prefixpool"
)

func TestEmptryPrefixPoolIsNotPanics(t *testing.T) {
	var p *prefixpool.PrefixPool

	var err error

	require.NotPanics(t, func() {
		p, err = prefixpool.New()
	})
	require.Nil(t, err)
	require.NotNil(t, p)
}

func TestNetExtractIPv4(t *testing.T) {
	testNetExtract(t, "10.10.1.0/24", "10.10.1.1/30", "10.10.1.2/30", networkservice.IpFamily_IPV4)
}

func TestNetExtractIPv6(t *testing.T) {
	testNetExtract(t, "100::/64", "100::1/126", "100::2/126", networkservice.IpFamily_IPV6)
}

func testNetExtract(t *testing.T, inPool, srcDesired, dstDesired string, family networkservice.IpFamily_Family) {
	pool, err := prefixpool.New(inPool)
	require.Nil(t, err)

	srcIP, dstIP, requested, err := pool.Extract("c1", family)
	require.Nil(t, err)
	require.Nil(t, requested)

	require.Equal(t, srcDesired, srcIP.String())
	require.Equal(t, dstDesired, dstIP.String())

	err = pool.Release("c1")
	require.Nil(t, err)
}

func TestExtractPrefixes_1_ipv4(t *testing.T) {
	newPrefixes, prefixes, err := prefixpool.ExtractPrefixes([]string{"10.10.1.0/24"},
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
}

func TestExtractPrefixes_1_ipv6(t *testing.T) {
	newPrefixes, prefixes, err := prefixpool.ExtractPrefixes([]string{"100::/64"},
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
}

func TestIntersect1(t *testing.T) {
	pp, err := prefixpool.New("10.10.1.0/24")
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
	pp, err := prefixpool.New("10.10.1.0/24", "10.32.1.0/16")
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
	pool, err := prefixpool.New("10.20.0.0/16")
	require.Nil(t, err)

	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	excluded, err := pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23"}, pool.GetPrefixes())

	err = pool.ReleaseExcludedPrefixes(excluded)
	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())
}

func TestReleaseExcludePrefixesNestedNetworks(t *testing.T) {
	pool, err := prefixpool.New("10.20.4.1/22", "127.0.0.1/22")
	require.Nil(t, err)

	expectedPrefixes := []string{"10.20.0.0/16", "127.0.0.0/22"}
	prefixesToRelease := []string{"10.20.0.1/21", "10.20.2.1/21", "10.20.2.1/16"}

	err = pool.ReleaseExcludedPrefixes(prefixesToRelease)
	require.Nil(t, err)
	require.Equal(t, expectedPrefixes, pool.GetPrefixes())

	err = pool.ReleaseExcludedPrefixes(prefixesToRelease)
	require.Nil(t, err)
	require.Equal(t, expectedPrefixes, pool.GetPrefixes())
}

func TestReleaseExcludePrefixesNoOverlap(t *testing.T) {
	pool, err := prefixpool.New("10.20.0.0/16")
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
	pool, err := prefixpool.New("10.20.0.0/16", "2.20.0.0/16")
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
	pool, err := prefixpool.New("10.20.0.0/16", "10.32.0.0/16")
	require.Nil(t, err)

	excludedPrefix := []string{"10.20.1.10/24", "10.20.32.0/19"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.8.0/21", "10.20.4.0/22", "10.20.2.0/23", "10.32.0.0/16"}, pool.GetPrefixes())
}

func TestExcludePrefixesPartialOverlapSmallNetworks(t *testing.T) {
	pool, err := prefixpool.New("10.20.0.0/16")
	require.Nil(t, err)

	excludedPrefix := []string{"10.20.1.0/30", "10.20.10.0/30", "10.20.20.0/30", "10.20.20.20/30", "10.20.40.20/30"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	require.Nil(t, err)
	require.Equal(t, []string{"10.20.32.0/21", "10.20.40.0/28", "10.20.40.16/30", "10.20.48.0/20", "10.20.44.0/22", "10.20.42.0/23", "10.20.41.0/24", "10.20.40.128/25", "10.20.40.64/26", "10.20.40.32/27", "10.20.40.24/29", "10.20.20.16/30", "10.20.20.24/29", "10.20.16.0/22", "10.20.24.0/21", "10.20.22.0/23", "10.20.21.0/24", "10.20.20.128/25", "10.20.20.64/26", "10.20.20.32/27", "10.20.20.8/29", "10.20.20.4/30", "10.20.8.0/23", "10.20.12.0/22", "10.20.11.0/24", "10.20.10.128/25", "10.20.10.64/26", "10.20.10.32/27", "10.20.10.16/28", "10.20.10.8/29", "10.20.10.4/30", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.4.0/22", "10.20.2.0/23", "10.20.1.128/25", "10.20.1.64/26", "10.20.1.32/27", "10.20.1.16/28", "10.20.1.8/29", "10.20.1.4/30"}, pool.GetPrefixes())
}

func TestExcludePrefixesNoOverlap(t *testing.T) {
	pool, err := prefixpool.New("10.20.0.0/16")
	require.Nil(t, err)

	excludedPrefix := []string{"10.32.1.0/16"}

	_, err = pool.ExcludePrefixes(excludedPrefix)
	require.Nil(t, err)

	require.Equal(t, []string{"10.20.0.0/16"}, pool.GetPrefixes())
}

func TestExcludePrefixesFullOverlap(t *testing.T) {
	pool, err := prefixpool.New("10.20.0.0/24")
	require.Nil(t, err)

	excludedPrefix := []string{"10.20.1.0/16"}

	_, err = pool.ExcludePrefixes(excludedPrefix)

	require.Equal(t, "IPAM: The available address pool is empty, probably intersected by excludedPrefix", err.Error())
}

func TestPrefixPoolValidation(t *testing.T) {
	_, err := prefixpool.New("10.20.0.0/24")
	require.Nil(t, err)
	_, err = prefixpool.New("10.20.0.0/56")

	if assert.Error(t, err) {
		expectedErr := &net.ParseError{Type: "CIDR address", Text: "10.20.0.0/56"}
		withStachErr := errors.Wrapf(expectedErr, "failed to parse %s as CIDR", "10.20.0.0/56")
		require.Equal(t, withStachErr.Error(), err.Error())
	}
}
