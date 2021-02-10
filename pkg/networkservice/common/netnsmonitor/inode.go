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

package netnsmonitor

import (
	"io/ioutil"
	"path"
	"unicode"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

// InodeSet is an index map of inodes
type InodeSet struct {
	inodes map[uint64]bool
}

// NewInodeSet creates new InodeSet from array
func NewInodeSet(inodes []uint64) *InodeSet {
	set := map[uint64]bool{}
	for _, inode := range inodes {
		set[inode] = true
	}
	return &InodeSet{inodes: set}
}

// Contains checks inode in IndexSet
func (i *InodeSet) Contains(inode uint64) bool {
	_, contains := i.inodes[inode]
	return contains
}

func isDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// GetAllNetNS returns all NetNS Inodes of all processes
func GetAllNetNS() ([]uint64, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, errors.Wrap(err, "can't read /proc directory")
	}
	inodes := make([]uint64, 0, len(files))
	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			filename := path.Join("/proc", name, "/ns/net")
			inode, err := fs.GetInode(filename)
			if err != nil {
				continue
			}
			inodes = append(inodes, uint64(inode))
		}
	}
	return inodes, nil
}
