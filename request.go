/*
Copyright Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"infini.sh/framework/core/util"
	"math/rand"
)

//- request:
//- method: GET
//url: /
//body:
//response:
//status: 200
//body:
//- request:
type Request struct {
	HasVariable bool `config:"has_variable"`
	Method string `config:"method"`
	Url    string `config:"url"`
	Body   string `config:"body"`
	Headers []map[string]string `config:"headers"`
	BasicAuth struct{
		Username string `config:"username"`
		Password string `config:"password"`
	} `config:"basic_auth"`
}

type Response struct {
	Status int    `config:"status"`
	Body   string `config:"body"`
	BodySize   int `config:"body_size"`
}

type Variable struct {
	Name   string `config:"name"`
	Path   string `config:"path"`
}

type LoadgenConfig struct {
	Variable []Variable `config:"variables"`
	Requests []Item `config:"requests"`
}

var dict= map[string][]string{}
func (config *LoadgenConfig)Init()  {
	for _,i:=range config.Variable{
		lines:=util.FileGetLines(i.Path)
		dict[util.TrimSpaces(i.Name)]=lines
	}
}

func (config *LoadgenConfig)GetVariable(key string)string  {
	d,ok:=dict[key]
	if ok{
		offset:=rand.Intn(len(d)-1)
		return d[offset]
	}
	return "not_found"
}

type Item struct {
	Request  Request  `config:"request"`
	Response Response `config:"response"`
}
