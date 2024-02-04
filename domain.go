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
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	log "github.com/cihub/seelog"
	"github.com/valyala/fasttemplate"

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

	defaultEndpoint *fasthttp.URI

	urlHasTemplate  bool
	bodyHasTemplate bool

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
	/*
		Required environments:
		- LR_ELASTICSEARCH_ENDPOINT // ES server endpoint
		- LR_GATEWAY_HOST // Gateway server host
		- LR_GATEWAY_CMD // The command to start gateway server
		Optional environments:
		- LR_TEST_DIR    // The root directory of all test cases, will automatically convert to absolute path. Default: ./testing
		- LR_GATEWAY_API_HOST // Gateway server api binding host
		- LR_MINIO_API_HOST // minio server host
		- LR_MINIO_API_USERNAME // minio server username
		- LR_MINIO_API_PASSWORD // minio server password
		- LR_MINIO_TEST_BUCKET // minio testing bucket, need to set as public access
		- LR_GATEWAY_FLOATING_IP_HOST // Gateway server floating IP host
	*/
	Environments map[string]string `config:"env"`
	Tests        []Test            `config:"tests"`
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
	// Exit(1) if any assert failed
	AssertInvalid bool `config:"assert_invalid"`
	// Exit(2) if any error occurred
	AssertError bool `config:"assert_error"`
	// Print the request sent to server
	LogRequests bool `config:"log_requests"`
	// Print the request sent to server if status code matched
	LogStatusCodes []int `config:"log_status_codes"`
	// Disable fasthttp client's header names normalizing, preserve original header key, for responses
	DisableHeaderNamesNormalizing bool `config:"disable_header_names_normalizing"`
	// Default endpoint if not specified in a request
	DefaultEndpoint string `config:"default_endpoint"`
	// Whether to reset the context, including variables, runtime KV pairs, etc.,
	// before this test run.
	ResetContext bool `config:"reset_context"`
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
	// Required configurations
	env_LR_TEST_DIR = "LR_TEST_DIR"

	// Gateway-related configurations
	env_LR_GATEWAY_CMD      = "LR_GATEWAY_CMD"
	env_LR_GATEWAY_HOST     = "LR_GATEWAY_HOST"
	env_LR_GATEWAY_API_HOST = "LR_GATEWAY_API_HOST"

	// Standard environments used by `testing` repo
	env_LR_ELASTICSEARCH_ENDPOINT   = "LR_ELASTICSEARCH_ENDPOINT"
	env_LR_MINIO_API_HOST           = "LR_MINIO_API_HOST"
	env_LR_MINIO_API_USERNAME       = "LR_MINIO_API_USERNAME"
	env_LR_MINIO_API_PASSWORD       = "LR_MINIO_API_PASSWORD"
	env_LR_GATEWAY_FLOATING_IP_HOST = "LR_GATEWAY_FLOATING_IP_HOST"
	env_LR_MINIO_TEST_BUCKET        = "LR_MINIO_TEST_BUCKET"
)

var (
	dict      = map[string][]string{}
	variables map[string]Variable
)

func (config *AppConfig) Init() {
	fullTestDir, err := filepath.Abs(config.Environments[env_LR_TEST_DIR])
	if err != nil {
		log.Warnf("failed to get the abs path of test_dir, error: %+v", err)
	} else {
		config.Environments[env_LR_TEST_DIR] = fullTestDir
	}
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

	defaultEndpoint := fasthttp.URI{}
	defaultEndpoint.Parse(nil, []byte(config.RunnerConfig.DefaultEndpoint))

	var err error
	for _, v := range config.Requests {
		if v.Request == nil {
			continue
		}
		v.Request.defaultEndpoint = &defaultEndpoint
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
					v.Request.headerTemplates[headerK], err = fasttemplate.NewTemplate(headerV, "$[[", "]]")
					if err != nil {
						return err
					}
				}
			}
		}
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
	result.RequestSize = 0
	result.ResponseSize = 0
	result.Invalid = false
	result.Duration = 0
}
