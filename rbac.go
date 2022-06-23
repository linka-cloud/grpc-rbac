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
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/mikespook/gorbac/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ RBAC       = (*rbac)(nil)
	_ Permission = (*GRPCPermission)(nil)

	NewStdRole    = gorbac.NewStdRole
	NewPermission = gorbac.NewStdPermission
)

type (
	Permission = gorbac.Permission
	Role       = gorbac.Role
)

type RoleFunc func(ctx context.Context) ([]gorbac.Role, error)

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

type RBAC interface {
	SetParents(id string, parents []string) error
	GetParents(id string) ([]string, error)
	SetParent(id string, parent string) error
	RemoveParent(id string, parent string) error
	Add(r gorbac.Role) (err error)
	Remove(id string) (err error)
	Get(id string) (r gorbac.Role, parents []string, err error)
	IsGranted(id string, p gorbac.Permission, assert gorbac.AssertionFunc) (rslt bool)

	Register(desc *grpc.ServiceDesc)

	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
	UnaryClientInterceptor() grpc.UnaryClientInterceptor
	StreamClientInterceptor() grpc.StreamClientInterceptor
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

func (r *rbac) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if err := r.match(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (r *rbac) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Printf("%s: checking rbac", info.FullMethod)
		if err := r.match(ss.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func (r *rbac) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := r.match(ctx, method); err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (r *rbac) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if err := r.match(ctx, method); err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func (r *rbac) match(ctx context.Context, fullMethod string) error {
	roles, err := r.roleFunc(ctx)
	if err != nil {
		return err
	}
	v, ok := r.reg.Load(fullMethod)
	if !ok {
		return fmt.Errorf("permission for '%s' not found", fullMethod)
	}
	perm := v.(GRPCPermission)
	granted := false
	var ids []string
	for _, v := range roles {
		if granted = r.rbac.IsGranted(v.ID(), perm, nil); granted {
			break
		}
		ids = append(ids, v.ID())
	}
	if !granted {
		return status.Errorf(codes.PermissionDenied, "%s: [%s]: not allowed to call %s", strings.Join(ids, ", "))
	}
	return nil
}

func (r *rbac) SetParents(id string, parents []string) error {
	return r.rbac.SetParents(id, parents)
}

func (r *rbac) GetParents(id string) ([]string, error) {
	return r.rbac.GetParents(id)
}

func (r *rbac) SetParent(id string, parent string) error {
	return r.rbac.SetParent(id, parent)
}

func (r *rbac) RemoveParent(id string, parent string) error {
	return r.rbac.RemoveParent(id, parent)
}

func (r *rbac) Add(role gorbac.Role) (err error) {
	return r.rbac.Add(role)
}

func (r *rbac) Remove(id string) (err error) {
	return r.rbac.Remove(id)
}

func (r *rbac) Get(id string) (role gorbac.Role, parents []string, err error) {
	return r.rbac.Get(id)
}

func (r *rbac) IsGranted(id string, p gorbac.Permission, assert gorbac.AssertionFunc) (rslt bool) {
	return r.rbac.IsGranted(id, p, assert)
}

func UnimplementedRoleFunc(ctx context.Context) ([]gorbac.Role, error) {
	return nil, errors.New("grpc rbac: missing role function")
}
