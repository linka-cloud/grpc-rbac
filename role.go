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
	"context"
	"errors"

	"github.com/mikespook/gorbac/v2"
)

type (
	Role    = gorbac.Role
	StdRole = gorbac.StdRole
)

func NewStdRole(name string) *StdRole {
	return gorbac.NewStdRole(name)
}

type RoleFunc func(ctx context.Context) ([]Role, error)

func UnimplementedRoleFunc(ctx context.Context) ([]gorbac.Role, error) {
	return nil, errors.New("grpc rbac: missing role function")
}
