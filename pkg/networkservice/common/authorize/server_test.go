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
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
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

func testPolicy() authorize.Policy {
	return opa.WithPolicyFromSource(`
		package test
	
		default allow = false
	
		allow {
			 input.path_segments[_].token = "allowed"
		}
`, "allow", opa.True)
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
	suits := []struct {
		name     string
		policy   authorize.Policy
		request  *networkservice.NetworkServiceRequest
		response *networkservice.Connection
		denied   bool
	}{
		{
			name:    "simple positive test",
			policy:  testPolicy(),
			request: requestWithToken("allowed"),
			denied:  false,
		},
		{
			name:    "simple negative test",
			policy:  testPolicy(),
			request: requestWithToken("not_allowed"),
			denied:  true,
		},
	}

	for i := range suits {
		s := suits[i]
		t.Run(s.name, func(t *testing.T) {
			srv := authorize.NewServer(authorize.WithPolicies(s.policy))
			checkResult := func(err error) {
				if !s.denied {
					require.Nil(t, err, "request expected to be not denied: ")
					return
				}
				require.NotNil(t, err, "request expected to be denied")
				s, ok := status.FromError(err)
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
