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
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"github.com/jamiealquiza/tachymeter"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/conditions"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type LoadGenerator struct {
	duration        int //seconds
	goroutines      int
	statsAggregator chan *LoadStats
	interrupted     int32
}

type LoadStats struct {
	TotReqSize     int64
	TotRespSize    int64
	TotDuration    time.Duration
	MinRequestTime time.Duration
	MaxRequestTime time.Duration
	NumRequests    int
	NumErrs        int
	NumInvalid     int
	StatusCode     map[int]int
}

var (
	httpClient fasthttp.Client
	resultPool = &sync.Pool{
		New: func() interface{} {
			return &RequestResult{}
		},
	}
)

func NewLoadGenerator(duration int, goroutines int, statsAggregator chan *LoadStats, disableHeaderNamesNormalizing bool) (rt *LoadGenerator) {
	if readTimeout <= 0 {
		readTimeout = timeout
	}
	if writeTimeout <= 0 {
		writeTimeout = timeout
	}
	if dialTimeout <= 0 {
		dialTimeout = timeout
	}

	httpClient = fasthttp.Client{
		MaxConnsPerHost:               goroutines,
		//MaxConns: goroutines,
		NoDefaultUserAgentHeader:      false,
		DisableHeaderNamesNormalizing: disableHeaderNamesNormalizing,
		Name:      global.Env().GetAppLowercaseName() + "/" + global.Env().GetVersion() + "/" + global.Env().GetBuildNumber(),
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if readTimeout>0{
		httpClient.ReadTimeout= time.Second * time.Duration(readTimeout)
	}
	if writeTimeout>0{
		httpClient.WriteTimeout= time.Second * time.Duration(writeTimeout)
	}
	if dialTimeout>0{
		httpClient.Dial=func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Duration(dialTimeout)*time.Second)
		}
	}

	rt = &LoadGenerator{duration, goroutines, statsAggregator, 0}
	return
}

var defaultHTTPPool = fasthttp.NewRequestResponsePool("default_http")



func doRequest(config *LoaderConfig,globalCtx util.MapStr,req *fasthttp.Request,resp *fasthttp.Response, item *RequestItem, loadStats *LoadStats,timer *tachymeter.Tachymeter) (continueNext bool,err error) {

	if item.Request != nil {


		if item.Request.ExecuteRepeatTimes<1{
			item.Request.ExecuteRepeatTimes=1
		}

		for i:=0;i<item.Request.ExecuteRepeatTimes;i++{
			resp.Reset()
			resp.ResetBody()
			start := time.Now()

			if global.Env().IsDebug{
				log.Info(req.String())
			}

			if timeout>0{
				err = httpClient.DoTimeout(req, resp, time.Duration(timeout)*time.Second)
			}else{
				err = httpClient.Do(req, resp)
			}

			if global.Env().IsDebug{
				log.Info(resp.String())
			}

			duration:=time.Since(start)
			statsCode:=resp.StatusCode()


			if !config.RunnerConfig.BenchmarkOnly && timer!=nil{
				timer.AddTime(duration)
			}

			if !config.RunnerConfig.NoStats {
				if config.RunnerConfig.DurationInUs {
					stats.Timing("request", "duration_in_us", duration.Microseconds())
				} else {
					stats.Timing("request", "duration", duration.Milliseconds())
				}

				stats.Increment("request", "total")
				stats.Increment("request", strconv.Itoa(resp.StatusCode()))

				if err!=nil {
					loadStats.NumErrs++
					loadStats.NumInvalid ++
				}

				if !config.RunnerConfig.NoSizeStats{
					loadStats.TotReqSize += int64(req.GetRequestLength()) //TODO inaccurate
					loadStats.TotRespSize += int64(resp.GetResponseLength()) //TODO inaccurate
				}

				loadStats.NumRequests ++
				loadStats.TotDuration+=duration
				loadStats.MaxRequestTime = util.MaxDuration(duration, loadStats.MaxRequestTime)
				loadStats.MinRequestTime = util.MinDuration(duration, loadStats.MinRequestTime)
				loadStats.StatusCode[statsCode]+=1
			}

			if config.RunnerConfig.BenchmarkOnly {
				return  true,err
			}

			if item.Register!=nil || item.Assert!=nil ||config.RunnerConfig.LogRequests{
				//only use last request and response
				reqBody := req.GetRawBody()
				respBody := resp.GetRawBody()
				if global.Env().IsDebug{
					log.Debugf("final response code: %v, body: %s", resp.StatusCode(),string(respBody))
				}

				if item.Request != nil && config.RunnerConfig.LogRequests || util.ContainsInAnyInt32Array(statsCode, config.RunnerConfig.LogStatusCodes) {
					log.Infof("[%v] %v, %v - %v", item.Request.Method, item.Request.Url, item.Request.Headers, util.SubString(string(reqBody), 0, 512))
					log.Infof("status: %v, error: %v, response: %v", statsCode, err, util.SubString(string(respBody), 0, 512))
				}

				if err!=nil{
					continue
				}

				event := buildCtx(resp, respBody, duration)
				if item.Register != nil {
					log.Debugf("registering %+v, event: %+v", item.Register, event)
					for _, item := range item.Register {
						for dest, src := range item {
							val, valErr := event.GetValue(src)
							if valErr != nil {
								log.Errorf("failed to get value with key: %s", src)
							}
							log.Debugf("put globalCtx %+v, %+v", dest, val)
							globalCtx.Put(dest, val)
						}
					}
				}

				if item.Assert != nil {
					// Dump globalCtx into assert event
					event.Update(globalCtx)
					if len(respBody) < 4096 {
						log.Debugf("assert _ctx: %+v", event)
					}
					condition, buildErr := conditions.NewCondition(item.Assert)
					if buildErr != nil {
						log.Errorf("failed to build conditions whilte assert existed, error: %+v", buildErr)
						loadStats.NumInvalid ++
						return
					}
					if !condition.Check(event) {
						loadStats.NumInvalid ++
						if item.Request != nil {
							log.Errorf("%s %s, assertion failed, skipping subsequent requests", item.Request.Method, item.Request.Url)
						}

						if !config.RunnerConfig.ContinueOnAssertInvalid{
							return false, err
						}
					}
				}
			}

			if item.Sleep != nil {
				time.Sleep(time.Duration(item.Sleep.SleepInMilliSeconds) * time.Millisecond)
			}
		}
	}

	return
}

func buildCtx(resp *fasthttp.Response, respBody []byte, duration time.Duration) util.MapStr {
	var statusCode int
	header := map[string]interface{}{}
	if resp != nil {
		resp.Header.VisitAll(func(k, v []byte) {
			header[string(k)] = string(v)
		})
		statusCode = resp.StatusCode()
	}
	event := util.MapStr{
		"_ctx": map[string]interface{}{
			"response": map[string]interface{}{
				"status":      statusCode,
				"header":      header,
				"body":        string(respBody),
				"body_length": len(respBody),
			},
			"elapsed": int64(duration / time.Millisecond),
		},
	}
	bodyJson := map[string]interface{}{}
	jsonErr := json.Unmarshal(respBody, &bodyJson)
	if jsonErr == nil {
		event.Put("_ctx.response.body_json", bodyJson)
	}
	return event
}

func (cfg *LoadGenerator) Run(config *LoaderConfig, countLimit int,timer *tachymeter.Tachymeter) {
	loadStats := &LoadStats{MinRequestTime: time.Millisecond, StatusCode: map[int]int{}}
	start := time.Now()

	limiter := rate.GetRateLimiter("loadgen", "requests", int(rateLimit), 1, time.Second*1)

	// TODO: support concurrent access
	globalCtx := util.MapStr{}
	req := defaultHTTPPool.AcquireRequest()
	defer defaultHTTPPool.ReleaseRequest(req)
	resp := defaultHTTPPool.AcquireResponse()
	defer defaultHTTPPool.ReleaseResponse(resp)


	totalRequests := 0
	totalRounds := 0

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {
		if config.RunnerConfig.TotalRounds > 0 && totalRounds >= config.RunnerConfig.TotalRounds {
			goto END
		}
		totalRounds += 1

		for _, item := range config.Requests {

			if !config.RunnerConfig.BenchmarkOnly{
				if countLimit > 0 && totalRequests >= countLimit {
					goto END
				}
				totalRequests += 1

				if rateLimit > 0 {
				RetryRateLimit:
					if !limiter.Allow() {
						time.Sleep(10 * time.Millisecond)
						goto RetryRateLimit
					}
				}
			}



			item.prepareRequest(config,globalCtx, req)

			next,_:=doRequest(config,globalCtx, req,resp,&item, loadStats,timer)
			if !next{
				break
			}

		}
	}

END:
	cfg.statsAggregator <- loadStats
}

func (v *RequestItem) prepareRequest(config *LoaderConfig, globalCtx util.MapStr, req *fasthttp.Request) {
	//cleanup
	req.Reset()
	req.ResetBody()

	if v.Request.BasicAuth!=nil && v.Request.BasicAuth.Username != "" {
		req.SetBasicAuth(v.Request.BasicAuth.Username, v.Request.BasicAuth.Password)
	}else{
		//try use default auth
		if config.RunnerConfig.DefaultBasicAuth!=nil&&config.RunnerConfig.DefaultBasicAuth.Username!=""{
			req.SetBasicAuth(config.RunnerConfig.DefaultBasicAuth.Username, config.RunnerConfig.DefaultBasicAuth.Password)
		}
	}

	if v.Request.SimpleMode{
		req.Header.SetMethod(v.Request.Method)
		req.SetRequestURI(v.Request.Url)
		return
	}

	bodyBuffer := req.BodyBuffer()
	var bodyWriter io.Writer = bodyBuffer
	if v.Request.DisableHeaderNamesNormalizing {
		req.Header.DisableNormalizing()
	}

	if compress {
		var err error
		gzipWriter, err := gzip.NewWriterLevel(bodyBuffer, fasthttp.CompressBestCompression)
		if err != nil {
			panic("failed to create gzip writer")
		}
		defer gzipWriter.Close()
		bodyWriter = gzipWriter
	}

	//init runtime variables
	// TODO: optimize overall variable populate flow
	runtimeVariables := util.MapStr{}
	runtimeVariables.Update(globalCtx)

	if v.Request.HasVariable() {
		if len(v.Request.RuntimeVariables) > 0 {
			for k, v := range v.Request.RuntimeVariables {
				runtimeVariables.Put(k, GetVariable(runtimeVariables, v))
			}
		}


	}

	//prepare url
	url := v.Request.Url
	if v.Request.urlHasTemplate {
		url = v.Request.urlTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			variable := GetVariable(runtimeVariables, tag)
			return w.Write(util.UnsafeStringToBytes(variable))
		})
	}

	//set default endpoint
	parsedUrl := fasthttp.URI{}
	err:=parsedUrl.Parse(nil, []byte(url))
	if err!=nil{
		panic(err)
	}
	if parsedUrl.Host() == nil || len(parsedUrl.Host()) == 0 {
		path,err:=config.RunnerConfig.parseDefaultEndpoint()
		//log.Infof("default endpoint: %v, %v",path,err)
		if err==nil{
			parsedUrl.SetSchemeBytes(path.Scheme())
			parsedUrl.SetHostBytes(path.Host())
		}
	}
	url = parsedUrl.String()

	req.SetRequestURI(url)

	if global.Env().IsDebug{
		log.Debugf("final request url: %v %s", v.Request.Method,url)
	}

	//prepare method
	req.Header.SetMethod(v.Request.Method)

	if len(v.Request.Headers) > 0 {
		for _, headers := range v.Request.Headers {
			for headerK, headerV := range headers {
				if tmpl, ok := v.Request.headerTemplates[headerK]; ok {
					headerV = tmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
						variable := GetVariable(runtimeVariables, tag)
						return w.Write(util.UnsafeStringToBytes(variable))
					})
				}
				req.Header.Set(headerK, headerV)
			}
		}
	}
	if global.Env().IsDebug {
		log.Debugf("final request headers: %s", req.Header.String())
	}

	//req.Header.Set("User-Agent", UserAgent)

	//prepare request body
	for i := 0; i < v.Request.RepeatBodyNTimes; i++ {
		body := v.Request.Body
		if len(body) > 0 {
			if v.Request.bodyHasTemplate {
				if len(v.Request.RuntimeBodyLineVariables) > 0 {
					for k, v := range v.Request.RuntimeBodyLineVariables {
						runtimeVariables[k] = GetVariable(runtimeVariables, v)
					}
				}

				v.Request.bodyTemplate.ExecuteFuncStringExtend(bodyWriter, func(w io.Writer, tag string) (int, error) {
					variable := GetVariable(runtimeVariables, tag)
					return w.Write([]byte(variable))
				})
			} else {
				bodyWriter.Write(util.UnsafeStringToBytes(body))
			}
		}
	}

	req.Header.Set("X-PayLoad-Size", util.ToString(bodyBuffer.Len()))

	if bodyBuffer.Len() > 0 && compress {
		req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
		req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
		req.Header.Set("X-PayLoad-Compressed", util.ToString(true))
	}
}

func (cfg *LoadGenerator) Warmup(config *LoaderConfig) int {
	log.Info("warmup started")
	loadStats := &LoadStats{MinRequestTime: time.Millisecond, StatusCode: map[int]int{}}
	req := defaultHTTPPool.AcquireRequest()
	defer defaultHTTPPool.ReleaseRequest(req)
	resp := defaultHTTPPool.AcquireResponse()
	defer defaultHTTPPool.ReleaseResponse(resp)
	globalCtx := util.MapStr{}
	for _, v := range config.Requests {
		v.prepareRequest(config,globalCtx, req)
		next,err:=doRequest(config,globalCtx,req,resp, &v, loadStats,nil)
		for k,_:=range loadStats.StatusCode{
			if len(config.RunnerConfig.ValidStatusCodesDuringWarmup)>0{
				if util.ContainsInAnyInt32Array(k,config.RunnerConfig.ValidStatusCodesDuringWarmup){
					continue
				}
			}
			if k >= 400 || k == 0 || err != nil {
				log.Infof("requests seems failed to process, err: %v, are you sure to continue?\nPress `Ctrl+C` to skip or press 'Enter' to continue...",err)
				reader := bufio.NewReader(os.Stdin)
				reader.ReadString('\n')
				break
			}
		}
		if !next{
			break
		}
	}

	log.Info("warmup finished")
	return loadStats.NumRequests
}

func (cfg *LoadGenerator) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}
