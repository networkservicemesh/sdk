// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package opautils_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/opautils"

	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/credentials"
)

const (
	// certPem is a X.509 certificate with spiffeId = "spiffe://test.com/workload"
	certPem = `-----BEGIN CERTIFICATE-----
MIIBvjCCAWWgAwIBAgIQbnFakUhzr52nHoLGltZDyDAKBggqhkjOPQQDAjAdMQsw
CQYDVQQGEwJVUzEOMAwGA1UEChMFU1BJUkUwHhcNMjAwMTAxMDEwMTAxWhcNMzAw
MTAxMDEwMTAxWjAdMQswCQYDVQQGEwJVUzEOMAwGA1UEChMFU1BJUkUwWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAASlFpbASv+NIyVdFwTp22JR5gx7D6LJ01Z8Wz0S
ZiBneWRAcYUBBQY6zKwr/RQtCDxUcFfFyq4zEfUD29a5Phnoo4GGMIGDMA4GA1Ud
DwEB/wQEAwIDqDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0T
AQH/BAIwADAdBgNVHQ4EFgQUJJpYlJa1eNEcks+zJcwKClopSAowJQYDVR0RBB4w
HIYac3BpZmZlOi8vdGVzdC5jb20vd29ya2xvYWQwCgYIKoZIzj0EAwIDRwAwRAIg
Dk6tlURSF8ULhNbnyUxFQ33rDic2dX8jOIstV2dWErwCIDRH2yw0swTcUMQWYgHy
aMp+T747AZGjOEfwHb9/w+7m
-----END CERTIFICATE-----
`
	spiffeID = "spiffe://test.com/workload"
)

func getConnectionWithToken(token string) *networkservice.Connection {
	return &networkservice.Connection{
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{
				{
					Token: token,
				},
			},
		},
	}
}

func TestPreparedOpaInput(t *testing.T) {
	block, _ := pem.Decode([]byte(certPem))
	x509cert, err := x509.ParseCertificate(block.Bytes)
	assert.Nil(t, err)
	authInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				x509cert,
			},
		},
	}
	testToken := "testToken"
	conn := getConnectionWithToken(testToken)

	expectedInput := map[string]interface{}{
		"connection": map[string]interface{}{
			"path": map[string]interface{}{
				"path_segments": []interface{}{
					map[string]interface{}{
						"token": testToken,
					},
				},
			},
		},
		"auth_info": map[string]interface{}{
			"certificate": certPem,
			"spiffe_id":   spiffeID,
		},
	}

	realInput, err := opautils.PreparedOpaInput(conn, authInfo)
	assert.Nil(t, err)

	assert.Equal(t, expectedInput, realInput)
}
