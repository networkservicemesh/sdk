// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

func serve(ctx context.Context, t *testing.T, u *url.URL, register func(server *grpc.Server)) {
	server := grpc.NewServer(append([]grpc.ServerOption{
		grpc.Creds(grpcfdTransportCredentials(insecure.NewCredentials())),
	}, tracing.WithTracing()...)...)
	register(server)

	errCh := grpcutils.ListenAndServe(ctx, u, server)
	uString := u.String()

	go func() {
		select {
		case <-ctx.Done():
			log.FromContext(ctx).Infof("Stop serve: %s", uString)
			return
		case err := <-errCh:
			require.NoError(t, err)
		}
	}()
}

// CheckURLFree returns is given url is free for Listen.
func CheckURLFree(u *url.URL) bool {
	ln, err := net.Listen(grpcutils.TargetToNetAddr(grpcutils.URLToTarget(u)))
	if err == nil {
		err = ln.Close()
	}
	return err == nil
}

// CloneURL clones given url.
func CloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}

	cloned := new(url.URL)
	*cloned = *u

	return cloned
}
