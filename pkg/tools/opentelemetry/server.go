// Copyright (c) 2024 Nordix Foundation.
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

// Package opentelemetry provides a set of utilities for assisting with telemetry data
package opentelemetry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Server is a server type for handling requests
type Server struct {
	TLSConfig     *tls.Config
	IP            string
	Port          int
	HeaderTimeout time.Duration
}

// Start initiates the server to begin handling incoming requests
func (s *Server) Start(ctx context.Context) error {
	log.FromContext(ctx).Info("Start metrics server", "ip", s.IP, "port", s.Port)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.IP, s.Port),
		TLSConfig:         s.TLSConfig,
		ReadHeaderTimeout: s.HeaderTimeout,
	}

	http.Handle("/metrics", promhttp.Handler())

	serverCtx, cancel := context.WithCancel(ctx)
	var ListenAndServeErr error

	go func() {
		ListenAndServeErr = server.ListenAndServeTLS("", "")
		if ListenAndServeErr != nil {
			cancel()
		}
	}()

	<-serverCtx.Done()

	if ListenAndServeErr != nil {
		return errors.Errorf("failed to ListenAndServe on metrics server: %s", ListenAndServeErr)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return errors.Errorf("failed to shutdown metrics server: %s", err)
	}

	return nil
}
