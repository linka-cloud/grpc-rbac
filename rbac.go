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
	"fmt"
	"sync"

	"github.com/mikespook/gorbac/v2"
	"google.golang.org/grpc"
)

var _ RBAC = (*rbac)(nil)

type RBAC interface {
	RBACBackend
	Interceptors
	Register(desc *grpc.ServiceDesc)
}

func New(opts ...Option) RBAC {
	r := &rbac{rbac: gorbac.New()}
	for _, v := range opts {
		v(r)
	}
	if r.roleFunc == nil {
		r.roleFunc = UnimplementedRoleFunc
	}
	return r
}

type rbac struct {
	rbac     *gorbac.RBAC
	reg      sync.Map
	roleFunc RoleFunc
	assertFn gorbac.AssertionFunc
}

func (r *rbac) Register(desc *grpc.ServiceDesc) {
	for _, v := range desc.Methods {
		f := fmt.Sprintf("/%s/%s", desc.ServiceName, v.MethodName)
		r.reg.Store(f, GRPCPermission{fullMethod: f, serviceName: desc.ServiceName, methodOrStreamName: v.MethodName})
	}
	for _, v := range desc.Streams {
		f := fmt.Sprintf("/%s/%s", desc.ServiceName, v.StreamName)
		r.reg.Store(f, GRPCPermission{fullMethod: f, serviceName: desc.ServiceName, methodOrStreamName: v.StreamName})
	}
}

type key struct{}

func FromContext(ctx context.Context) (RBAC, bool) {
	v, ok := ctx.Value(key{}).(RBAC)
	return v, ok
}

func MustFromContext(ctx context.Context) RBAC {
	v, ok := FromContext(ctx)
	if !ok {
		panic("no RBAC in context")
	}
	return v
}
