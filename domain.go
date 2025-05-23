// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Loadgen is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"infini.sh/framework/core/model"
	"math/rand"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	log "github.com/cihub/seelog"
	"infini.sh/framework/lib/fasttemplate"

	"infini.sh/framework/core/conditions"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
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
	Method     string `config:"method"`
	Url        string `config:"url"`
	Body       string `config:"body"`
	SimpleMode bool   `config:"simple_mode"`

	RepeatBodyNTimes int                 `config:"body_repeat_times"`
	Headers          []map[string]string `config:"headers"`
	BasicAuth        *model.BasicAuth    `config:"basic_auth"`

	// Disable fasthttp client's header names normalizing, preserve original header key, for requests
	DisableHeaderNamesNormalizing bool `config:"disable_header_names_normalizing"`

	RuntimeVariables         map[string]string `config:"runtime_variables"`
	RuntimeBodyLineVariables map[string]string `config:"runtime_body_line_variables"`

	ExecuteRepeatTimes int `config:"execute_repeat_times"`

	urlHasTemplate    bool
	headerHasTemplate bool
	bodyHasTemplate   bool

	headerTemplates map[string]*fasttemplate.Template
	urlTemplate     *fasttemplate.Template
	bodyTemplate    *fasttemplate.Template
}

func (req *Request) HasVariable() bool {
	return req.urlHasTemplate || req.bodyHasTemplate || len(req.headerTemplates) > 0
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

	Replace map[string]string `config:"replace"`

	Size int `config:"size"`

	//type: random_int_array
	RandomArrayKey          string `config:"variable_key"`
	RandomArrayType         string `config:"variable_type"`
	RandomSquareBracketChar bool   `config:"square_bracket"`
	RandomStringBracketChar string `config:"string_bracket"`

	replacer *strings.Replacer
}

type AppConfig struct {
	Environments map[string]string `config:"env"`
	Tests        []Test            `config:"tests"`
	LoaderConfig
}

type LoaderConfig struct {
	// Access order: runtime_variables -> register -> variables
	Variable     []Variable    `config:"variables"`
	Requests     []RequestItem `config:"requests"`
	RunnerConfig RunnerConfig  `config:"runner"`
}

type RunnerConfig struct {
	// How many rounds of `requests` to run
	TotalRounds int `config:"total_rounds"`
	// Skip warming up round
	NoWarm bool `config:"no_warm"`

	ValidStatusCodesDuringWarmup []int `config:"valid_status_codes_during_warmup"`

	// Exit(1) if any assert failed
	AssertInvalid bool `config:"assert_invalid"`

	ContinueOnAssertInvalid bool `config:"continue_on_assert_invalid"`
	SkipInvalidAssert       bool `config:"skip_invalid_assert"`

	// Exit(2) if any error occurred
	AssertError bool `config:"assert_error"`
	// Print the request sent to server
	LogRequests bool `config:"log_requests"`

	BenchmarkOnly    bool `config:"benchmark_only"`
	DurationInUs     bool `config:"duration_in_us"`
	NoStats          bool `config:"no_stats"`
	NoSizeStats      bool `config:"no_size_stats"`
	MetricSampleSize int  `config:"metric_sample_size"`

	// Print the request sent to server if status code matched
	LogStatusCodes []int `config:"log_status_codes"`
	// Disable fasthttp client's header names normalizing, preserve original header key, for responses
	DisableHeaderNamesNormalizing bool `config:"disable_header_names_normalizing"`

	// Whether to reset the context, including variables, runtime KV pairs, etc.,
	// before this test run.
	ResetContext bool `config:"reset_context"`

	// Default endpoint if not specified in a request
	DefaultEndpoint  string           `config:"default_endpoint"`
	DefaultBasicAuth *model.BasicAuth `config:"default_basic_auth"`
	defaultEndpoint  *fasthttp.URI
}

func (config *RunnerConfig) parseDefaultEndpoint() (*fasthttp.URI, error) {
	if config.defaultEndpoint != nil {
		return config.defaultEndpoint, nil
	}

	if config.DefaultEndpoint != "" {
		uri := &fasthttp.URI{}
		err := uri.Parse(nil, []byte(config.DefaultEndpoint))
		if err != nil {
			return nil, err
		}
		config.defaultEndpoint = uri
		return config.defaultEndpoint, err
	}

	return config.defaultEndpoint, errors.New("no valid default endpoint")
}

/*
A test case is a standalone directory containing the following configs:
- gateway.yml: The configuration to start the gateway server
- loadgen.yml: The configuration to define the test cases
*/
type Test struct {
	// The directory of the test configurations
	Path string `config:"path"`
	// Whether to use --compress for loadgen
	Compress bool `config:"compress"`
}

const (
	// Gateway-related configurations
	env_LR_GATEWAY_CMD      = "LR_GATEWAY_CMD"
	env_LR_GATEWAY_HOST     = "LR_GATEWAY_HOST"
	env_LR_GATEWAY_API_HOST = "LR_GATEWAY_API_HOST"
)

var (
	dict      = map[string][]string{}
	variables map[string]Variable
)

func (config *AppConfig) Init() {

}

func (config *AppConfig) testEnv(envVars ...string) bool {
	for _, envVar := range envVars {
		if v, ok := config.Environments[envVar]; !ok || v == "" {
			return false
		}
	}
	return true
}

func (config *LoaderConfig) Init() error {
	// As we do not allow duplicate variable definitions, it is necessary to clear
	// any previously defined variables.
	variables = map[string]Variable{}
	if config.RunnerConfig.ResetContext {
		dict = map[string][]string{}
		util.ClearAllID()
	}

	for _, i := range config.Variable {
		i.Name = util.TrimSpaces(i.Name)
		_, ok := variables[i.Name]
		if ok {
			return fmt.Errorf("variable [%s] defined twice", i.Name)
		}
		var lines []string
		if len(i.Path) > 0 {
			lines = util.FileGetLines(i.Path)
			log.Debugf("path:%v, num of lines:%v", i.Path, len(lines))
		}

		if len(i.Data) > 0 {
			for _, v := range i.Data {
				v = util.TrimSpaces(v)
				if len(v) > 0 {
					lines = append(lines, v)
				}
			}
		}

		if len(i.Replace) > 0 {
			var replaces []string

			for k, v := range i.Replace {
				replaces = append(replaces, k, v)
			}
			i.replacer = strings.NewReplacer(replaces...)
		}

		dict[i.Name] = lines

		variables[i.Name] = i
	}

	var err error
	for _, v := range config.Requests {
		if v.Request == nil {
			continue
		}
		v.Request.headerTemplates = map[string]*fasttemplate.Template{}
		if util.ContainStr(v.Request.Url, "$[[") {
			v.Request.urlHasTemplate = true
			v.Request.urlTemplate, err = fasttemplate.NewTemplate(v.Request.Url, "$[[", "]]")
			if err != nil {
				return err
			}
		}

		if v.Request.RepeatBodyNTimes <= 0 && len(v.Request.Body) > 0 {
			v.Request.RepeatBodyNTimes = 1
		}

		if util.ContainStr(v.Request.Body, "$") {
			v.Request.bodyHasTemplate = true
			v.Request.bodyTemplate, err = fasttemplate.NewTemplate(v.Request.Body, "$[[", "]]")
			if err != nil {
				return err
			}
		}

		for _, headers := range v.Request.Headers {
			for headerK, headerV := range headers {
				if util.ContainStr(headerV, "$") {
					v.Request.headerHasTemplate = true
					v.Request.headerTemplates[headerK], err = fasttemplate.NewTemplate(headerV, "$[[", "]]")
					if err != nil {
						return err
					}
				}
			}
		}

		////if there is no $[[ in the request, then we can assume that the request is in simple mode
		//if !v.Request.urlHasTemplate && !v.Request.bodyHasTemplate&& !v.Request.headerHasTemplate {
		//	v.Request.SimpleMode = true
		//}
	}

	return nil
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

	return getVariable(key)
}

func getVariable(key string) string {
	x, ok := variables[key]
	if !ok {
		return "not_found"
	}

	rawValue := buildVariableValue(x)
	if x.replacer == nil {
		return rawValue
	}
	return x.replacer.Replace(rawValue)
}

func buildVariableValue(x Variable) string {
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
	case "now_unix_in_ms":
		return util.IntToString(int(time.Now().Local().UnixMilli()))
	case "now_unix_in_micro":
		return util.IntToString(int(time.Now().Local().UnixMicro()))
	case "now_unix_in_nano":
		return util.IntToString(int(time.Now().Local().UnixNano()))
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

					v := getVariable(x.RandomArrayKey)

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
	case "file", "list":
		d, ok := dict[x.Name]
		if ok {

			if len(d) == 1 {
				return d[0]
			}
			offset := rand.Intn(len(d))
			return d[offset]
		}
	}
	return "invalid_variable_type"
}

type RequestItem struct {
	Request *Request `config:"request"`
	// TODO: mask invalid gateway fields
	Assert    *conditions.Config `config:"assert"`
	AssertDsl string             `config:"assert_dsl"`
	Sleep     *SleepAction       `config:"sleep"`
	// Populate global context with `_ctx` values
	Register []map[string]string `config:"register"`
}

type SleepAction struct {
	SleepInMilliSeconds int64 `config:"sleep_in_milli_seconds"`
}

type RequestResult struct {
	RequestCount int
	RequestSize  int
	ResponseSize int
	Status       int
	Error        bool
	Invalid      bool
	Duration     time.Duration
}

func (result *RequestResult) Reset() {
	result.Error = false
	result.Status = 0
	result.RequestCount = 0
	result.RequestSize = 0
	result.ResponseSize = 0
	result.Invalid = false
	result.Duration = 0
}
