// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package spire provides two simple functions:
//   - Start to start a SpireServer/SpireAgent for local testing
//   - AddEntry to add entries into the spire server
package spire

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/workload"

	"github.com/matryer/try"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/errctx"
	"github.com/networkservicemesh/sdk/pkg/tools/executils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// AddEntry - adds an entry to the spire server for parentID, spiffeID, and selector
//            parentID is usually the same as the agentID provided to Start()
func AddEntry(ctx context.Context, parentID, spiffeID, selector string) error {
	cmdStr := "spire-server entry create -parentID %s -spiffeID %s -selector %s -registrationUDSPath %s/spire-registration.sock"
	cmdStr = fmt.Sprintf(cmdStr, parentID, spiffeID, selector, spireRoot)
	return executils.Run(log.WithField(ctx, "cmd", cmdStr), cmdStr)
}

var spireRoot string = ""

// Start - start a spire-server and spire-agent with the given agentId
func Start(ctx context.Context, agentID string) (context.Context, error) {
	// Setup our context
	ctx, cancel := context.WithCancel(ctx)
	ctx = errctx.WithErr(ctx)

	var err error
	spireRoot, err = ioutil.TempDir("", "spire")
	if err != nil {
		errctx.SetErr(ctx, err)
		cancel()
		return nil, err
	}

	// Write the config files (if not present)
	var spireSocketPath string
	spireSocketPath, err = writeDefaultConfigFiles(ctx, spireRoot)

	if err != nil {
		errctx.SetErr(ctx, err)
		cancel()
		return ctx, err
	}
	logrus.Infof("Env variable %s=%s are set", workload.SocketEnv, "unix:"+spireSocketPath)
	if err = os.Setenv(workload.SocketEnv, "unix:"+spireSocketPath); err != nil {
		errctx.SetErr(ctx, err)
		cancel()
		return ctx, err
	}

	// Start the Spire Server
	spireCmd := fmt.Sprintf("spire-server run -config %s", path.Join(spireRoot, spireServerConfFileName))
	spireServerCtx, err := executils.Start(ctx, spireCmd, executils.WithDir(spireRoot))
	if err != nil {
		errctx.SetErr(ctx, errors.Wrap(err, "Error starting spire-server"))
		cancel()
		return ctx, err
	}

	// Healthcheck the Spire Server
	err = try.Do(func(attempts int) (bool, error) {
		return attempts < 10, executils.Run(ctx, fmt.Sprintf("spire-server healthcheck -registrationUDSPath %s/spire-registration.sock", spireRoot))
	})
	if err != nil {
		errctx.SetErr(ctx, errors.Wrap(err, "Error starting spire-server"))
		cancel()
		return ctx, err
	}

	// Get the SpireServers Token
	cmdStr := "spire-server token generate -spiffeID %s -registrationUDSPath %s/spire-registration.sock"
	cmdStr = fmt.Sprintf(cmdStr, agentID, spireRoot)
	outputBytes, err := executils.Output(log.WithField(ctx, "cmd", cmdStr), cmdStr)
	if err != nil {
		errctx.SetErr(ctx, errors.Wrap(err, "Error acquiring spire-server token"))
		cancel()
		return ctx, err
	}
	spireToken := strings.Replace(string(outputBytes), "Token:", "", 1)
	spireToken = strings.TrimSpace(spireToken)

	// Start the Spire Agent
	spireAgentCtx, err := executils.Start(log.WithField(ctx, "cmd", "spire-agent run"), "spire-agent run"+" -config "+spireAgentConfFilename+" -joinToken "+spireToken,
		executils.WithDir(spireRoot))
	if err != nil {
		errctx.SetErr(ctx, errors.Wrap(err, "Error starting spire-agent"))
		cancel()
		return ctx, err
	}

	// Healthcheck the Spire Agent
	err = try.Do(func(attempts int) (bool, error) {
		cmdStr := fmt.Sprintf("spire-agent healthcheck -socketPath %s", spireSocketPath)
		return attempts < 10, executils.Run(log.WithField(ctx, "cmd", cmdStr), cmdStr)
	})
	if err != nil {
		errctx.SetErr(ctx, errors.Wrap(err, "Error starting spire-server"))
		cancel()
		return ctx, err
	}

	// Cleanup if either server we spawned dies
	go func() {
		select {
		case <-spireServerCtx.Done():
			errctx.SetErr(ctx, errors.Errorf("spireServer quit unexpectedly: %+v", errctx.Err(spireServerCtx)))
			cancel()
		case <-spireAgentCtx.Done():
			errctx.SetErr(ctx, errors.Errorf("SpireAgent quit unexpectedly: %+v", errctx.Err(spireAgentCtx)))
			cancel()
		case <-ctx.Done():
		}
		// Remove spire temporary folder
		_ = os.RemoveAll(spireRoot)
	}()
	return ctx, nil
}

// writeDefaultConfigFiles - write config files into configRoot and return a spire socket file to use
func writeDefaultConfigFiles(ctx context.Context, spireRoot string) (string, error) {
	spireSocketName := path.Join(spireRoot, spireEndpointSocket)
	configFiles := map[string]string{
		spireServerConfFileName: fmt.Sprintf(spireServerConfContents, spireRoot, spireServerRegSock),
		spireAgentConfFilename:  fmt.Sprintf(spireAgentConfContents, spireRoot, spireEndpointSocket),
	}
	for configName, contents := range configFiles {
		filename := path.Join(spireRoot, configName)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			log.Entry(ctx).Infof("Configuration file: %q not found, using defaults", filename)
			if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
				return "", err
			}
			if err := ioutil.WriteFile(filename, []byte(contents), 0700); err != nil {
				return "", err
			}
		}
	}
	return spireSocketName, nil
}
