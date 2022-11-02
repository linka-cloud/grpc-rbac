# grpc-rbac

*This project is currently in **alpha**. The API should be considered unstable and likely to change*


**grpc-rbac** is a minimalist wrapper around [go-rbac (v2)](https://github.com/mikespook/gorbac) which provide
a small **rbac** engine and *gRPC interceptors*. 

**protoc-gen-go-rbac** is a protoc plugin generating the service permissions and a method to register permissions
to the **rbac** engine.

### Install

```bash
go get -v go.linka.cloud/grpc-rbac
go install go.linka.cloud/grpc-rbac/cmd/protoc-gen-go-rbac
```

### Import

```go
import grpc_rbac "go.linka.cloud/grpc-rbac"
```

### Usage

Use the plugin as any other **protoc** plugins.

### Generated code

For a given **example.proto**:

```protobuf
syntax = "proto3";

package example;

option go_package = "go.linka.cloud/grpc-rbac/example";

import "rbac/rbac.proto";

message Resource {
  string id = 1;
}

service ResourceService {
  option(rbac.def) = {
    roles: [{
      name: "admin",
      parents: ["writer", "reader", "watcher"],
    }],
  };
  rpc Create(CreateRequest) returns (CreateResponse) {
    option (rbac.access) = {
      roles: ["writer"]
    };
  }
  rpc Read(ReadRequest) returns (ReadResponse) {
    option (rbac.access) = {
      roles: ["reader"]
    };
  }
  rpc Update(UpdateRequest) returns (UpdateResponse) {
    option (rbac.access) = {
      roles: ["writer"]
    };
  };
  rpc Delete(DeleteRequest) returns (DeleteResponse) {
    option (rbac.access) = {
      roles: ["writer"]
    };
  };
  rpc List(ListRequest) returns (ListResponse) {
    option (rbac.access) = {
      roles: ["reader"]
    };
  };
  rpc Watch(WatchRequest) returns (stream Event) {
    option (rbac.access) = {
      roles: ["watcher"]
    };
  };
}

// ... requests, responses and event definitions ...
```

The following **example.pb.rbac.go** will be generated:

```go
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

// Code generated by protoc-gen-go-rbac. DO NOT EDIT.
package example

import (
	grpc_rbac "go.linka.cloud/grpc-rbac"
)

var ResourceServicePermissions = struct {
	Create grpc_rbac.Permission
	Read   grpc_rbac.Permission
	Update grpc_rbac.Permission
	Delete grpc_rbac.Permission
	List   grpc_rbac.Permission
	Watch  grpc_rbac.Permission
}{
	Create: grpc_rbac.NewGRPCPermission("example.ResourceService", "Create"),
	Read:   grpc_rbac.NewGRPCPermission("example.ResourceService", "Read"),
	Update: grpc_rbac.NewGRPCPermission("example.ResourceService", "Update"),
	Delete: grpc_rbac.NewGRPCPermission("example.ResourceService", "Delete"),
	List:   grpc_rbac.NewGRPCPermission("example.ResourceService", "List"),
	Watch:  grpc_rbac.NewGRPCPermission("example.ResourceService", "Watch"),
}

var ResourceServiceRoles = struct {
	Admin   *grpc_rbac.StdRole
	Reader  *grpc_rbac.StdRole
	Watcher *grpc_rbac.StdRole
	Writer  *grpc_rbac.StdRole
}{
	Admin:   grpc_rbac.NewStdRole("ResourceService.Admin"),
	Reader:  grpc_rbac.NewStdRole("ResourceService.Reader"),
	Watcher: grpc_rbac.NewStdRole("ResourceService.Watcher"),
	Writer:  grpc_rbac.NewStdRole("ResourceService.Writer"),
}

func RegisterResourceServicePermissions(rbac grpc_rbac.RBAC) {
	// Register Admin role
	if err := rbac.Add(ResourceServiceRoles.Admin); err != nil {
		panic(err)
	}

	// Assign Reader permissions
	if err := ResourceServiceRoles.Reader.Assign(ResourceServicePermissions.Read); err != nil {
		panic(err)
	}
	if err := ResourceServiceRoles.Reader.Assign(ResourceServicePermissions.List); err != nil {
		panic(err)
	}
	// Register Reader role
	if err := rbac.Add(ResourceServiceRoles.Reader); err != nil {
		panic(err)
	}

	// Assign Watcher permissions
	if err := ResourceServiceRoles.Watcher.Assign(ResourceServicePermissions.Watch); err != nil {
		panic(err)
	}
	// Register Watcher role
	if err := rbac.Add(ResourceServiceRoles.Watcher); err != nil {
		panic(err)
	}

	// Assign Writer permissions
	if err := ResourceServiceRoles.Writer.Assign(ResourceServicePermissions.Create); err != nil {
		panic(err)
	}
	if err := ResourceServiceRoles.Writer.Assign(ResourceServicePermissions.Update); err != nil {
		panic(err)
	}
	if err := ResourceServiceRoles.Writer.Assign(ResourceServicePermissions.Delete); err != nil {
		panic(err)
	}
	// Register Writer role
	if err := rbac.Add(ResourceServiceRoles.Writer); err != nil {
		panic(err)
	}

	// Assign Admin parents
	if err := rbac.SetParent(ResourceServiceRoles.Admin.ID(), ResourceServiceRoles.Writer.ID()); err != nil {
		panic(err)
	}
	if err := rbac.SetParent(ResourceServiceRoles.Admin.ID(), ResourceServiceRoles.Reader.ID()); err != nil {
		panic(err)
	}
	if err := rbac.SetParent(ResourceServiceRoles.Admin.ID(), ResourceServiceRoles.Watcher.ID()); err != nil {
		panic(err)
	}

	// Register ResourceService Service rules
	rbac.Register(&ResourceService_ServiceDesc)
}

```

### Usage

See [the example directory](./example/) for complete example.

```go
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
)

// rbacCtx set role in request outgoing metadata
func rbacCtx(ctx context.Context, role grbac.Role) context.Context {
	return metadata.AppendToOutgoingContext(ctx, roleKey, role.ID())
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
	if _, err := client.List(rbacCtx(ctx, example.ResourceServiceRoles.Reader), &example.ListRequest{}); err != nil {
		log.Fatal(err)
	}

	if _, err := client.Create(rbacCtx(ctx, example.ResourceServiceRoles.Reader), &example.CreateRequest{Payload: &example.Resource{Id: "0"}}); err == nil {
		log.Fatal("reader should not be able to create")
	}

	if _, err := client.Create(rbacCtx(ctx, example.ResourceServiceRoles.Writer), &example.CreateRequest{Payload: &example.Resource{Id: "0"}}); err != nil {
		log.Fatal(err)
	}

	ss, err := client.Watch(rbacCtx(ctx, example.ResourceServiceRoles.Writer), &example.WatchRequest{})
	if err != nil {
		log.Fatal(err)
	}
	// try to receive an event as the interceptor is not called when creating the stream
	if _, err := ss.Recv(); err == nil {
		log.Fatal("writer should not be able to watch")
	}

	ss, err = client.Watch(rbacCtx(ctx, example.ResourceServiceRoles.Admin), &example.WatchRequest{})
	if err != nil {
		log.Fatal(err)
	}

	// create a resource to trigger an event
	go func() {
		time.Sleep(100 * time.Millisecond)
		if _, err := client.Create(rbacCtx(ctx, example.ResourceServiceRoles.Writer), &example.CreateRequest{Payload: &example.Resource{Id: "1"}}); err != nil {
			log.Fatal(err)
		}
	}()
	if _, err := ss.Recv(); err != nil {
		log.Fatal(err)
	}
}

```
