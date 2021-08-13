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
	"math/rand"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
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

type ResponseAssert struct {
	Status int    `config:"status"`
	Body   string `config:"body"`
	BodySize   int `config:"body_size"`
}

type Variable struct {
	Type   string `config:"type"`
	Name   string `config:"name"`
	Path   string `config:"path"`
}

type AppConfig struct {
	Variable []Variable    `config:"variables"`
	Requests []RequestItem `config:"requests"`
}

var dict= map[string][]string{}
var variables= map[string]Variable{}

func (config *AppConfig)Init()  {
	for _,i:=range config.Variable{
		name:=util.TrimSpaces(i.Name)
		if len(i.Path)>0{
			lines:=util.FileGetLines(i.Path)
			dict[name]=lines
		}
		variables[name]=i
	}
}

func (config *AppConfig)GetVariable(key string)string  {
	 x,ok:=variables[key]
	 if ok{
	 	if x.Type=="sequence"{
	 		return util.IntToString(int(util.GetIncrementID(x.Name)))
		}

	 	if x.Type=="uuid"{
	 		return util.GetUUID()
		}

	 	if x.Type=="now_local"{
	 		return time.Now().Local().String()
		}

	 	if x.Type=="now_utc"{
	 		return time.Now().UTC().String()
		}

	 	if x.Type=="now_unix"{
	 		return util.IntToString(int(time.Now().Local().Unix()))
		}
	 }

	 

	d,ok:=dict[key]
	if ok{

		if len(d)==1 {
		 return d[0]
		}

		offset:=rand.Intn(len(d))
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
	Request  Request         `config:"request"`
	ResponseAssert *ResponseAssert `config:"response"`
}

type RequestResult struct {
	RequestSize int
	ResponseSize int
	Status int
	Error bool
	Valid bool
	Duration time.Duration
}
