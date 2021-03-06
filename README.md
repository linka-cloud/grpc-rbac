# grpc-rbac

*This project is currently in **alpha**. The API should be considered unstable and likely to change*


**grpc-rbac** is a minimalist wrapper around [go-rbac (v2)](github.com/mikespook/gorbac) which provide
a small **rbac** engine and *gRPC interceptors*. 

**protoc-gen-go-rbac** is a protoc plugin generating the service permissions and a method to register permissions
to the **rbac** engine.

### Install

```bash
go get -v go.linka.cloud/grpc-rbac/cmd/protoc-gen-go-rbac
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

message Resource {
  string id = 1;
}

service ResourceService {
  rpc Create(CreateRequest) returns (CreateResponse);
  rpc Read(ReadRequest) returns (ReadResponse);
  rpc Update(UpdateRequest) returns (UpdateResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  rpc List(ListRequest) returns (ListResponse);
  rpc Watch(WatchRequest) returns (stream Event);
}

// ... requests, responses and event definitions ...
```

The following **example.pb.rbac.go** will be generated:

```go
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

func RegisterResourceServicePermissions(rbac grpc_rbac.RBAC) {
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

```
