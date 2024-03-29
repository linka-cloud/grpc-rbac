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
	"fmt"
	"sort"
	"strings"
	"text/template"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"

	"go.linka.cloud/grpc-rbac/rbac"
)

func Module() *module {
	return &module{
		ModuleBase: &pgs.ModuleBase{},
	}
}

type module struct {
	*pgs.ModuleBase
	ctx pgsgo.Context
	tpl *template.Template
}

func (p *module) Name() string {
	return "rbac"
}

func (p *module) InitContext(c pgs.BuildContext) {
	p.ModuleBase.InitContext(c)
	p.ctx = pgsgo.InitContext(c.Parameters())

	type role struct {
		Name    string
		Value   string
		Perms   []string
		Parents []string
	}

	tpl := template.New("fields").Funcs(map[string]interface{}{
		"package": p.ctx.PackageName,
		"name":    p.ctx.Name,
		"comment": func(s string) string {
			var out string
			parts := strings.Split(s, "\n")
			for i, v := range parts {
				if i == len(parts)-1 && v == "" {
					return out
				}
				out += "//" + v + "\n"
			}
			return out
		},
		"roles": func(s pgs.Service) []*role {
			roles := make(map[string]*role)
			var def rbac.RoleDefinition
			_, err := s.Extension(rbac.E_Def, &def)
			if err != nil {
				p.Fail(err)
			}
			for _, v := range def.Roles {
				val := fmt.Sprintf("%s.%s", s.Name(), strings.Title(v.GetName()))
				r := &role{
					Name:  strings.Replace(strings.Title(strings.NewReplacer(".", " ", "-", " ", ":", " ", "_", " ").Replace(v.GetName())), " ", "", -1),
					Value: val,
				}
				for _, vv := range v.Parents {
					r.Parents = append(r.Parents, strings.Title(strings.Replace(strings.Title(strings.NewReplacer(".", " ", "-", " ", ":", " ", "_", " ").Replace(vv)), " ", "", -1)))
				}
				roles[val] = r
			}
			for _, m := range s.Methods() {
				o := &rbac.RBAC{}
				ok, err := m.Extension(rbac.E_Access, o)
				if err != nil {
					p.Fail(err)
				}
				if !ok {
					continue
				}
				for _, v := range o.Roles {
					val := fmt.Sprintf("%s.%s", s.Name(), strings.Title(v))
					if _, ok := roles[val]; !ok {
						roles[val] = &role{
							Name:  strings.Replace(strings.Title(strings.NewReplacer(".", " ", "-", " ", ":", " ", "_", " ").Replace(v)), " ", "", -1),
							Value: val,
						}
					}
					roles[val].Perms = append(roles[val].Perms, m.Name().String())
				}
			}
			var out []*role
			for _, v := range roles {
				out = append(out, v)
			}
			sort.Slice(out, func(i, j int) bool {
				return sort.StringsAreSorted([]string{out[i].Name, out[j].Name})
			})
			return out
		},
	})
	p.tpl = template.Must(tpl.Parse(fieldsTpl))
}

func (p *module) Execute(targets map[string]pgs.File, _ map[string]pgs.Package) []pgs.Artifact {
	for _, f := range targets {
		p.generate(f)
	}
	return p.Artifacts()
}

func (p *module) generate(f pgs.File) {
	if len(f.Services()) == 0 {
		return
	}
	name := p.ctx.OutputPath(f).SetExt(".rbac.go")
	p.AddGeneratorTemplateFile(name.String(), p.tpl, f)
}

const fieldsTpl = `{{ comment .SyntaxSourceCodeInfo.LeadingComments }}
{{ range .SyntaxSourceCodeInfo.LeadingDetachedComments }}
{{ comment . }}
{{ end }}
// Code generated by protoc-gen-go-rbac. DO NOT EDIT.
package {{ package . }}
{{ $file := . }}
import (
	grpc_rbac "go.linka.cloud/grpc-rbac"
)

{{ range .Services }}
{{- $svc := . }} 
var {{ .Name }}Permissions = struct {
	{{- range .Methods }}
	{{ name . }} grpc_rbac.Permission
	{{- end }}
}{
	{{- range .Methods }}
	{{ name . }}: grpc_rbac.NewGRPCPermission("{{ $file.Package.ProtoName }}.{{ $svc.Name }}", "{{ .Name }}"),
	{{- end }}
}

var {{ .Name }}Roles = struct {
	{{- range roles . }}
	{{ .Name }} *grpc_rbac.StdRole
	{{- end }}
}{
	{{- range roles . }}
	{{ .Name }}: grpc_rbac.NewStdRole("{{ .Value }}"),
	{{- end }}
}

func Register{{ .Name }}Permissions(rbac grpc_rbac.RBAC) {
	{{- range roles . }}
	{{- $role := . }}
	{{- if .Perms }}// Assign {{ .Name }} permissions{{ end }}
	{{- range .Perms }}
	if err := {{ $svc.Name }}Roles.{{ $role.Name }}.Assign({{ $svc.Name }}Permissions.{{ . }}); err != nil {
		panic(err)
	}
	{{- end }}
	// Register {{ .Name }} role
	if err := rbac.Add({{ $svc.Name }}Roles.{{ .Name }}); err != nil {
		panic(err)
	}
	
	{{ end }}
	{{- range roles . }}
	{{- $role := . }}
	{{- if .Parents }}// Assign {{ .Name }} parents{{ end }}
	{{- range .Parents }}
	if err := rbac.SetParent({{ $svc.Name }}Roles.{{ $role.Name }}.ID(), {{ $svc.Name }}Roles.{{ . }}.ID()); err != nil {
		panic(err)
	}
	{{- end }}
	{{ end }}

	// Register {{ .Name }} Service rules
	rbac.Register(&{{ .Name }}_ServiceDesc)
}

{{ end }}
`
