// Copyright 2022 Linka Cloud  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpc_rbac

import (
	"fmt"
	"strings"

	"github.com/mikespook/gorbac/v2"
)

var _ Permission = (*GRPCPermission)(nil)

type Permission = gorbac.Permission

func NewLayerPermission(name ...string) Permission {
	return gorbac.NewLayerPermission(strings.Join(name, ":"))
}

type GRPCPermission struct {
	fullMethod         string
	serviceName        string
	methodOrStreamName string
}

func NewGRPCPermission(serviceName string, methodOrStreamName string) Permission {
	return GRPCPermission{
		fullMethod:         fmt.Sprintf("/%s/%s", serviceName, methodOrStreamName),
		serviceName:        serviceName,
		methodOrStreamName: methodOrStreamName,
	}
}

// ID returns the identity of permission
func (p GRPCPermission) ID() string {
	return p.fullMethod
}

// Match another permission
func (p GRPCPermission) Match(a gorbac.Permission) bool {
	return p.fullMethod == a.ID()
}
