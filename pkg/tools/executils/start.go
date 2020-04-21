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

// Package executils provides convenience utilities that closely mirror "exec".  In particular it allows;
// - Use of a cmdStr string rather than forcing it to be broken up into 'name' and 'args' as in exec.
// - CmdOptions to set optional exec.Cmd parameters
// - Use of returned context.Context from Start which will be canceled after exec.Cmd.Wait returns.
//     - Any error from Wait is accessible from the returned context with errctx.Err(ctx)
// - By default (overridable with options) exec.Cmd.Stdout and exec.Cmd.Stdin are set to the output of
//   log.Entry(ctx).Writer().  The output then goes to a Writer for the desired logrus Entry.
//   This allows fun tricks like:
//       executils.Start(log.WithFields("cmd",cmdStr),cmdStr)
//   which will cause all output from the cmd to carry a field showing the command being run
package executils

import (
	"context"
	"os/exec"
	"regexp"

	"github.com/networkservicemesh/sdk/pkg/tools/errctx"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Start - Starts cmdStr using exec.Start.  Returns a context that will be canceled when exec.Cmd.Wait() returns
// With any error returned by exec.Cmd.Wait available by using errctx.Err(ctx) on the returned context
// StdOut and Stderr are set to log.Entry(ctx).Writer().
func Start(ctx context.Context, cmdStr string, options ...CmdOption) (context.Context, error) {
	ctx, cancel := context.WithCancel(ctx)
	ctx = errctx.WithErr(ctx)
	args := regexp.MustCompile(`\s+`).Split(cmdStr, -1)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = log.Entry(ctx).Writer()
	cmd.Stderr = log.Entry(ctx).Writer()
	for _, option := range options {
		if err := option(cmd); err != nil {
			errctx.SetErr(ctx, err)
			cancel()
			return ctx, err
		}
	}
	if err := cmd.Start(); err != nil {
		errctx.SetErr(ctx, err)
		cancel()
		return ctx, err
	}
	go func() {
		errctx.SetErr(ctx, cmd.Wait())
		cancel()
	}()
	return ctx, nil
}
