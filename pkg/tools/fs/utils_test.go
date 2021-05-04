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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

func Test_WatchFile(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	root := filepath.Join(os.TempDir(), t.Name())

	path := filepath.Join(root, uuid.New().String())
	err := os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filePath := filepath.Join(path, "file1.txt")
	ch := fs.WatchFile(ctx, filePath)

	expectEvent := func() []byte {
		select {
		case <-time.After(time.Second):
			debug.PrintStack()
			t.Fatal("timeout waiting for event", filePath)
		case event := <-ch:
			return event
		}
		return nil
	}

	require.Nil(t, expectEvent(), filePath) // Initial file read. nil because file doesn't exist yet

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	require.NoError(t, err)
	require.NotNil(t, expectEvent(), filePath) // file created

	_, err = f.Write([]byte("data"))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	require.NotNil(t, expectEvent(), filePath) // file write

	err = os.RemoveAll(path)
	require.NoError(t, err)
	require.Nil(t, expectEvent(), filePath) // file removed

	if os.MkdirAll(path, os.ModePerm) != nil {
		// Removing file is async operation.
		// Waiting for events should theoretically sync us with the filesystem,
		// but apparently sometimes it's not enough, so MkdirAll can fail because the folder is still locked by the remove operation.
		// Particularly, this can be observed on slow Windows systems.
		require.Eventually(t, func() bool {
			return os.MkdirAll(path, os.ModePerm) == nil
		}, time.Millisecond*300, time.Millisecond*50)
	}

	err = ioutil.WriteFile(filePath, []byte("data"), os.ModePerm)
	require.NoError(t, err)
	require.NotNil(t, expectEvent(), filePath) // file created
}
