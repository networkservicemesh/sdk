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

package injectspiffeid

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net/url"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func generateCert(spiffeIDPath string) []byte {
	u := &url.URL{Scheme: "spiffe", Host: "test.com", Path: spiffeIDPath}

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
