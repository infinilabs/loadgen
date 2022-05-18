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
	"github.com/valyala/fasttemplate"
	"math/rand"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

type Request struct {
	Method           string              `config:"method"`
	Url              string              `config:"url"`
	Body             string              `config:"body"`
	RepeatBodyNTimes int                 `config:"body_repeat_times"`
	Headers          []map[string]string `config:"headers"`
	BasicAuth        struct {
		Username string `config:"username"`
		Password string `config:"password"`
	} `config:"basic_auth"`

	RuntimeVariables         []map[string]string `config:"runtime_variables"`
	RuntimeBodyLineVariables []map[string]string `config:"runtime_body_line_variables"`

	urlHasTemplate  bool
	bodyHasTemplate bool

	urlTemplate  *fasttemplate.Template
	bodyTemplate *fasttemplate.Template
}

func (req *Request) HasVariable() bool {
	return req.urlHasTemplate || req.bodyHasTemplate
}

type ResponseAssert struct {
	Status   int    `config:"status"`
	Body     string `config:"body"`
	BodySize int    `config:"body_size"`
}

type Variable struct {
	Type string `config:"type"`
	Name string `config:"name"`
	Path string `config:"path"`

	//type: range
	From int `config:"from"`
	To   int `config:"to"`
}

type AppConfig struct {
	Variable []Variable    `config:"variables"`
	Requests []RequestItem `config:"requests"`
}

var dict= map[string][]string{}
var variables= map[string]Variable{}

func (config *AppConfig) Init() {
	for _, i := range config.Variable {
		name := util.TrimSpaces(i.Name)
		if len(i.Path) > 0 {
			lines := util.FileGetLines(i.Path)
			for i, v := range lines {
				v = strings.ReplaceAll(v, "\\", "\\\\")
				v = strings.ReplaceAll(v, "\"", "\\\"")
				lines[i] = v
			}
			dict[name] = lines
		}
		variables[name] = i
	}

	var err error
	for _, v := range config.Requests {

		if util.ContainStr(v.Request.Url, "$[[") {
			v.Request.urlHasTemplate = true
			v.Request.urlTemplate, err = fasttemplate.NewTemplate(v.Request.Url, "$[[", "]]")
			if err != nil {
				panic(err)
			}
		}

		if v.Request.RepeatBodyNTimes <= 0 && len(v.Request.Body) > 0 {
			v.Request.RepeatBodyNTimes = 1
		}

		if util.ContainStr(v.Request.Body, "$") {
			v.Request.bodyHasTemplate = true
			v.Request.bodyTemplate, err = fasttemplate.NewTemplate(v.Request.Body, "$[[", "]]")
			if err != nil {
				panic(err)
			}
		}
	}

}

//"2021-08-23T11:13:36.274"
const TsLayout = "2006-01-02T15:04:05.000"

func GetVariable(runtimeKV map[string]string, key string) string {

	if runtimeKV != nil {
		x, ok := runtimeKV[key]
		if ok {
			return x
		}
	}

	x, ok := variables[key]
	if ok {
		switch x.Type {
		case "sequence":
			return util.IntToString(int(util.GetIncrementID(x.Name)))
		case "uuid":
			return util.GetUUID()
		case "now_local":
			return time.Now().Local().String()
		case "now_utc":
			return time.Now().UTC().String()
		case "now_utc_lite":
			return time.Now().UTC().Format(TsLayout)
		case "now_unix":
			return util.IntToString(int(time.Now().Local().Unix()))
		case "range":
			return util.IntToString(rand.Intn(x.To-x.From+1) + x.From)
		}
	}

	d, ok := dict[key]
	if ok {

		if len(d) == 1 {
			return d[0]
		}

		offset := rand.Intn(len(d))
		return d[offset]
	}
	return "not_found"
}

func (config *AppConfig)ReplaceVariable(runtimeKV map[string]string,v string) string {
	matchs :=regex.FindAllString(v,-1)
	for _,v1:=range matchs {
		old :=v1
		v1=util.TrimLeftStr(v1,"$[[")
		v1=util.TrimRightStr(v1,"]]")
		variable := GetVariable(runtimeKV, v1)
		v=strings.ReplaceAll(v, old,fmt.Sprintf("%s",util.TrimSpaces(variable)))
	}
	if global.Env().IsDebug{
		log.Trace("replaced:",v)
	}
	return v
}

type RequestItem struct {
	Request        *Request        `config:"request"`
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

func (result *RequestResult) Reset()  {
	result.Error=false
	result.Status=0
	result.RequestSize=0
	result.ResponseSize=0
	result.Valid=false
	result.Duration=0
}
