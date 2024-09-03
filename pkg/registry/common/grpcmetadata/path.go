// Copyright (c) 2022 Cisco and/or its affiliates.
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

package grpcmetadata

// Path represents a private path that is passed via grpcmetadata during NS and NSE registration.
type Path struct {
	Index        uint32
	PathSegments []*PathSegment
}

// PathSegment represents segment of a private path.
type PathSegment struct {
	Token string `json:"token"`
}

// GetCurrentPathSegment returns path.Index segment if it exists.
func (p *Path) GetCurrentPathSegment() *PathSegment {
	if p == nil {
		return nil
	}
	if len(p.PathSegments) == 0 {
		return nil
	}
	if int(p.Index) >= len(p.PathSegments) {
		return nil
	}
	return p.PathSegments[p.Index]
}

// Clone clones Path.
func (p *Path) Clone() *Path {
	result := &Path{
		Index: p.Index,
	}

	for _, s := range p.PathSegments {
		result.PathSegments = append(result.PathSegments, &PathSegment{
			Token: s.Token,
		})
	}

	return result
}
