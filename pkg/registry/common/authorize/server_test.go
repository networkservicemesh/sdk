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
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"go.uber.org/goleak"
)

const (
	policy = `
	package test

default refresh = false

refresh {
	not input.SpiffieIDNSEMap[input.NSEName]
}
refresh {
	input.SpiffieIDNSEMap[input.NSEName] == input.SpiffieID
}
`
)

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func generateCert(host string, expirationTime time.Duration) (string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		return "", errors.Errorf("Failed to generate private key: %v", err)
	}

	keyUsage := x509.KeyUsageDigitalSignature

	notBefore := time.Now()
	notAfter := notBefore.Add(expirationTime)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", errors.Errorf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"NSM"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	u, err := url.Parse(host)
	if err != nil {
		return "", errors.Errorf("Failed to parse host: %v", err)
	}

	template.URIs = append(template.URIs, u)

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return "", errors.Errorf("Failed to create certificate: %v", err)
	}

	sb := new(strings.Builder)

	if err := pem.Encode(sb, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", errors.Errorf("Failed to write data to cert.pem: %v", err)
	}

	return sb.String(), nil
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

func withPeer(ctx context.Context, certPem string) (context.Context, error) {
	block, _ := pem.Decode([]byte(certPem))
	x509cert, err := x509.ParseCertificate(block.Bytes)
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

func TestAuthzEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := authorize.NewNetworkServiceEndpointRegistryServer(authorize.WithPolicies(opa.WithPolicyFromSource(policy, "refresh", opa.True)))

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "nse-1",
		NetworkServiceNames: []string{"ns-1"},
	}

	cert1Pem, err := generateCert("spiffe://test.com/workload1", time.Hour)
	cert2Pem, err := generateCert("spiffe://test.com/workload2", time.Hour)
	cert1Ctx, _ := withPeer(context.Background(), cert1Pem)
	cert2Ctx, _ := withPeer(context.Background(), cert2Pem)

	_, err = server.Register(cert1Ctx, nseReg)
	require.NoError(t, err)

	_, err = server.Register(cert2Ctx, nseReg)
	require.Error(t, err)
}
