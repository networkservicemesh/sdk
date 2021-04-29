// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package fs_test

import (
	"context"
	iofs "io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

func checkPathError(t *testing.T, err error) {
	if err == nil {
		return
	}

	if ferr, ok := err.(*iofs.PathError); ok {
		if u := ferr.Unwrap(); u != nil {
			t.Log(u)
		}
	}
	require.Nil(t, err)
}

func Test_WatchFile(t *testing.T) {
	root := filepath.Join(os.TempDir(), t.Name())

	path := filepath.Join(root, uuid.New().String())
	err := os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	checkPathError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filePath := filepath.Join(path, "file1.txt")
	ch := fs.WatchFile(ctx, filePath)

	expectEvent := func(assertFunc func(require.TestingT, interface{}, ...interface{})) {
		select {
		case <-time.After(time.Second):
			debug.PrintStack()
			t.Fatal("no events")
		case update := <-ch:
			assertFunc(t, update)
		}
	}

	expectEvent(require.Nil)

	checkPathError(t, ioutil.WriteFile(filePath, []byte("data"), os.ModePerm))

	expectEvent(require.NotNil)

	// https://github.com/fsnotify/fsnotify/issues/11
	if runtime.GOOS != "darwin" {
		expectEvent(require.NotNil)
	}

	err = os.RemoveAll(path)
	checkPathError(t, err)

	expectEvent(require.Nil)

	err = os.MkdirAll(path, os.ModePerm)
	checkPathError(t, err)
	checkPathError(t, ioutil.WriteFile(filePath, []byte("data"), os.ModePerm))

	expectEvent(require.NotNil)
}
