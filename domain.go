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
	"bytes"
	"encoding/base64"
	"math/rand"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	log "github.com/cihub/seelog"
	"github.com/valyala/fasttemplate"

	"infini.sh/framework/core/conditions"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
)

type valuesMap map[string]interface{}

func (m valuesMap) GetValue(key string) (interface{}, error) {
	v, ok := m[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

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
	// Disable fasthttp client's header names normalizing, preserve original header key, for requests
	DisableHeaderNamesNormalizing bool `config:"disable_header_names_normalizing"`

	RuntimeVariables         map[string]string `config:"runtime_variables"`
	RuntimeBodyLineVariables map[string]string `config:"runtime_body_line_variables"`

	urlHasTemplate  bool
	bodyHasTemplate bool

	headerTemplates map[string]*fasttemplate.Template
	urlTemplate     *fasttemplate.Template
	bodyTemplate    *fasttemplate.Template
}

func (req *Request) HasVariable() bool {
	return req.urlHasTemplate || req.bodyHasTemplate || len(req.headerTemplates) > 0
}

type ResponseAssert struct {
	Status   int    `config:"status"`
	Body     string `config:"body"`
	BodySize int    `config:"body_size"`
}

type Variable struct {
	Type   string   `config:"type"`
	Name   string   `config:"name"`
	Path   string   `config:"path"`
	Data   []string `config:"data"`
	Format string   `config:"format"`

	//type: range
	From uint64 `config:"from"`
	To   uint64 `config:"to"`

	Size int `config:"size"`

	//type: random_int_array
	RandomArrayKey          string `config:"variable_key"`
	RandomArrayType         string `config:"variable_type"`
	RandomSquareBracketChar bool   `config:"square_bracket"`
	RandomStringBracketChar string `config:"string_bracket"`
}

type AppConfig struct {
	// Access order: runtime_variables -> register -> variables
	Variable     []Variable    `config:"variables"`
	Requests     []RequestItem `config:"requests"`
	RunnerConfig RunnerConfig  `config:"runner_config"`
}

type RunnerConfig struct {
	// How many rounds of `requests` to run
	TotalRounds int `config:"total_rounds"`
	// Skip warming up round
	NoWarm bool `config:"no_warm"`
	// Exit(1) if any assert failed
	AssertInvalid bool `config:"assert_invalid"`
	// Exit(2) if any error occurred
	AssertError bool `config:"assert_error"`
	// Print the request sent to server
	LogRequests bool `config:"log_requests"`
	// Disable fasthttp client's header names normalizing, preserve original header key, for responses
	DisableHeaderNamesNormalizing bool `config:"disable_header_names_normalizing"`
}

var dict = map[string][]string{}
var variables = map[string]Variable{}

func (config *AppConfig) Init() {
	for _, i := range config.Variable {
		name := util.TrimSpaces(i.Name)
		var lines []string
		if len(i.Path) > 0 {
			lines = util.FileGetLines(i.Path)
			log.Debugf("path:%v, num of lines:%v", i.Path, len(lines))
			for i, v := range lines {
				v = strings.ReplaceAll(v, "\\", "\\\\")
				v = strings.ReplaceAll(v, "\"", "\\\"")
				lines[i] = v
			}
		}

		if len(i.Data) > 0 {
			for _, v := range i.Data {
				v = util.TrimSpaces(v)
				if len(v) > 0 {
					v = strings.ReplaceAll(v, "\\", "\\\\")
					v = strings.ReplaceAll(v, "\"", "\\\"")
					lines = append(lines, v)
				}
			}
		}

		dict[name] = lines

		variables[name] = i
	}

	var err error
	for _, v := range config.Requests {
		v.Request.headerTemplates = map[string]*fasttemplate.Template{}

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

		for _, headers := range v.Request.Headers {
			for headerK, headerV := range headers {
				if util.ContainStr(headerV, "$") {
					v.Request.headerTemplates[headerK], err = fasttemplate.NewTemplate(headerV, "$[[", "]]")
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}

}

// "2021-08-23T11:13:36.274"
const TsLayout = "2006-01-02T15:04:05.000"

func GetVariable(runtimeKV util.MapStr, key string) string {

	if runtimeKV != nil {
		x, err := runtimeKV.GetValue(key)
		if err == nil {
			return util.ToString(x)
		}
	}

	x, ok := variables[key]
	if ok {
		switch x.Type {
		case "sequence":
			return util.ToString(util.GetAutoIncrement32ID(x.Name, uint32(x.From), uint32(x.To)).Increment())
		case "sequence64":
			return util.ToString(util.GetAutoIncrement64ID(x.Name, x.From, x.To).Increment64())
		case "uuid":
			return util.GetUUID()
		case "now_local":
			return time.Now().Local().String()
		case "now_with_format":
			if x.Format == "" {
				panic(errors.Errorf("date format is not set, [%v]", x))
			}
			return time.Now().Format(x.Format)
		case "now_utc":
			return time.Now().UTC().String()
		case "now_utc_lite":
			return time.Now().UTC().Format(TsLayout)
		case "now_unix":
			return util.IntToString(int(time.Now().Local().Unix()))
		case "int_array_bitmap":
			rb3 := roaring.New()
			if x.Size > 0 {
				for y := 0; y < x.Size; y++ {
					v := rand.Intn(int(x.To-x.From+1)) + int(x.From)
					rb3.Add(uint32(v))
				}
			}
			buf := new(bytes.Buffer)
			rb3.WriteTo(buf)
			return base64.StdEncoding.EncodeToString(buf.Bytes())
		case "range":
			return util.IntToString(rand.Intn(int(x.To-x.From+1)) + int(x.From))
		case "random_array":
			str := bytes.Buffer{}

			if x.RandomSquareBracketChar {
				str.WriteString("[")
			}

			if x.RandomArrayKey != "" {
				if x.Size > 0 {
					for y := 0; y < x.Size; y++ {
						if x.RandomSquareBracketChar && str.Len() > 1 || (!x.RandomSquareBracketChar && str.Len() > 0) {
							str.WriteString(",")
						}

						v := GetVariable(runtimeKV, x.RandomArrayKey)

						//left "
						if x.RandomArrayType == "string" {
							if x.RandomStringBracketChar != "" {
								str.WriteString(x.RandomStringBracketChar)
							} else {
								str.WriteString("\"")
							}
						}

						str.WriteString(v)

						// right "
						if x.RandomArrayType == "string" {
							if x.RandomStringBracketChar != "" {
								str.WriteString(x.RandomStringBracketChar)
							} else {
								str.WriteString("\"")
							}
						}
					}
				}
			}

			if x.RandomSquareBracketChar {
				str.WriteString("]")
			}
			return str.String()
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

type RequestItem struct {
	Request        *Request        `config:"request"`
	ResponseAssert *ResponseAssert `config:"response"`
	// TODO: mask invalid gateway fields
	Assert *conditions.Config `config:"assert"`
	// Populate global context with `_ctx` values
	Register []map[string]string `config:"register"`
}

type RequestResult struct {
	RequestSize  int
	ResponseSize int
	Status       int
	Error        bool
	Valid        bool
	Duration     time.Duration
}

func (result *RequestResult) Reset() {
	result.Error = false
	result.Status = 0
	result.RequestSize = 0
	result.ResponseSize = 0
	result.Valid = false
	result.Duration = 0
}
