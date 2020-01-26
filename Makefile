# Copyright (c) 2019-2020 Cisco and/or its affiliates.
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# We want to use bash
SHELL:=/bin/bash
WORKER_COUNT ?= 1

# Default target, no other targets should be before default
.PHONY: default
default: all

# Code formatting
include .mk/formatting.mk

# Static code analysis
include .mk/code_analysis.mk

GOPATH?=$(shell go env GOPATH 2>/dev/null)
GOCMD=go
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test
GOVET=${GOCMD} vet --all

# Export some of the above variables so they persist for the shell scripts
# which are run from the Makefiles
export GOPATH \
       GOCMD \
       GOGET \
       GOGENERATE \
       GOINSTALL \
       GOTEST \
       GOVET

.PHONY: all check verify # docker-build docker-push

all: check verify

check:
	@shellcheck `find . -name "*.sh" -not -path "*vendor/*"`; \

.PHONY: format deps generate install test test-race vet
#
# The following targets are meant to be run when working with the code locally.
#
deps:


generate:
	@${GOGENERATE} ./...

install:
	@${GOINSTALL} ./...

test:
	@${GOTEST} $$(go list ./... | grep -v -e "sample")

test-race:
	@${GOTEST} -race ./... -cover

vet:
	@${GOVET} ./...
