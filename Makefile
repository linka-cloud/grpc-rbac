# Copyright 2022 Linka Cloud  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

MODULE := go.linka.cloud/grpc-rbac

PROTO_BASE_PATH = .

$(shell mkdir -p .bin)

export GOBIN=$(PWD)/.bin

export PATH := $(GOBIN):$(PATH)

PROTO_OPTS = paths=source_relative

.PHONY: install
install:
	@cd ./cmd/protoc-gen-go-rbac; go install .

.PHONY: lint
lint:
	@gofmt -w $(PWD)
	@goimports -w -local $(MODULE) $(PWD)

.PHONY: proto
proto: gen-proto lint

.PHONY: gen-proto
gen-proto: install
	@protoc -I. --go_out=$(PROTO_OPTS):. --go-grpc_out=$(PROTO_OPTS):. --go-rbac_out=$(PROTO_OPTS):. example/pb/example.proto

clean:
	@rm -rf .bin
	@find $(PROTO_BASE_PATH) -name '*.pb*.go' -type f -exec rm {} \;
