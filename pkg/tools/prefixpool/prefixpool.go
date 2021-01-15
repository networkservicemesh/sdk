// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

// Package prefixpool provides service for prefix managing
package prefixpool

import (
	"math/big"
	"net"
	"sort"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
)

// PrefixPool is a structure that contains information about prefixes
type PrefixPool struct {
	mutex       sync.RWMutex
	prefixes    []string
	connections map[string]*connectionRecord
}

// GetPrefixes returns the list of saved prefixes
func (impl *PrefixPool) GetPrefixes() []string {
	impl.mutex.Lock()
	copyArray := make([]string, len(impl.prefixes))
	copy(copyArray, impl.prefixes)
	impl.mutex.Unlock()
	return copyArray
}

type connectionRecord struct {
	ipNet    *net.IPNet
	prefixes []string
}

// New is a PrefixPool constructor
func New(prefixes ...string) (*PrefixPool, error) {
	for _, prefix := range prefixes {
		_, _, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}
	}
	return &PrefixPool{
		prefixes:    prefixes,
		connections: map[string]*connectionRecord{},
	}, nil
}

// ReleaseExcludedPrefixes releases excluded prefixes back the pool of available ones
func (impl *PrefixPool) ReleaseExcludedPrefixes(excludedPrefixes []string) error {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()

	remaining, err := releasePrefixes(impl.prefixes, excludedPrefixes...)
	if err != nil {
		return err
	}
	/* Sort the prefixes, so their order is consistent during unit testing */
	sort.Slice(remaining, func(i, j int) bool { return remaining[i] < remaining[j] })
	impl.prefixes = remaining
	return nil
}

// ExcludePrefixes excludes prefixes from the pool of available prefixes
func (impl *PrefixPool) ExcludePrefixes(excludedPrefixes []string) (removedPrefixesList []string, retErr error) {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()
	/* Use a working copy for the available prefixes */
	copyPrefixes := append([]string{}, impl.prefixes...)

	removedPrefixes := []string{}

	for _, excludedPrefix := range excludedPrefixes {
		splittedEntries := []string{}
		prefixesToRemove := []string{}
		_, subnetExclude, _ := net.ParseCIDR(excludedPrefix)

		/* 1. Check if each excluded entry overlaps with the available prefix */
		for _, prefix := range copyPrefixes {
			intersecting := false
			excludedIsBigger := false
			_, subnetPrefix, _ := net.ParseCIDR(prefix)
			intersecting, excludedIsBigger = intersect(subnetExclude, subnetPrefix)
			/* 1.1. If intersecting, check which one is bigger */
			if intersecting {
				/* 1.1.1. If excluded is bigger, we remove the original entry */
				if !excludedIsBigger {
					/* 1.1.2. If the original entry is bigger, we split it and remove the avoided range */
					res, err := extractSubnet(subnetPrefix, subnetExclude)
					if err != nil {
						return nil, err
					}
					/* 1.1.3. Collect the resulted split prefixes */
					splittedEntries = append(splittedEntries, res...)
					/* 1.1.5. Collect the actual excluded prefixes that should be added back to the original pool */
					removedPrefixes = append(removedPrefixes, subnetExclude.String())
				} else {
					/* 1.1.5. Collect the actual excluded prefixes that should be added back to the original pool */
					removedPrefixes = append(removedPrefixes, subnetPrefix.String())
				}
				/* 1.1.4. Collect prefixes that should be removed from the original pool */
				prefixesToRemove = append(prefixesToRemove, subnetPrefix.String())

				break
			}
			/* 1.2. If not intersecting, proceed verifying the next one */
		}
		/* 2. Keep only the prefixes that should not be removed from the original pool */
		if len(prefixesToRemove) != 0 {
			for _, presentPrefix := range copyPrefixes {
				prefixFound := false
				for _, prefixToRemove := range prefixesToRemove {
					if presentPrefix == prefixToRemove {
						prefixFound = true
						break
					}
				}
				if !prefixFound {
					/* 2.1. Add the original non-split prefixes to the split ones */
					splittedEntries = append(splittedEntries, presentPrefix)
				}
			}
			/* 2.2. Update the original prefix list */
			copyPrefixes = splittedEntries
		}
	}
	/* Raise an error, if there aren't any available prefixes left after excluding */
	if len(copyPrefixes) == 0 {
		err := errors.New("IPAM: The available address pool is empty, probably intersected by excludedPrefix")
		return nil, err
	}
	/* Everything should be fine, update the available prefixes with what's left */
	impl.prefixes = copyPrefixes
	return removedPrefixes, nil
}

/* Split the wider range removing the avoided smaller range from it */
func extractSubnet(wider, smaller *net.IPNet) (retSubnets []string, retErr error) {
	root := wider
	prefixLen, _ := smaller.Mask.Size()
	leftParts, rightParts := []string{}, []string{}
	for {
		rootLen, _ := root.Mask.Size()
		if rootLen == prefixLen {
			// we found the required prefix
			break
		}
		sub1, err := subnet(root, 0)
		if err != nil {
			return nil, err
		}
		sub2, err := subnet(root, 1)
		if err != nil {
			return nil, err
		}
		switch {
		case sub1.Contains(smaller.IP):
			rightParts = append(rightParts, sub2.String())
			root = sub1
		case sub2.Contains(smaller.IP):
			leftParts = append(leftParts, sub1.String())
			root = sub2
		default:
			return nil, errors.New("split failed")
		}
	}
	return append(leftParts, rightParts...), nil
}

// Extract extracts source and destination from the given connection
func (impl *PrefixPool) Extract(connectionID string, family networkservice.IpFamily_Family, requests ...*networkservice.ExtraPrefixRequest) (srcIP, dstIP *net.IPNet, requested []string, err error) {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()

	prefixLen := 30 // At lest 4 addresses
	if family == networkservice.IpFamily_IPV6 {
		prefixLen = 126
	}
	result, remaining, err := ExtractPrefixes(impl.prefixes, &networkservice.ExtraPrefixRequest{
		RequiredNumber:  1,
		RequestedNumber: 1,
		PrefixLen:       uint32(prefixLen),
		AddrFamily:      &networkservice.IpFamily{Family: family},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	ip, ipNet, err := net.ParseCIDR(result[0])
	if err != nil {
		return nil, nil, nil, err
	}

	src, err := incrementIP(ip, ipNet)
	if err != nil {
		return nil, nil, nil, err
	}

	dst, err := incrementIP(src, ipNet)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(requests) > 0 {
		requested, remaining, err = ExtractPrefixes(remaining, requests...)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	impl.prefixes = remaining

	impl.connections[connectionID] = &connectionRecord{
		ipNet:    ipNet,
		prefixes: requested,
	}
	return &net.IPNet{IP: src, Mask: ipNet.Mask}, &net.IPNet{IP: dst, Mask: ipNet.Mask}, requested, nil
}

// Release releases prefixes from the connection
func (impl *PrefixPool) Release(connectionID string) error {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()

	conn := impl.connections[connectionID]
	if conn == nil {
		return errors.Errorf("Failed to release connection infomration: %s", connectionID)
	}
	delete(impl.connections, connectionID)

	remaining, err := releasePrefixes(impl.prefixes, conn.prefixes...)
	if err != nil {
		return err
	}

	remaining, err = releasePrefixes(remaining, conn.ipNet.String())
	if err != nil {
		return err
	}

	impl.prefixes = remaining
	return nil
}

// GetConnectionInformation returns information about connection
func (impl *PrefixPool) GetConnectionInformation(connectionID string) (ipNet string, prefixes []string, err error) {
	impl.mutex.RLock()
	defer impl.mutex.RUnlock()
	conn := impl.connections[connectionID]
	if conn == nil {
		return "", nil, errors.Errorf("No connection with id: %s is found", connectionID)
	}
	return conn.ipNet.String(), conn.prefixes, nil
}

// Intersect returns is there any intersection with existing prefixes
func (impl *PrefixPool) Intersect(prefix string) (intersection bool, err error) {
	_, subnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false, err
	}

	for _, p := range impl.prefixes {
		_, sn, _ := net.ParseCIDR(p)
		if ret, _ := intersect(sn, subnet); ret {
			return true, nil
		}
	}
	return false, nil
}

func intersect(first, second *net.IPNet) (contains, isFirstBigger bool) {
	f, _ := first.Mask.Size()
	s, _ := second.Mask.Size()
	firstIsBigger := false

	var widerRange, narrowerRange *net.IPNet
	if f < s {
		widerRange, narrowerRange = first, second
		firstIsBigger = true
	} else {
		widerRange, narrowerRange = second, first
	}

	return widerRange.Contains(narrowerRange.IP), firstIsBigger
}

// ExtractPrefixes extracts prefixes from given requests
func ExtractPrefixes(prefixes []string, requests ...*networkservice.ExtraPrefixRequest) (requested, remaining []string, err error) {
	// Check if requests are valid.
	for _, request := range requests {
		err := request.IsValid()
		if err != nil {
			return nil, prefixes, err
		}
	}

	// 1 find prefix of desired prefix len and return it
	result := []string{}

	// Make a copy since we need to fit all prefixes before we could finish.
	newPrefixes := []string{}
	newPrefixes = append(newPrefixes, prefixes...)

	// We need to firstly find required prefixes available.
	for _, request := range requests {
		for i := uint32(0); i < request.RequiredNumber; i++ {
			prefix, leftPrefixes, err := extractPrefix(newPrefixes, request.PrefixLen)
			if err != nil {
				return nil, prefixes, err
			}
			result = append(result, prefix)
			newPrefixes = leftPrefixes
		}
	}
	// We need to fit some more prefixes up to Requested ones
	for _, request := range requests {
		for i := request.RequiredNumber; i < request.RequestedNumber; i++ {
			prefix, leftPrefixes, err := extractPrefix(newPrefixes, request.PrefixLen)
			if err != nil {
				// It seems there is no more prefixes available, but since we have all Required already we could go.
				break
			}
			result = append(result, prefix)
			newPrefixes = leftPrefixes
		}
	}
	if len(result) == 0 {
		return nil, prefixes, errors.Errorf("Failed to extract prefixes, there is no available %v", prefixes)
	}
	return result, newPrefixes, nil
}

func extractPrefix(prefixes []string, prefixLen uint32) (retPrefix string, retLeftPrefixes []string, retError error) {
	// Check if we already have required CIDR
	maxPrefix := 0
	maxPrefixIdx := -1

	// Check if we already have required prefix,
	for idx, prefix := range prefixes {
		_, netip, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}
		parentLen, _ := netip.Mask.Size()
		// Check if some of requests are fit into this prefix.
		if prefixLen == uint32(parentLen) {
			// We found required one.
			resultPrefix := prefixes[idx]
			resultPrefixes := append(prefixes[:idx], prefixes[idx+1:]...)
			// Lets remove from list and return
			return resultPrefix, resultPrefixes, nil
		} else if uint32(parentLen) < prefixLen && (parentLen > maxPrefix || maxPrefix == 0) {
			// Update minimal root for split
			maxPrefix = parentLen
			maxPrefixIdx = idx
		}
	}
	// Not found, lets split minimal found prefix
	if maxPrefixIdx == -1 {
		// There is no room to split
		return "", prefixes, errors.Errorf("Failed to find room to have prefix len %d at %v", prefixLen, prefixes)
	}

	resultPrefixRoot := prefixes[maxPrefixIdx]
	rightParts := []string{}

	_, rootCIDRNet, _ := net.ParseCIDR(resultPrefixRoot)
	for {
		rootLen, _ := rootCIDRNet.Mask.Size()
		if uint32(rootLen) == prefixLen {
			// we found required prefix
			break
		}
		sub1, err := subnet(rootCIDRNet, 0)
		if err != nil {
			return "", prefixes, err
		}

		sub2, err := subnet(rootCIDRNet, 1)
		if err != nil {
			return "", prefixes, err
		}
		rightParts = append(rightParts, sub2.String())
		rootCIDRNet = sub1
	}
	var resultPrefixes []string = nil
	resultPrefixes = append(resultPrefixes, prefixes[:maxPrefixIdx]...)
	resultPrefixes = append(resultPrefixes, reverse(rightParts)...)
	resultPrefixes = append(resultPrefixes, prefixes[maxPrefixIdx+1:]...)

	// return result
	return rootCIDRNet.String(), resultPrefixes, nil
}

func reverse(values []string) []string {
	newValues := make([]string, len(values))

	for i, j := 0, len(values)-1; i <= j; i, j = i+1, j-1 {
		newValues[i], newValues[j] = values[j], values[i]
	}
	return newValues
}

func releasePrefixes(prefixes []string, released ...string) (remaining []string, err error) {
	if len(released) == 0 {
		return prefixes, nil
	}

	allPrefixes := append(prefixes, released...)
	subnets, err := getSubnets(allPrefixes)
	if err != nil {
		return nil, err
	}

	result := removeNestedNetworks(allPrefixes, subnets)
	prefixByPrefixLen := map[int][]*net.IPNet{}

	for _, prefix := range result {
		ipnet := subnets[prefix]
		parentLen, _ := ipnet.Mask.Size()
		nets := prefixByPrefixLen[parentLen]
		nets = append(nets, ipnet)
		prefixByPrefixLen[parentLen] = nets
	}

	for {
		newPrefixByPrefixLen := map[int][]*net.IPNet{}
		changes := 0
		for basePrefix, value := range prefixByPrefixLen {
			base := map[string]*net.IPNet{}

			cvalue := append(newPrefixByPrefixLen[basePrefix], value...)
			parentPrefix := []*net.IPNet{}
			if len(cvalue) < 2 {
				newPrefixByPrefixLen[basePrefix] = cvalue
				continue
			}
			for _, netValue := range cvalue {
				baseIP := &net.IPNet{
					IP:   clearNetIndexInIP(netValue.IP, basePrefix),
					Mask: netValue.Mask,
				}
				parentLen, addrLen := baseIP.Mask.Size()
				bv := base[baseIP.String()]
				if bv == nil {
					base[baseIP.String()] = netValue
				} else {
					// We found item with same base IP, we we could join this two networks into one.
					// Remove from current level
					delete(base, baseIP.String())
					// And put to next level
					parentPrefix = append(parentPrefix, &net.IPNet{
						IP:   baseIP.IP,
						Mask: net.CIDRMask(parentLen-1, addrLen),
					})
					changes++
				}
			}
			leftPrefixes := []*net.IPNet{}
			// Put all not merged values
			for _, value := range base {
				leftPrefixes = append(leftPrefixes, value)
			}
			newPrefixByPrefixLen[basePrefix] = leftPrefixes
			newPrefixByPrefixLen[basePrefix-1] = append(newPrefixByPrefixLen[basePrefix-1], parentPrefix...)
		}
		if changes == 0 {
			// All is merged, we could exit
			result = []string{}
			for _, value := range newPrefixByPrefixLen {
				for _, v := range value {
					result = append(result, v.String())
				}
			}
			return result, nil
		}
		prefixByPrefixLen = newPrefixByPrefixLen
	}
}

func getSubnets(prefixes []string) (map[string]*net.IPNet, error) {
	subnets := make(map[string]*net.IPNet)
	for _, prefix := range prefixes {
		_, subnet, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, errors.Wrapf(err, "Wrong CIDR: %v", prefix)
		}
		subnets[prefix] = subnet
	}

	return subnets, nil
}

func removeNestedNetworks(prefixes []string, subnets map[string]*net.IPNet) []string {
	newPrefixes := make(map[string]struct{})

	for newPrefixIndex, newPrefix := range prefixes {
		intersected := false
		for prefixIndex, prefix := range prefixes {
			if prefixIndex == newPrefixIndex {
				continue
			}
			if intersect, firstIsWider := intersect(subnets[prefix], subnets[newPrefix]); intersect {
				intersected = true
				if !firstIsWider {
					delete(newPrefixes, prefix)
					newPrefixes[newPrefix] = struct{}{}
				}
			}
		}

		if !intersected {
			newPrefixes[newPrefix] = struct{}{}
		}
	}

	prefixesList := make([]string, 0, len(newPrefixes))
	for key := range newPrefixes {
		prefixesList = append(prefixesList, key)
	}

	return prefixesList
}

func subnet(ipnet *net.IPNet, subnetIndex int) (retNet *net.IPNet, reterr error) {
	mask := ipnet.Mask

	parentLen, addrLen := mask.Size()
	newPrefixLen := parentLen + 1
	if newPrefixLen > addrLen {
		return nil, errors.Errorf("insufficient address space to extend prefix of %d", parentLen)
	}

	if uint64(subnetIndex) > 2 {
		return nil, errors.Errorf("prefix extension does not accommodate a subnet numbered %d", subnetIndex)
	}

	return &net.IPNet{
		IP:   setNetIndexInIP(ipnet.IP, subnetIndex, newPrefixLen),
		Mask: net.CIDRMask(newPrefixLen, addrLen),
	}, nil
}

func setNetIndexInIP(ip net.IP, num, prefixLen int) net.IP {
	ipInt, totalBits := fromIP(ip)
	bigNum := big.NewInt(int64(num))
	bigNum.Lsh(bigNum, uint(totalBits-prefixLen))
	ipInt.Or(ipInt, bigNum)
	return toIP(ipInt, totalBits)
}

func clearNetIndexInIP(ip net.IP, prefixLen int) net.IP {
	ipInt, totalBits := fromIP(ip)
	ipInt.SetBit(ipInt, totalBits-prefixLen, 0)
	return toIP(ipInt, totalBits)
}

func toIP(ipInt *big.Int, bits int) net.IP {
	ipBytes := ipInt.Bytes()
	ret := make([]byte, bits/8)
	// Pack our IP bytes into the end of the return array,
	// since big.Int.Bytes() removes front zero padding.
	for i := 1; i <= len(ipBytes); i++ {
		ret[len(ret)-i] = ipBytes[len(ipBytes)-i]
	}
	return net.IP(ret)
}

func fromIP(ip net.IP) (ipVal *big.Int, ipLen int) {
	val := &big.Int{}
	val.SetBytes([]byte(ip))
	i := len(ip)
	if i == net.IPv4len {
		return val, 32
	} // else if i == net.IPv6len
	return val, 128
}

func incrementIP(sourceIP net.IP, ipNet *net.IPNet) (incrIP net.IP, retErr error) {
	ip := make([]byte, len(sourceIP))
	copy(ip, sourceIP)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	if !ipNet.Contains(ip) {
		return nil, errors.Errorf("Overflowed CIDR while incrementing IP")
	}
	return ip, nil
}
