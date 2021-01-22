// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package jaeger provides a set of utilities for assisting with using jaeger
package jaeger

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/networkservicemesh/sdk/pkg/tools/logger/logruslogger"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	opentracingEnv     = "TRACER_ENABLED"
	opentracingDefault = true
)

// IsOpentracingEnabled returns true if opentracing enabled
func IsOpentracingEnabled() bool {
	val, err := readEnvBool(opentracingEnv, opentracingDefault)
	if err == nil {
		return val
	}
	return opentracingDefault
}

func readEnvBool(env string, value bool) (bool, error) {
	str := os.Getenv(env)
	if str == "" {
		return value, nil
	}

	return strconv.ParseBool(str)
}

type emptyCloser struct {
}

func (*emptyCloser) Close() error {
	// Ignore
	return nil
}

// InitJaeger -  returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitJaeger(service string) io.Closer {
	if !IsOpentracingEnabled() {
		return &emptyCloser{}
	}
	_, log := logruslogger.New(context.Background())
	if opentracing.IsGlobalTracerRegistered() {
		log.Warnf("global opentracer is already initialized")
	}
	cfg, err := config.FromEnv()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot create Jaeger configuration: %v\n", err))
	}

	if cfg.ServiceName == "" {
		var hostname string
		hostname, err = os.Hostname()
		if err == nil {
			cfg.ServiceName = fmt.Sprintf("%s@%s", service, hostname)
		} else {
			cfg.ServiceName = service
		}
	}
	if cfg.Sampler.Type == "" {
		cfg.Sampler.Type = "const"
	}
	if cfg.Sampler.Param == 0 {
		cfg.Sampler.Param = 1
	}
	if !cfg.Reporter.LogSpans {
		cfg.Reporter.LogSpans = true
	}

	log.Infof("Creating logger from config: %v", cfg)
	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		log.Errorf("ERROR: cannot init Jaeger: %v\n", err)
		return &emptyCloser{}
	}
	opentracing.SetGlobalTracer(tracer)
	return closer
}
