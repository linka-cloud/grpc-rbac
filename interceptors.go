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
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Interceptors interface {
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
	UnaryClientInterceptor() grpc.UnaryClientInterceptor
	StreamClientInterceptor() grpc.StreamClientInterceptor
}

func (r *rbac) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx = context.WithValue(ctx, key{}, r)
		if err := r.match(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (r *rbac) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := context.WithValue(ss.Context(), key{}, r)
		if err := r.match(ss.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, &wrapper{ctx: ctx, ServerStream: ss})
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
		return status.Errorf(codes.PermissionDenied, "[%s]: not allowed to call %s", strings.Join(ids, ", "), fullMethod)
	}
	return nil
}

type wrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrapper) Context() context.Context {
	return w.ctx
}
