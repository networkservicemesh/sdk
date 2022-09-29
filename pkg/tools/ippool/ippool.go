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

// Package ippool provides service for managing ip addresses
package ippool

import (
	"errors"
	"math"
	"net"
	"sync"
)

type color bool

const (
	black, red     color = true, false
	prefixBitsSize       = 64
)

// IPPool holds available ip addresses in the structure of red-black tree
type IPPool struct {
	root     *treeNode
	lock     sync.Mutex
	size     uint64
	ipLength int
}

// treeNode is a single element within the IP pool tree
type treeNode struct {
	Value  *ipRange
	color  color
	Left   *treeNode
	Right  *treeNode
	Parent *treeNode
}

type iterator struct {
	node *treeNode
}

// New instantiates a ip pool as red-black tree with the specified ip length.
func New(ipLength int) *IPPool {
	return &IPPool{
		ipLength: ipLength,
	}
}

// NewWithNet instantiates a ip pool as red-black tree with the specified ip network
func NewWithNet(ipNet *net.IPNet) *IPPool {
	ipPool := &IPPool{
		ipLength: len(ipNet.IP),
	}
	ipPool.AddNet(ipNet)
	return ipPool
}

// NewWithNetString instantiates a ip pool as red-black tree with the specified ip network
func NewWithNetString(ipNetString string) *IPPool {
	_, ipNet, err := net.ParseCIDR(ipNetString)
	if err != nil {
		return nil
	}

	return NewWithNet(ipNet)
}

// Clone - make a clone of the pool
func (tree *IPPool) Clone() *IPPool {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	return tree.clone()
}

func (tree *IPPool) clone() *IPPool {
	newPool := &IPPool{
		root:     nil,
		size:     tree.size,
		ipLength: tree.ipLength,
	}

	if tree.root == nil {
		return newPool
	}

	newPool.root = tree.root.clone()

	return newPool
}

// Add - adds ip address to the pool
func (tree *IPPool) Add(ip net.IP) {
	if ip == nil || tree.ipLength != len(ip) {
		return
	}

	tree.lock.Lock()
	defer tree.lock.Unlock()

	tree.add(ipAddressFromIP(ip))
}

// AddString - adds ip address to the pool by string value
func (tree *IPPool) AddString(in string) {
	ip := net.ParseIP(in)
	if tree.ipLength == net.IPv4len {
		ip = ip.To4()
	}
	tree.Add(ip)
}

// AddNet - adds ip addresses from network to the pool
func (tree *IPPool) AddNet(ipNet *net.IPNet) {
	if ipNet == nil || tree.ipLength != len(ipNet.IP) {
		return
	}

	tree.lock.Lock()
	defer tree.lock.Unlock()

	tree.addRange(ipRangeFromIPNet(ipNet))
}

// AddNetString - adds ip addresses from network to the pool by string value
func (tree *IPPool) AddNetString(ipNetString string) {
	_, ipNet, err := net.ParseCIDR(ipNetString)
	if err != nil {
		return
	}

	tree.AddNet(ipNet)
}

// ContainsNetString parses ipNetRaw string and checks that pool contains whole ipNet
func (tree *IPPool) ContainsNetString(ipNetRaw string) bool {
	_, ipNet, err := net.ParseCIDR(ipNetRaw)
	if err != nil {
		return false
	}

	return tree.ContainsNet(ipNet)
}

// ContainsNet checks that pool contains whole ipNet
func (tree *IPPool) ContainsNet(ipNet *net.IPNet) bool {
	if ipNet == nil {
		return false
	}

	var node = tree.root
	var ipRange = ipRangeFromIPNet(ipNet)

	for node != nil {
		compare := node.Value.CompareRange(ipRange)
		switch {
		case compare < 0:
			node = node.Left
		case compare > 0:
			node = node.Right
		default:
			lRange, rRange := ipRange.Sub(node.Value)
			return lRange == nil && rRange == nil
		}
	}

	return false
}

// Contains - check the pool contains ip address
func (tree *IPPool) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}

	return tree.lookup(ipAddressFromIP(ip)) != nil
}

// ContainsString - check the pool contains ip by string value
func (tree *IPPool) ContainsString(in string) bool {
	return tree.Contains(net.ParseIP(in))
}

// Exclude - exclude network from pool
func (tree *IPPool) Exclude(ipNet *net.IPNet) {
	if ipNet == nil {
		return
	}

	tree.lock.Lock()
	defer tree.lock.Unlock()

	tree.deleteRange(ipRangeFromIPNet(ipNet))
}

// ExcludeString - exclude network from pool by string value
func (tree *IPPool) ExcludeString(ipNetString string) {
	_, ipNet, err := net.ParseCIDR(ipNetString)
	if err != nil {
		return
	}

	tree.Exclude(ipNet)
}

// Pull - returns next IP address from pool
func (tree *IPPool) Pull() (net.IP, error) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	ip := tree.pull()
	if ip == nil {
		return nil, errors.New("IPPool is empty")
	}
	return ipFromIPAddress(ip, tree.ipLength), nil
}

// PullIPString - returns requested IP address from the pool by string
func (tree *IPPool) PullIPString(ipString string, exclude ...*IPPool) (*net.IPNet, error) {
	ip, _, err := net.ParseCIDR(ipString)
	if err != nil {
		return nil, err
	}

	return tree.PullIP(ip, exclude...)
}

// PullIP - returns requested IP address from the pool
func (tree *IPPool) PullIP(ip net.IP, exclude ...*IPPool) (*net.IPNet, error) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	clone := tree.clone()
	for _, pool := range exclude {
		clone.excludePool(pool)
	}

	if clone.Contains(ip) {
		tree.deleteRange(&ipRange{
			start: ipAddressFromIP(ip),
			end:   ipAddressFromIP(ip),
		})
	} else {
		return nil, errors.New("IPPool doesn't contain required IP")
	}

	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(tree.ipLength*8, tree.ipLength*8),
	}, nil
}

// PullP2PAddrs - returns next IP addresses pair from pool for peer-to-peer connection
func (tree *IPPool) PullP2PAddrs(exclude ...*IPPool) (srcNet, dstNet *net.IPNet, err error) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	clone := tree.clone()

	for _, pool := range exclude {
		clone.excludePool(pool)
	}

	srcIP := clone.pull()
	if srcIP == nil {
		return nil, nil, errors.New("IPPool is empty")
	}

	dstIP := clone.pull()
	if dstIP == nil {
		return nil, nil, errors.New("IPPool is empty")
	}

	tree.deleteRange(&ipRange{
		start: srcIP.Clone(),
		end:   srcIP.Clone(),
	})
	tree.deleteRange(&ipRange{
		start: dstIP.Clone(),
		end:   dstIP.Clone(),
	})

	srcNet = &net.IPNet{
		IP:   ipFromIPAddress(srcIP, tree.ipLength),
		Mask: net.CIDRMask(tree.ipLength*8, tree.ipLength*8),
	}

	dstNet = &net.IPNet{
		IP:   ipFromIPAddress(dstIP, tree.ipLength),
		Mask: net.CIDRMask(tree.ipLength*8, tree.ipLength*8),
	}

	return srcNet, dstNet, nil
}

// GetPrefixes returns the list of saved prefixes
func (tree *IPPool) GetPrefixes() []string {
	tree.lock.Lock()
	clone := tree.clone()
	tree.lock.Unlock()

	if clone.root == nil {
		return nil
	}

	it := iterator{
		node: clone.root,
	}
	for it.node.Left != nil {
		it.node = it.node.Left
	}

	var prefixes []string
	for node := it.Next(); node != nil; node = it.Next() {
		prefixes = append(prefixes, node.getPrefixes(clone.ipLength)...)
	}

	return prefixes
}

func (tree *IPPool) excludePool(exclude *IPPool) {
	if exclude == nil {
		return
	}

	tree.excludeNode(exclude.root)
}

func (tree *IPPool) excludeNode(exclude *treeNode) {
	if exclude == nil {
		return
	}

	tree.excludeNode(exclude.Left)
	tree.deleteRange(exclude.Value)
	tree.excludeNode(exclude.Right)
}

// Empty returns true if pool does not contain any nodes
func (tree *IPPool) Empty() bool {
	return tree.root == nil
}

func (tree *IPPool) addRange(ipR *ipRange) {
	var insertedNode *treeNode
	if tree.root == nil {
		tree.root = &treeNode{Value: ipR, color: red}
		insertedNode = tree.root
	} else {
		node := tree.root
		loop := true
		for loop {
			compare := node.Value.CompareRange(ipR)
			switch {
			case compare >= -1 && compare <= 1:
				value := node.Value.Clone()
				tree.removeNode(node)
				tree.addRange(value.Unite(ipR))
				return
			case compare < -1:
				if node.Left == nil {
					node.Left = &treeNode{Value: ipR, color: red}
					insertedNode = node.Left
					loop = false
				} else {
					node = node.Left
				}
			case compare > 1:
				if node.Right == nil {
					node.Right = &treeNode{Value: ipR, color: red}
					insertedNode = node.Right
					loop = false
				} else {
					node = node.Right
				}
			}
		}
		insertedNode.Parent = node
	}
	tree.insertCase1(insertedNode)
	tree.size++
}

// add - inserts IP address into the pool
func (tree *IPPool) add(ip *ipAddress) {
	ipR := &ipRange{
		start: ip.Clone(),
		end:   ip.Clone(),
	}
	tree.addRange(ipR)
}

func (tree *IPPool) pull() *ipAddress {
	node := tree.left()
	if node == nil {
		return nil
	}

	ip := node.Value.start
	if node.Value.start.Equal(node.Value.end) {
		tree.removeNode(node)
		return ip
	}
	node.Value.start = node.Value.start.Next()
	return ip
}

func (tree *IPPool) deleteRange(ipR *ipRange) {
	node := tree.root
	for node != nil {
		compare := node.Value.CompareRange(ipR)
		switch {
		case compare < 0:
			node = node.Left
		case compare > 0:
			node = node.Right
		default:
			lRange, rRange := node.Value.Sub(ipR)
			// Remove Node and try again to find other intersection
			tree.removeNode(node)
			tree.deleteRange(ipR)

			// Add ranges out of ipR
			if lRange != nil {
				tree.addRange(lRange)
			}
			if rRange != nil {
				tree.addRange(rRange)
			}
			return
		}
	}
}

func (tree *IPPool) removeNode(node *treeNode) {
	if node == nil {
		return
	}
	if node.Left != nil && node.Right != nil {
		pred := node.Left.maximumNode()
		node.Value = pred.Value
		node = pred
	}
	var child *treeNode
	if node.Left == nil || node.Right == nil {
		if node.Right == nil {
			child = node.Left
		} else {
			child = node.Right
		}
		if node.color == black {
			node.color = nodeColor(child)
			tree.deleteCase1(node)
		}
		tree.replaceNode(node, child)
		if node.Parent == nil && child != nil {
			child.color = black
		}
	}
	tree.size--
}

func (tree *IPPool) left() *treeNode {
	var parent *treeNode
	current := tree.root
	for current != nil {
		parent = current
		current = current.Left
	}
	return parent
}

func (tree *IPPool) right() *treeNode {
	var parent *treeNode
	current := tree.root
	for current != nil {
		parent = current
		current = current.Right
	}
	return parent
}

// Clear removes all addresses from the IP pool.
func (tree *IPPool) Clear() {
	tree.root = nil
}

func (tree *IPPool) lookup(ip *ipAddress) *treeNode {
	node := tree.root
	for node != nil {
		compare := node.Value.Compare(ip)
		switch {
		case compare == 0:
			return node
		case compare < 0:
			node = node.Left
		case compare > 0:
			node = node.Right
		}
	}
	return nil
}

func (node *treeNode) grandparent() *treeNode {
	if node != nil && node.Parent != nil {
		return node.Parent.Parent
	}
	return nil
}

func (node *treeNode) uncle() *treeNode {
	if node == nil || node.Parent == nil || node.Parent.Parent == nil {
		return nil
	}
	return node.Parent.sibling()
}

func (node *treeNode) sibling() *treeNode {
	if node == nil || node.Parent == nil {
		return nil
	}
	if node == node.Parent.Left {
		return node.Parent.Right
	}
	return node.Parent.Left
}

func (tree *IPPool) rotateLeft(node *treeNode) {
	right := node.Right
	tree.replaceNode(node, right)
	node.Right = right.Left
	if right.Left != nil {
		right.Left.Parent = node
	}
	right.Left = node
	node.Parent = right
}

func (tree *IPPool) rotateRight(node *treeNode) {
	left := node.Left
	tree.replaceNode(node, left)
	node.Left = left.Right
	if left.Right != nil {
		left.Right.Parent = node
	}
	left.Right = node
	node.Parent = left
}

func (tree *IPPool) replaceNode(oldNode, newNode *treeNode) {
	if oldNode.Parent == nil {
		tree.root = newNode
	} else {
		if oldNode == oldNode.Parent.Left {
			oldNode.Parent.Left = newNode
		} else {
			oldNode.Parent.Right = newNode
		}
	}
	if newNode != nil {
		newNode.Parent = oldNode.Parent
	}
}

func (tree *IPPool) insertCase1(node *treeNode) {
	if node.Parent == nil {
		node.color = black
	} else {
		tree.insertCase2(node)
	}
}

func (tree *IPPool) insertCase2(node *treeNode) {
	if nodeColor(node.Parent) == black {
		return
	}
	tree.insertCase3(node)
}

func (tree *IPPool) insertCase3(node *treeNode) {
	uncle := node.uncle()
	if nodeColor(uncle) == red {
		node.Parent.color = black
		uncle.color = black
		node.grandparent().color = red
		tree.insertCase1(node.grandparent())
	} else {
		tree.insertCase4(node)
	}
}

func (tree *IPPool) insertCase4(node *treeNode) {
	grandparent := node.grandparent()
	if node == node.Parent.Right && node.Parent == grandparent.Left {
		tree.rotateLeft(node.Parent)
		node = node.Left
	} else if node == node.Parent.Left && node.Parent == grandparent.Right {
		tree.rotateRight(node.Parent)
		node = node.Right
	}
	tree.insertCase5(node)
}

func (tree *IPPool) insertCase5(node *treeNode) {
	node.Parent.color = black
	grandparent := node.grandparent()
	grandparent.color = red
	if node == node.Parent.Left && node.Parent == grandparent.Left {
		tree.rotateRight(grandparent)
	} else if node == node.Parent.Right && node.Parent == grandparent.Right {
		tree.rotateLeft(grandparent)
	}
}

func (node *treeNode) clone() *treeNode {
	if node == nil {
		return nil
	}
	newNode := &treeNode{
		Value: node.Value.Clone(),
		color: node.color,
	}
	if node.Right != nil {
		newNode.Right = node.Right.clone()
		newNode.Right.Parent = newNode
	}
	if node.Left != nil {
		newNode.Left = node.Left.clone()
		newNode.Left.Parent = newNode
	}
	return newNode
}

func (node *treeNode) maximumNode() *treeNode {
	if node == nil {
		return nil
	}
	for node.Right != nil {
		node = node.Right
	}
	return node
}

func (node *treeNode) getPrefixes(ipLength int) (result []string) {
	start := node.Value.start.Clone()
	end := node.Value.end.Clone()

	// if interval has a few available prefixes on /64 and higher
	if start.high != end.high {
		// get available prefixes from the first /64 network
		if start.low != 0 {
			for start.low != 0 {
				z := trailingZeros(start.low)

				ipNet := &net.IPNet{
					IP:   ipFromIPAddress(start, ipLength),
					Mask: net.CIDRMask(ipLength*8-z, ipLength*8),
				}
				result = append(result, ipNet.String())
				start.low += uint64(1) << z
			}

			start.low = 0
			start.high++
		}

		// exclude the last /64 network if it is not full
		if end.low != math.MaxUint64 {
			end.high--
		}

		// get available prefixes bigger than /64
		for start.high <= end.high {
			z := trailingZeros(start.high)

			if z == prefixBitsSize {
				if end.high == math.MaxUint64 {
					ipNet := &net.IPNet{
						IP:   ipFromIPAddress(start, ipLength),
						Mask: net.CIDRMask(0, ipLength*8),
					}
					result = append(result, ipNet.String())
					return
				}
				z--
			}

			for z > 0 && end.high-start.high+1 < uint64(1)<<z {
				z--
			}

			ipNet := &net.IPNet{
				IP:   ipFromIPAddress(start, ipLength),
				Mask: net.CIDRMask(ipLength*8-z-64, ipLength*8),
			}
			result = append(result, ipNet.String())
			start.high += uint64(1) << z
		}

		if end.low == math.MaxUint64 {
			return result
		}

		end.high++
		start.high = end.high
		start.low = 0
	}

	// get available prefixes from the last /64 network
	for start.low <= end.low {
		z := trailingZeros(start.low)

		if z == prefixBitsSize {
			if end.low == math.MaxUint64 {
				ipNet := &net.IPNet{
					IP:   ipFromIPAddress(start, ipLength),
					Mask: net.CIDRMask(ipLength*8-64, ipLength*8),
				}
				result = append(result, ipNet.String())
				return
			}
			z--
		}

		for z > 0 && end.low-start.low+1 < uint64(1)<<z {
			z--
		}

		ipNet := &net.IPNet{
			IP:   ipFromIPAddress(start, ipLength),
			Mask: net.CIDRMask(ipLength*8-z, ipLength*8),
		}
		result = append(result, ipNet.String())
		start.low += uint64(1) << z
	}

	return result
}

func (tree *IPPool) deleteCase1(node *treeNode) {
	if node.Parent == nil {
		return
	}
	tree.deleteCase2(node)
}

func (tree *IPPool) deleteCase2(node *treeNode) {
	sibling := node.sibling()
	if nodeColor(sibling) == red {
		node.Parent.color = red
		sibling.color = black
		if node == node.Parent.Left {
			tree.rotateLeft(node.Parent)
		} else {
			tree.rotateRight(node.Parent)
		}
	}
	tree.deleteCase3(node)
}

func (tree *IPPool) deleteCase3(node *treeNode) {
	sibling := node.sibling()
	if nodeColor(node.Parent) == black &&
		nodeColor(sibling) == black &&
		nodeColor(sibling.Left) == black &&
		nodeColor(sibling.Right) == black {
		sibling.color = red
		tree.deleteCase1(node.Parent)
	} else {
		tree.deleteCase4(node)
	}
}

func (tree *IPPool) deleteCase4(node *treeNode) {
	sibling := node.sibling()
	if nodeColor(node.Parent) == red &&
		nodeColor(sibling) == black &&
		nodeColor(sibling.Left) == black &&
		nodeColor(sibling.Right) == black {
		sibling.color = red
		node.Parent.color = black
	} else {
		tree.deleteCase5(node)
	}
}

func (tree *IPPool) deleteCase5(node *treeNode) {
	sibling := node.sibling()
	if node == node.Parent.Left &&
		nodeColor(sibling) == black &&
		nodeColor(sibling.Left) == red &&
		nodeColor(sibling.Right) == black {
		sibling.color = red
		sibling.Left.color = black
		tree.rotateRight(sibling)
	} else if node == node.Parent.Right &&
		nodeColor(sibling) == black &&
		nodeColor(sibling.Right) == red &&
		nodeColor(sibling.Left) == black {
		sibling.color = red
		sibling.Right.color = black
		tree.rotateLeft(sibling)
	}
	tree.deleteCase6(node)
}

func (tree *IPPool) deleteCase6(node *treeNode) {
	sibling := node.sibling()
	sibling.color = nodeColor(node.Parent)
	node.Parent.color = black
	if node == node.Parent.Left && nodeColor(sibling.Right) == red {
		sibling.Right.color = black
		tree.rotateLeft(node.Parent)
	} else if nodeColor(sibling.Left) == red {
		sibling.Left.color = black
		tree.rotateRight(node.Parent)
	}
}

func nodeColor(node *treeNode) color {
	if node == nil {
		return black
	}
	return node.color
}

func (it *iterator) Next() *treeNode {
	if it.node == nil {
		return nil
	}

	result := it.node

	if it.node.Right == nil {
		for it.node.Parent != nil && it.node.Parent.Right == it.node {
			it.node = it.node.Parent
		}
		it.node = it.node.Parent
		return result
	}

	it.node = it.node.Right
	for it.node.Left != nil {
		it.node = it.node.Left
	}

	return result
}

func trailingZeros(num uint64) int {
	if num == 0 {
		return prefixBitsSize
	}
	power := 1
	for num != 0 && num%2 == 0 {
		num /= 2
		power++
	}
	return power - 1
}
