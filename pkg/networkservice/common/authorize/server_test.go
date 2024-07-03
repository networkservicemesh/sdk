// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package authorize_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	mathrand "math/rand"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

func generateCert(u *url.URL) []byte {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		URIs:         []*url.URL{u},
	}

	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pub := &priv.PublicKey

	certBytes, _ := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	return certBytes
}

func withPeer(ctx context.Context, certBytes []byte) (context.Context, error) {
	x509cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	authInfo := &credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{x509cert},
		},
	}
	return peer.NewContext(ctx, &peer.Peer{AuthInfo: authInfo}), nil
}

func testPolicy() string {
	return `
package test
	
default valid = false

valid {
		input.path_segments[_].token = "allowed"
}
`
}

func requestWithToken(token string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Path: &networkservice.Path{
				Index: 0,
				PathSegments: []*networkservice.PathSegment{
					{
						Token: token,
					},
				},
			},
		},
	}
}

func TestAuthorize_ShouldCorrectlyWorkWithHeal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Path: &networkservice.Path{
				PathSegments: []*networkservice.PathSegment{
					{},
				},
			},
		},
	}

	// simulate heal request
	conn, err := authorize.NewServer().Request(context.Background(), r)
	require.NoError(t, err)

	// simulate timeout close
	_, err = authorize.NewServer().Close(context.Background(), conn)
	require.NoError(t, err)
}

func TestAuthzEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir := filepath.Clean(path.Join(os.TempDir(), t.Name()))
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	err := os.MkdirAll(dir, os.ModePerm)
	require.Nil(t, err)

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = os.WriteFile(policyPath, []byte(testPolicy()), os.ModePerm)
	require.Nil(t, err)

	suits := []struct {
		name       string
		policyPath string
		request    *networkservice.NetworkServiceRequest
		response   *networkservice.Connection
		denied     bool
	}{
		{
			name:       "simple positive test",
			policyPath: policyPath,
			request:    requestWithToken("allowed"),
			denied:     false,
		},
		{
			name:       "simple negative test",
			policyPath: policyPath,
			request:    requestWithToken("not_allowed"),
			denied:     true,
		},
	}

	for i := range suits {
		s := suits[i]
		t.Run(s.name, func(t *testing.T) {
			srv := authorize.NewServer(authorize.WithPolicies(s.policyPath))
			checkResult := func(err error) {
				if !s.denied {
					require.Nil(t, err, "request expected to be not denied: ")
					return
				}
				require.NotNil(t, err, "request expected to be denied")
				s, ok := status.FromError(errors.Cause(err))
				require.True(t, ok, "error without error status code"+err.Error())
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			ctx := peer.NewContext(context.Background(), &peer.Peer{})

			_, err := srv.Request(ctx, s.request)
			checkResult(err)

			_, err = srv.Close(ctx, s.request.GetConnection())
			checkResult(err)
		})
	}
}

func TestAuthorize_EmptySpiffeIDConnectionMapOnClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	conn := &networkservice.Connection{
		Id: "conn",
		Path: &networkservice.Path{
			Index: 1,
			PathSegments: []*networkservice.PathSegment{
				{Id: "id-1"},
				{Id: "id-2"},
			},
		},
	}

	server := authorize.NewServer(authorize.Any())
	certBytes := generateCert(&url.URL{Scheme: "spiffe", Host: "test.com", Path: "test"})

	ctx, err := withPeer(context.Background(), certBytes)
	require.NoError(t, err)

	_, err = server.Close(ctx, conn)
	require.NoError(t, err)
}

type randomErrorServer struct {
	errorChance float32
}

// NewServer returns a server chain element returning error on Close/Request on given times
func NewServer(errorChance float32) networkservice.NetworkServiceServer {
	return &randomErrorServer{errorChance: errorChance}
}

func (s randomErrorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// nolint
	val := mathrand.Float32()
	if val > s.errorChance {
		return nil, errors.New("random error")
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s randomErrorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type closeData struct {
	conn *networkservice.Connection
	cert []byte
}

func TestAuthorize_SpiffeIDConnectionMapHaveNoLeaks(t *testing.T) {
	dir := filepath.Clean(path.Join(os.TempDir(), t.Name()))
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	err := os.MkdirAll(dir, os.ModePerm)
	require.Nil(t, err)

	policyPath := filepath.Clean(path.Join(dir, "policy.rego"))
	err = os.WriteFile(policyPath, []byte(testPolicy()), os.ModePerm)
	require.Nil(t, err)

	var authorizeMap genericsync.Map[spiffeid.ID, *genericsync.Map[string, struct{}]]
	chain := next.NewNetworkServiceServer(
		authorize.NewServer(authorize.WithPolicies(policyPath), authorize.WithSpiffeIDConnectionMap(&authorizeMap)),
		NewServer(0.2),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
			Path: &networkservice.Path{
				Index: 2,
				PathSegments: []*networkservice.PathSegment{
					{Id: "client", Name: "client", Token: "allowed"},
					{Id: "nsmgr", Name: "nsmgr", Token: "allowed"},
					{Id: "forwarder", Name: "forwarder", Token: "allowed"},
				},
			},
		},
	}

	// Make 1000 requests with random spiffe IDs
	count := 1000
	data := make([]closeData, 0)
	for i := 0; i < count; i++ {
		spiffeidPath, err := nanoid.GenerateString(10, nanoid.WithAlphabet("abcdefghijklmnopqrstuvwxyz"))
		require.NoError(t, err)

		certBytes := generateCert(&url.URL{Scheme: "spiffe", Host: "test.com", Path: spiffeidPath})
		ctx, err := withPeer(context.Background(), certBytes)
		require.NoError(t, err)

		conn, err := chain.Request(ctx, request)
		if err == nil {
			data = append(data, closeData{conn: conn, cert: certBytes})
		}
	}

	// Close the connections established in the previous loop
	for _, closeData := range data {
		ctx, err := withPeer(context.Background(), closeData.cert)
		require.NoError(t, err)
		_, err = chain.Close(ctx, closeData.conn)
		require.NoError(t, err)
	}

	mapLen := 0
	authorizeMap.Range(func(key spiffeid.ID, value *genericsync.Map[string, struct{}]) bool {
		mapLen++
		return true
	})
	require.Equal(t, mapLen, 0)
}
