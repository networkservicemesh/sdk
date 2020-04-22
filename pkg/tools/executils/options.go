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
	"io"
	"os"
	"os/exec"
	"path"
)

// CmdOption - can be used with Start,Run,or Output to optionally set other exec.Cmd parameters
type CmdOption func(cmd *exec.Cmd) error

// WithDir - CmdOption that will create the requested dir if it does not exist and set exec.Cmd.Dir = dir
func WithDir(dir string) CmdOption {
	return func(cmd *exec.Cmd) error {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(path.Dir(dir), 0750); err != nil {
				return err
			}
		}
		cmd.Dir = dir
		return nil
	}
}

// WithStdout - option to set exec.Cmd.Stdout
func WithStdout(writer io.Writer) CmdOption {
	return func(cmd *exec.Cmd) error {
		cmd.Stdout = writer
		return nil
	}
}

// WithStderr - option to set exec.Cmd.Stdout
func WithStderr(writer io.Writer) CmdOption {
	return func(cmd *exec.Cmd) error {
		cmd.Stderr = writer
		return nil
	}
}

// WithEnviron - option to set exec.Cmd.Env, Combine os environ with passed values
func WithEnviron(env []string) CmdOption {
	return func(cmd *exec.Cmd) error {
		cmd.Env = append(os.Environ(), env...)
		return nil
	}
}
