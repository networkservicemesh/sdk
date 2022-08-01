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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"go.uber.org/goleak"
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

func TestAuthzEndpointRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := authorize.NewNetworkServiceEndpointRegistryServer(
		authorize.WithRegisterPolicies(opa.WithNSERegisterValidPolicy()),
		authorize.WithUnregisterPolicies(opa.WithNSEUnregisterValidPolicy()),
	)

	nseReg := &registry.NetworkServiceEndpoint{Name: "nse-1"}

	u1, _ := url.Parse("spiffe://test.com/workload1")
	u2, _ := url.Parse("spiffe://test.com/workload2")
	cert1 := generateCert(u1)
	cert2 := generateCert(u2)
	cert1Ctx, err := withPeer(context.Background(), cert1)
	require.NoError(t, err)
	cert2Ctx, err := withPeer(context.Background(), cert2)
	require.NoError(t, err)

	_, err = server.Register(cert1Ctx, nseReg)
	require.NoError(t, err)

	_, err = server.Register(cert2Ctx, nseReg)
	require.Error(t, err)

	_, err = server.Register(cert1Ctx, nseReg)
	require.NoError(t, err)

	_, err = server.Unregister(cert2Ctx, nseReg)
	require.Error(t, err)

	_, err = server.Unregister(cert1Ctx, nseReg)
	require.NoError(t, err)
}
