// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

package kernel

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/bits"
	"net/url"

	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
)

const (
	// Alphabet is the alphabet for Nano ID.
	Alphabet = "!\"#$&'()*+,-.012456789;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	ifPrefix = "nsm"
)

var netNSURL = (&url.URL{Scheme: "file", Path: "/proc/thread-self/ns/net"}).String()

// generateInterfaceName - returns a random interface name with "nsm" prefix
func generateInterfaceName() string {
	ifIDLen := kernelmech.LinuxIfMaxLength - len(ifPrefix)
	id, _ := GenerateRandomString(ifIDLen)
	name := fmt.Sprintf("%s%s", ifPrefix, id)

	return limitName(name)
}

func limitName(name string) string {
	if len(name) > kernelmech.LinuxIfMaxLength {
		return name[:kernelmech.LinuxIfMaxLength]
	}
	return name
}

func generateRandomBuffer(step int) ([]byte, error) {
	buffer := make([]byte, step)
	if _, err := rand.Read(buffer); err != nil {
		return nil, err
	}
	return buffer, nil
}

// GenerateRandomString generates a random string based on size.
func GenerateRandomString(size int) (string, error) {
	mask := 2<<uint32(31-bits.LeadingZeros32(uint32(len(Alphabet)-1|1))) - 1
	step := int(math.Ceil(1.6 * float64(mask*size) / float64(len(Alphabet))))

	id := make([]byte, size)

	for {
		randomBuffer, err := generateRandomBuffer(step)
		if err != nil {
			return "", err
		}

		j := 0
		for i := 0; i < step; i++ {
			currentIndex := int(randomBuffer[i]) & mask

			if currentIndex < len(Alphabet) {
				id[j] = Alphabet[currentIndex]
				j++
				if j == size {
					return string(id), nil
				}
			}
		}
	}
}
