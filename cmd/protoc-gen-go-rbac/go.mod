module go.linka.cloud/grpc-rbac/cmd/protoc-gen-go-rbac

go 1.16

require (
	github.com/lyft/protoc-gen-star v0.6.0
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/sirupsen/logrus v1.9.0
	go.linka.cloud/grpc-rbac v0.0.0-00010101000000-000000000000
)

replace go.linka.cloud/grpc-rbac => ../..
