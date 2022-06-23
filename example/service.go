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
	"errors"
	"io"
	"sync"
	"time"

	"go.linka.cloud/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.linka.cloud/grpc-rbac/example/pb"
)

type resourceService struct {
	example.UnimplementedResourceServiceServer
	store  sync.Map
	pubsub pubsub.Publisher
}

func NewResourceService() example.ResourceServiceServer {
	return &resourceService{pubsub: pubsub.NewPublisher(time.Second, 1)}
}

func (r *resourceService) Create(ctx context.Context, request *example.CreateRequest) (*example.CreateResponse, error) {
	r.store.Store(request.GetPayload().GetId(), request.GetPayload())
	defer r.pubsub.Publish(&example.Event{Type: example.Event_CREATED, Payload: request.Payload})
	return &example.CreateResponse{Result: request.GetPayload()}, nil
}

func (r *resourceService) Read(ctx context.Context, request *example.ReadRequest) (*example.ReadResponse, error) {
	v, ok := r.store.Load(request.GetId())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "%s does not exists", request.GetId())
	}
	return &example.ReadResponse{Result: v.(*example.Resource)}, nil
}

func (r *resourceService) Update(ctx context.Context, request *example.UpdateRequest) (*example.UpdateResponse, error) {
	r.store.Store(request.GetPayload().GetId(), request.GetPayload())
	defer r.pubsub.Publish(&example.Event{Type: example.Event_UPDATED, Payload: request.Payload})
	return &example.UpdateResponse{Result: request.GetPayload()}, nil
}

func (r *resourceService) Delete(ctx context.Context, request *example.DeleteRequest) (*example.DeleteResponse, error) {
	v, ok := r.store.Load(request.GetId())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "%s does not exists", request.GetId())
	}
	defer r.pubsub.Publish(&example.Event{Type: example.Event_DELETED, Payload: v.(*example.Resource)})
	r.store.Delete(request.GetId())
	return &example.DeleteResponse{}, nil
}

func (r *resourceService) List(ctx context.Context, request *example.ListRequest) (*example.ListResponse, error) {
	res := &example.ListResponse{}
	r.store.Range(func(key, value interface{}) bool {
		res.Results = append(res.Results, value.(*example.Resource))
		return true
	})
	return res, nil
}

func (r *resourceService) Watch(_ *example.WatchRequest, server example.ResourceService_WatchServer) error {
	ch := r.pubsub.Subscribe()
	defer r.pubsub.Evict(ch)
	for {
		select {
		case e := <-ch:
			if err := server.Send(proto.Clone(e.(*example.Event)).(*example.Event)); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		case <-server.Context().Done():
			return server.Context().Err()
		}
	}
}
