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

// Output - Run cmdStr using exec.Output and returns the resulting output as []byte and error
// Stderr are set to log.Entry(ctx).Writer().
func Output(ctx context.Context, cmdStr string, options ...CmdOption) ([]byte, error) {
	args := regexp.MustCompile(`\s+`).Split(cmdStr, -1)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = log.Entry(ctx).Writer()
	for _, option := range options {
		if err := option(cmd); err != nil {
			return nil, err
		}
	}
	return cmd.Output()
}
