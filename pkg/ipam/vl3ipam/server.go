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

	"github.com/google/uuid"

	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// ErrUndefined means that operation is not supported.
var ErrUndefined = errors.New("request type is undefined")

// ErrOutOfRange means that ip pool of IPAM is empty.
var ErrOutOfRange = errors.New("prefix is out of range or already in use")

type vl3IPAMServer struct {
	pool             *ippool.IPPool
	excludedPrefixes []string
	poolMutex        sync.Mutex
	initalSize       uint8
}

// NewIPAMServer creates a new ipam.IPAMServer handler for grpc.Server.
func NewIPAMServer(prefix string, initialNSEPrefixSize uint8) ipam.IPAMServer {
	return &vl3IPAMServer{
		pool:       ippool.NewWithNetString(prefix),
		initalSize: initialNSEPrefixSize,
	}
}

var _ ipam.IPAMServer = (*vl3IPAMServer)(nil)

func (s *vl3IPAMServer) ManagePrefixes(prefixServer ipam.IPAM_ManagePrefixesServer) error {
	var clientsPrefixes []string
	var err error

	logger := log.Default().WithField("ID", uuid.New().String())
	for err == nil {
		var r *ipam.PrefixRequest

		r, err = prefixServer.Recv()
		if err != nil {
			break
		}

		logger.Debugf("PrefixRequest: %v", r.String())
		switch r.GetType() {
		case ipam.Type_UNDEFINED:
			return ErrUndefined

		case ipam.Type_ALLOCATE:
			var resp *ipam.PrefixResponse
			resp, err = s.allocate(r)
			if err != nil {
				break
			}
			clientsPrefixes = append(clientsPrefixes, resp.GetPrefix())
			err = prefixServer.Send(resp)
			logger.Debugf("Allocated: %v", resp.String())

		case ipam.Type_DELETE:
			for i, p := range clientsPrefixes {
				if p != r.GetPrefix() {
					continue
				}
				s.delete(r)
				clientsPrefixes = append(clientsPrefixes[:i], clientsPrefixes[i+1:]...)
				logger.Debugf("Deleted: %v", r.GetPrefix())
				break
			}
		}
	}

	s.poolMutex.Lock()
	for _, prefix := range clientsPrefixes {
		s.pool.AddNetString(prefix)
	}
	s.poolMutex.Unlock()
	logger.Debugf("Disconnected. Error: %v", err.Error())
	logger.Debugf("Deleted: %v", clientsPrefixes)

	if prefixServer.Context().Err() != nil {
		return nil
	}

	return errors.Wrap(err, "failed to manage prefixes")
}

func (s *vl3IPAMServer) allocate(r *ipam.PrefixRequest) (*ipam.PrefixResponse, error) {
	s.poolMutex.Lock()
	// We don't need to exclude prefixes which were indicated in the PrefixRequest from the main pool
	pool := s.pool.Clone()
	for _, excludePrefix := range r.GetExcludePrefixes() {
		pool.ExcludeString(excludePrefix)
	}
	resp := &ipam.PrefixResponse{
		Prefix: r.GetPrefix(),
	}
	if resp.GetPrefix() == "" || !s.pool.ContainsNetString(resp.GetPrefix()) {
		ip, err := pool.Pull()
		if err != nil {
			s.poolMutex.Unlock()
			return nil, err
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
	s.pool.ExcludeString(resp.GetPrefix())
	s.poolMutex.Unlock()
	resp.ExcludePrefixes = r.GetExcludePrefixes()
	resp.ExcludePrefixes = append(resp.ExcludePrefixes, s.excludedPrefixes...)
	return resp, nil
}

func (s *vl3IPAMServer) delete(r *ipam.PrefixRequest) {
	s.poolMutex.Lock()
	s.pool.AddNetString(r.GetPrefix())
	s.poolMutex.Unlock()
}
