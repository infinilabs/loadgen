package main

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
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
		ReadTimeout:                   time.Second * time.Duration(readTimeout),
		WriteTimeout:                  time.Second * time.Duration(writeTimeout),
		MaxConnsPerHost:               goroutines,
		NoDefaultUserAgentHeader:      false,
		DisableHeaderNamesNormalizing: disableHeaderNamesNormalizing,
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Duration(dialTimeout)*time.Second)
		},
		Name:      global.Env().GetAppLowercaseName() + "/" + global.Env().GetVersion() + "/" + global.Env().GetBuildNumber(),
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}

	rt = &LoadGenerator{duration, goroutines, statsAggregator, 0}
	return
}

func doRequest(globalCtx util.MapStr, item *RequestItem, result *RequestResult) (reqBody, respBody []byte, err error) {
	result.Reset()

	var resp *fasthttp.Response

	if item.Request != nil {
		req := fasthttp.AcquireRequest()
		req.Reset()
		req.ResetBody()
		defer fasthttp.ReleaseRequest(req)
		//replace url variable
		item.prepareRequest(globalCtx, req)
		resp = fasthttp.AcquireResponse()
		resp.Reset()
		resp.ResetBody()
		defer fasthttp.ReleaseResponse(resp)

		start := time.Now()
		err = httpClient.DoTimeout(req, resp, time.Duration(timeout)*time.Second)
		result.Duration = time.Since(start)
		result.Status = resp.StatusCode()

		stats.Timing("request", "duration", result.Duration.Milliseconds())
		stats.Increment("request", "total")
		stats.Increment("request", strconv.Itoa(resp.StatusCode()))

		result.RequestSize = req.GetRequestLength()
		result.ResponseSize = resp.GetResponseLength()

		reqBody = req.GetRawBody()
		respBody = resp.GetRawBody()

		// skip verify
		if err != nil {
			result.Error = true
			if item.Assert != nil {
				result.Invalid = true
			}
			return
		}
	}

	if item.Sleep != nil {
		time.Sleep(time.Duration(item.Sleep.SleepInMilliSeconds) * time.Millisecond)
	}

	if item.Register != nil {
		event := buildCtx(resp, respBody, result)
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
		event := buildCtx(resp, respBody, result)
		// Dump globalCtx into assert event
		event.Update(globalCtx)
		if len(respBody) < 4096 {
			log.Debugf("assert _ctx: %+v", event)
		}
		condition, buildErr := conditions.NewCondition(item.Assert)
		if buildErr != nil {
			log.Errorf("failed to build conditions whilte assert existed, error: %+v", buildErr)
			result.Invalid = true
			return
		}
		if !condition.Check(event) {
			result.Invalid = true
		}
	}

	return
}

func buildCtx(resp *fasthttp.Response, respBody []byte, result *RequestResult) util.MapStr {
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
			"elapsed": int64(result.Duration / time.Millisecond),
		},
	}
	bodyJson := map[string]interface{}{}
	jsonErr := json.Unmarshal(respBody, &bodyJson)
	if jsonErr == nil {
		event.Put("_ctx.response.body_json", bodyJson)
	}
	return event
}

func (cfg *LoadGenerator) Run(config LoaderConfig, countLimit int) {
	stats := &LoadStats{MinRequestTime: time.Minute, StatusCode: map[int]int{}}
	start := time.Now()

	limiter := rate.GetRateLimiter("loadgen", "requests", int(rateLimit), 1, time.Second*1)
	result := resultPool.Get().(*RequestResult)
	defer resultPool.Put(result)

	// TODO: support concurrent access
	globalCtx := util.MapStr{}

	totalRequests := 0
	totalRounds := 0

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {
		if config.RunnerConfig.TotalRounds > 0 && totalRounds >= config.RunnerConfig.TotalRounds {
			goto END
		}
		totalRounds += 1

		result.Reset()

		for idx, item := range config.Requests {
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

			reqBody, respBody, err := doRequest(globalCtx, &item, result)

			if item.Request != nil && config.RunnerConfig.LogRequests || util.ContainsInAnyInt32Array(result.Status, config.RunnerConfig.LogStatusCodes) {
				log.Infof("[%v] %v, %v - %v", item.Request.Method, item.Request.Url, item.Request.Headers, util.SubString(string(reqBody), 0, 512))
				log.Infof("status: %v, error: %v, response: %v", result.Status, err, util.SubString(string(respBody), 0, 512))
			}

			if result.Error {
				stats.NumErrs++
			}

			if result.RequestSize > 0 {
				stats.TotReqSize += int64(result.RequestSize)
			}

			if result.ResponseSize > 0 {
				stats.TotRespSize += int64(result.ResponseSize)
			}

			////move to async
			stats.TotDuration += result.Duration
			stats.MaxRequestTime = util.MaxDuration(result.Duration, stats.MaxRequestTime)
			stats.MinRequestTime = util.MinDuration(result.Duration, stats.MinRequestTime)
			stats.NumRequests++
			stats.StatusCode[result.Status]++

			if result.Invalid {
				if item.Request != nil {
					fmt.Printf("#%d request, %s %s, assertion failed, skiping subsequent requests", idx, item.Request.Method, item.Request.Url)
				} else {
					fmt.Printf("#%d action, assertion failed, skiping subsequent requests", idx)
				}
				stats.NumInvalid++
				break
			}
		}
	}

END:
	cfg.statsAggregator <- stats
}

func (v *RequestItem) prepareRequest(globalCtx util.MapStr, req *fasthttp.Request) {
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

	req.SetRequestURI(url)

	log.Debugf("final request url: %s", url)

	//prepare method
	req.Header.SetMethod(v.Request.Method)

	if v.Request.BasicAuth.Username != "" {
		req.SetBasicAuth(v.Request.BasicAuth.Username, v.Request.BasicAuth.Password)
	}

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

func (cfg *LoadGenerator) Warmup(config LoaderConfig) int {
	log.Info("warmup started")
	result := resultPool.Get().(*RequestResult)
	defer resultPool.Put(result)
	globalCtx := util.MapStr{}
	reqCount := 0
	for _, v := range config.Requests {

		reqBody, respBody, err := doRequest(globalCtx, &v, result)
		reqCount += 1

		log.Infof("[%v] %v -%v", v.Request.Method, v.Request.Url, util.SubString(string(reqBody), 0, 512))
		log.Infof("status: %v, error: %v, response: %v", result.Status, err, util.SubString(string(respBody), 0, 512))
		if result.Status >= 400 || result.Status == 0 || err != nil {
			log.Info("requests seems failed to process, are you sure to continue?\nPress `Ctrl+C` to skip or press 'Enter' to continue...")
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
		}
	}

	log.Info("warmup finished")
	return reqCount
}

func (cfg *LoadGenerator) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}
