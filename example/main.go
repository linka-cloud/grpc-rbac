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

package main

import (
	"context"
	"log"
	"time"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grbac "go.linka.cloud/grpc-rbac"
	example "go.linka.cloud/grpc-rbac/example/pb"
)

const (
	// roleKey is the metadata key where roles are stored
	roleKey = "roles"

	// some roles
	admin   = "admin"
	reader  = "reader"
	writer  = "writer"
	watcher = "watcher"
)

var (
	// map roles to permissions
	roles = map[string][]grbac.Permission{
		reader: {
			example.ResourceServicePermissions.Read,
			example.ResourceServicePermissions.List,
		},
		writer: {
			example.ResourceServicePermissions.Create,
			example.ResourceServicePermissions.Update,
			example.ResourceServicePermissions.Delete,
		},
		watcher: {
			example.ResourceServicePermissions.Watch,
		},
	}
)

// rbacCtx set role in request outgoing metadata
func rbacCtx(ctx context.Context, role string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, roleKey, role)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create the rbac engine with our metadata role extraction function
	rbac := grbac.New(grbac.WithRoleFunc(func(ctx context.Context) ([]grbac.Role, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing role from metadata")
		}
		var roles []grbac.Role
		for _, v := range md.Get(roleKey) {
			log.Printf("role: %s", v)
			roles = append(roles, grbac.NewStdRole(v))
		}
		return roles, nil
	}))

	// create the admin role
	adminRole := grbac.NewStdRole(admin)
	if err := rbac.Add(adminRole); err != nil {
		log.Fatal(err)
	}

	// create our roles
	for name, perms := range roles {
		role := grbac.NewStdRole(name)
		for _, perm := range perms {
			if err := role.Assign(perm); err != nil {
				log.Fatal(err)
			}
		}
		if err := rbac.Add(role); err != nil {
			log.Fatal(err)
		}
		if err := rbac.SetParent(adminRole.ID(), role.ID()); err != nil {
			log.Fatal(err)
		}
	}

	// create the service
	svc := NewResourceService()

	// register the service permissions
	example.RegisterResourceServicePermissions(rbac)

	// create the in-process grpc channel
	channel := (&inprocgrpc.Channel{}).
		WithServerUnaryInterceptor(rbac.UnaryServerInterceptor()).
		WithServerStreamInterceptor(rbac.StreamServerInterceptor())

	// register the service as usual
	example.RegisterResourceServiceServer(channel, svc)

	// create a client
	client := example.NewResourceServiceClient(channel)

	// validate checks
	if _, err := client.List(rbacCtx(ctx, reader), &example.ListRequest{}); err != nil {
		log.Fatal(err)
	}

	if _, err := client.Create(rbacCtx(ctx, reader), &example.CreateRequest{Payload: &example.Resource{Id: "0"}}); err == nil {
		log.Fatal("reader should not be able to create")
	}

	if _, err := client.Create(rbacCtx(ctx, writer), &example.CreateRequest{Payload: &example.Resource{Id: "0"}}); err != nil {
		log.Fatal(err)
	}

	ss, err := client.Watch(rbacCtx(ctx, writer), &example.WatchRequest{})
	if err != nil {
		log.Fatal(err)
	}
	// try to receive an event as the interceptor is not called when creating the stream
	if _, err := ss.Recv(); err == nil {
		log.Fatal("writer should not be able to watch")
	}

	ss, err = client.Watch(rbacCtx(ctx, admin), &example.WatchRequest{})
	if err != nil {
		log.Fatal(err)
	}

	// create a resource to trigger an event
	go func() {
		time.Sleep(100 * time.Millisecond)
		if _, err := client.Create(rbacCtx(ctx, writer), &example.CreateRequest{Payload: &example.Resource{Id: "1"}}); err != nil {
			log.Fatal(err)
		}
	}()
	if _, err := ss.Recv(); err != nil {
		log.Fatal(err)
	}
}
