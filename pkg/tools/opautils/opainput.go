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

// Now we can consider the OPA input as a map with the keys:
//    1) "connection" -- we can use all information about connection in OPA (e.g. input.connection.path.path_segments[0].token)
//    2) "auth_info" with sub keys:
//          i) "certificate" -- is a pem encoded x509cert (usage: input.auth_info.certificate)
//			ii) "spiffe_id" -- is a spiffeID from SVIDX509Certificate (usage: input.auth_info.spiffe_id)

// An example of using OPA input for the case of token signature verification:
//
//		package test
//
//		default allow = false
//
//		allow {
//			token := input.connection.path.path_segments[0].token
//			cert := input.auth_info.certificate
//			io.jwt.verify_es256(token, cert) # signature verification
//		}
//

// Package opautils provides of utilities for using OPA
package opautils

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"

	"github.com/pkg/errors"

	"google.golang.org/grpc/credentials"
)

// PreparedOpaInput - returns a prepared input for using in OPA
func PreparedOpaInput(connection *networkservice.Connection, authInfo credentials.AuthInfo) (map[string]interface{}, error) {
	connectionAsMap, err := convertConnectionToMap(connection)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot convert connection %v to map", connection)
	}

	cert := parseX509Cert(authInfo)

	spiffeID, err := spiffeutils.SpiffeIDFromX509(cert)
	if err != nil {
		return nil, err
	}

	pemcert := pemEncodingX509Cert(cert)

	rv := map[string]interface{}{
		"connection": connectionAsMap,
		"auth_info": map[string]interface{}{
			"certificate": pemcert,
			"spiffe_id":   spiffeID,
		},
	}
	return rv, nil
}

func pemEncodingX509Cert(cert *x509.Certificate) string {
	certpem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return string(certpem)
}

func parseX509Cert(authInfo credentials.AuthInfo) *x509.Certificate {
	var peerCert *x509.Certificate
	if tlsInfo, ok := authInfo.(credentials.TLSInfo); ok {
		if len(tlsInfo.State.PeerCertificates) > 0 {
			peerCert = tlsInfo.State.PeerCertificates[0]
		}
	}
	return peerCert
}

func convertConnectionToMap(connection *networkservice.Connection) (map[string]interface{}, error) {
	jsonConn, err := json.Marshal(connection)
	if err != nil {
		return nil, err
	}
	rv := make(map[string]interface{})
	if err := json.Unmarshal(jsonConn, &rv); err != nil {
		return nil, err
	}
	return rv, nil
}
