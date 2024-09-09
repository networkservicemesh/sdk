// Copyright (c) 2021-2024 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Nordix Foundation.
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

package logruslogger

import (
	"bytes"
	"strings"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

// formatter - implements logrus.Formatter
// Duplicates fields and other metadata from nested-logrus-formatter to each line of the message.
type formatter struct {
	nf nested.Formatter
}

func newFormatter() *formatter {
	f := formatter{}
	f.nf.FieldsOrder = []string{"id", "name"}
	f.nf.NoColors = true
	return &f
}

// Format an log entry.
func (f *formatter) Format(entry *logrus.Entry) ([]byte, error) {
	formattedBytes, err := f.nf.Format(entry)
	if err != nil {
		return nil, err
	}

	bytesString := string(formattedBytes)
	lineBreakIndex := strings.Index(bytesString, "\n")
	if lineBreakIndex == -1 || lineBreakIndex == len(bytesString)-1 {
		return formattedBytes, nil
	}

	// output buffer
	bb := &bytes.Buffer{}

	bytesString = bytesString[:len(bytesString)-1] // remove trailing \n
	for _, line := range strings.Split(bytesString, "\n") {
		bb.WriteString(line)
		bb.WriteString(";\t")
	}
	bb.WriteString("\n")

	return bb.Bytes(), nil
}
