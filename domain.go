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
	"fmt"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"math/rand"
	log "src/github.com/cihub/seelog"
	"strings"
	"time"
)

type Request struct {
	HasVariable bool `config:"has_variable"`
	Method string `config:"method"`
	Url    string `config:"url"`
	Body   string `config:"body"`
	RepeatBodyNTimes   int `config:"body_repeat_times"`
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

type AppConfig struct {
	Variable []Variable    `config:"variables"`
	Requests []RequestItem `config:"requests"`
}

var dict= map[string][]string{}

func (config *AppConfig)Init()  {
	for _,i:=range config.Variable{
		lines:=util.FileGetLines(i.Path)
		dict[util.TrimSpaces(i.Name)]=lines
	}
}

func (config *AppConfig)GetVariable(key string)string  {
	d,ok:=dict[key]
	if ok{
		offset:=rand.Intn(len(d)-1)
		return d[offset]
	}
	return "not_found"
}

func (config *AppConfig)ReplaceVariable(v string) string {
	matchs :=regex.FindAllString(v,-1)
	for _,v1:=range matchs {
		old :=v1
		v1=util.TrimLeftStr(v1,"$[[")
		v1=util.TrimRightStr(v1,"]]")
		variable:=config.GetVariable(v1)
		v=strings.ReplaceAll(v, old,fmt.Sprintf("%s",util.TrimSpaces(variable)))
	}
	if global.Env().IsDebug{
		log.Trace("replaced:",v)
	}
	return v
}

type RequestItem struct {
	Request  Request  `config:"request"`
	Response *Response `config:"response"`
}

type RequestResult struct {
	RequestSize int
	ResponseSize int
	Status int
	Error bool
	Valid bool
	Duration time.Duration
}
