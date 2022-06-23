module go.linka.cloud/grpc-rbac/example

go 1.13

require (
	github.com/fullstorydev/grpchan v1.1.2-0.20220223040110-9b5ad76b6f3d
	go.linka.cloud/grpc-rbac v0.0.0-00010101000000-000000000000
	go.linka.cloud/pubsub v0.0.0-20211207164231-07a5a95fc4ff
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
)

replace go.linka.cloud/grpc-rbac => ./..
