// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

// Package vl3ipam provides implementation of api/pkg/api/ipam.IPAMServer for vL3 scenario.
package vl3ipam

import (
	"net"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

// ErrUndefined means that operation is not supported
var ErrUndefined = errors.New("request type is undefined")

// ErrOutOfRange means that ip pool of IPAM is empty
var ErrOutOfRange = errors.New("prefix is out of range or already in use")

type vl3IPAMServer struct {
	pool             *ippool.IPPool
	excludedPrefixes []string
	poolMutex        sync.Mutex
	initalSize       uint8
}

// NewIPAMServer creates a new ipam.IPAMServer handler for grpc.Server
func NewIPAMServer(prefix string, initialNSEPrefixSize uint8) ipam.IPAMServer {
	return &vl3IPAMServer{
		pool:       ippool.NewWithNetString(prefix),
		initalSize: initialNSEPrefixSize,
	}
}

var _ ipam.IPAMServer = (*vl3IPAMServer)(nil)

func (s *vl3IPAMServer) ManagePrefixes(prefixServer ipam.IPAM_ManagePrefixesServer) error {
	var pool = s.pool
	var mutex = &s.poolMutex
	var clientsPrefixes []string
	var err error

	for err == nil {
		var r *ipam.PrefixRequest

		r, err = prefixServer.Recv()
		if err != nil {
			break
		}

		switch r.Type {
		case ipam.Type_UNDEFINED:
			return ErrUndefined

		case ipam.Type_ALLOCATE:
			var resp ipam.PrefixResponse
			mutex.Lock()
			for _, excludePrefix := range r.ExcludePrefixes {
				pool.ExcludeString(excludePrefix)
			}
			resp.Prefix = r.Prefix
			if resp.Prefix == "" || !pool.ContainsNetString(resp.Prefix) {
				var ip net.IP
				ip, err = pool.Pull()
				if err != nil {
					mutex.Unlock()
					break
				}
				ipNet := &net.IPNet{
					IP: ip,
					Mask: net.CIDRMask(
						int(s.initalSize),
						len(ip)*8,
					),
				}
				resp.Prefix = ipNet.String()
			}
			s.excludedPrefixes = append(s.excludedPrefixes, r.Prefix)
			clientsPrefixes = append(clientsPrefixes, resp.Prefix)
			pool.ExcludeString(resp.Prefix)
			mutex.Unlock()
			resp.ExcludePrefixes = r.ExcludePrefixes
			resp.ExcludePrefixes = append(resp.ExcludePrefixes, s.excludedPrefixes...)
			err = prefixServer.Send(&resp)

		case ipam.Type_DELETE:
			for i, p := range clientsPrefixes {
				if p != r.Prefix {
					continue
				}
				mutex.Lock()
				pool.AddNetString(p)
				mutex.Unlock()
				clientsPrefixes = append(clientsPrefixes[:i], clientsPrefixes[i+1:]...)
				break
			}
		}
	}

	s.poolMutex.Lock()
	for _, prefix := range clientsPrefixes {
		pool.AddNetString(prefix)
	}
	s.poolMutex.Unlock()

	if prefixServer.Context().Err() != nil {
		return nil
	}

	return errors.WithStack(err)
}
