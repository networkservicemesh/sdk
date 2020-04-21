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

package executils

import (
	"context"
	"os/exec"
	"regexp"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Run - Run cmdStr using exec.Run and returns the resulting error
// StdOut and Stderr are set to log.Entry(ctx).Writer().
func Run(ctx context.Context, cmdStr string, options ...CmdOption) error {
	args := regexp.MustCompile(`\s+`).Split(cmdStr, -1)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = log.Entry(ctx).Writer()
	cmd.Stderr = log.Entry(ctx).Writer()
	for _, option := range options {
		if err := option(cmd); err != nil {
			return err
		}
	}
	return cmd.Run()
}
