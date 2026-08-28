// Copyright (c) 2026 Nordix Foundation.
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

//go:build linux
// +build linux

package recvfd_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/peer"

	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const forwarderServiceName = "forwarder"

// fileTransceiver hands out a distinct real file per inode url, standing in for
// the endpoint's unix socket. A new inode url means the endpoint recreated its
// socket, i.e. it restarted.
type fileTransceiver struct {
	net.Addr
	dir string
}

func (f *fileTransceiver) RecvFileByURL(inodeURL string) (<-chan *os.File, error) {
	name := filepath.Join(f.dir, strings.NewReplacer("/", "_", ":", "_").Replace(inodeURL))
	if _, err := os.Stat(name); os.IsNotExist(err) {
		if werr := os.WriteFile(name, []byte("x"), 0o600); werr != nil {
			return nil, werr
		}
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	ch := make(chan *os.File, 1)
	ch <- file
	return ch, nil
}

func (f *fileTransceiver) RecvFD(dev, inode uint64) <-chan uintptr { return nil }

func (f *fileTransceiver) RecvFile(dev, ino uint64) <-chan *os.File { return nil }

func (f *fileTransceiver) RecvFDByURL(urlStr string) (<-chan uintptr, error) { return nil, nil }

func (f *fileTransceiver) SendFD(fd uintptr) <-chan error { return nil }

func (f *fileTransceiver) SendFile(file grpcfd.SyscallConn) <-chan error { return nil }

func (f *fileTransceiver) SendFilename(filename string) <-chan error { return nil }

// failingNSEServer models an unavailable upstream registry.
type failingNSEServer struct{}

func (s *failingNSEServer) Register(_ context.Context, _ *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return nil, errors.New("registry is unavailable")
}

func (s *failingNSEServer) Find(_ *registry.NetworkServiceEndpointQuery, _ registry.NetworkServiceEndpointRegistry_FindServer) error {
	return nil
}

func (s *failingNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

// countOpenFilesIn returns the number of fds of the current process pointing
// into dir, i.e. the number of endpoint sockets currently held.
func countOpenFilesIn(t *testing.T, dir string) int {
	entries, err := os.ReadDir("/proc/self/fd")
	require.NoError(t, err)

	var count int
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			// fd was closed while we were walking the directory
			continue
		}
		if strings.HasPrefix(target, dir) {
			count++
		}
	}
	return count
}

// TestNseRecvfdServerReleasesStaleForwarderFiles - a forwarder that re-registers
// with a new socket while the upstream registry is unavailable must not make the
// server accumulate a file per attempt. Only the socket the forwarder is
// listening on now has to be kept, so that a transient registry outage does not
// tear down the data path.
func TestNseRecvfdServerReleasesStaleForwarderFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &fileTransceiver{dir: dir}})

	server := chain.NewNetworkServiceEndpointRegistryServer(
		registryrecvfd.NewNetworkServiceEndpointRegistryServer(
			registryrecvfd.WithForwarderServiceName(forwarderServiceName),
		),
		new(failingNSEServer),
	)

	const restarts = 20
	for i := 0; i < restarts; i++ {
		// same endpoint name, new socket each time: a restarted forwarder
		_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                "forwarder-abcde",
			NetworkServiceNames: []string{forwarderServiceName},
			Url:                 fmt.Sprintf("inode://1/%d", 1000+i),
		})
		require.Error(t, err)
	}

	open := countOpenFilesIn(t, dir)
	require.Equal(t, 1, open,
		"expected only the current forwarder socket to be held, found %d fds after %d forwarder restarts", open, restarts)
}
