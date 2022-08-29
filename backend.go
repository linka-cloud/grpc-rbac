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
	"github.com/mikespook/gorbac/v2"
)

var _ RBACBackend = (*rbac)(nil)

type AssertionFunc = gorbac.AssertionFunc

type RBACBackend interface {
	SetParents(role string, parents ...string) error
	GetParents(id string) ([]string, error)
	SetParent(id string, parent string) error
	RemoveParent(id string, parent string) error
	Add(r Role) (err error)
	Remove(id string) (err error)
	Get(id string) (r Role, parents []string, err error)
	IsGranted(id string, p Permission, assert AssertionFunc) (rslt bool)

	Walk(h gorbac.WalkHandler) error
	InherCircle() (err error)
	AnyGranted(roles []string, permission Permission, assert AssertionFunc) (rslt bool)
	AllGranted(roles []string, permission Permission, assert AssertionFunc) (rslt bool)
}

func (r *rbac) SetParents(id string, parents ...string) error {
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

func (r *rbac) Walk(h gorbac.WalkHandler) error {
	return gorbac.Walk(r.rbac, h)
}

func (r *rbac) InherCircle() (err error) {
	return gorbac.InherCircle(r.rbac)
}

func (r *rbac) AnyGranted(roles []string, permission Permission, assert AssertionFunc) (rslt bool) {
	return gorbac.AnyGranted(r.rbac, roles, permission, assert)
}

func (r *rbac) AllGranted(roles []string, permission Permission, assert AssertionFunc) (rslt bool) {
	return gorbac.AllGranted(r.rbac, roles, permission, assert)
}
