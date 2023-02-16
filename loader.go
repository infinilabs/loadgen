package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"io"
	"os"
	"regexp"
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
	"infini.sh/framework/lib/bytebufferpool"
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

func NewLoadGenerator(duration int, goroutines int, statsAggregator chan *LoadStats, disableHeaderNamesNormalizing bool) (rt *LoadGenerator) {

	httpClient = fasthttp.Client{
		ReadTimeout:                   time.Second * 60,
		WriteTimeout:                  time.Second * 60,
		MaxConnsPerHost:               goroutines,
		NoDefaultUserAgentHeader:      false,
		DisableHeaderNamesNormalizing: disableHeaderNamesNormalizing,
		Name:                          global.Env().GetAppLowercaseName() + "/" + global.Env().GetVersion() + "/" + global.Env().GetBuildNumber(),
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
	}

	rt = &LoadGenerator{duration, goroutines, statsAggregator, 0}
	return
}

var httpClient fasthttp.Client

var resultPool = &sync.Pool{
	New: func() interface{} {
		return &RequestResult{}
	},
}

func doRequest(globalCtx util.MapStr, item *RequestItem, buffer *bytebufferpool.ByteBuffer, result *RequestResult) (reqBody, respBody []byte, err error) {

	result.Reset()
	result.Valid = true
	buffer.Reset()
	req := fasthttp.AcquireRequest()
	req.Reset()
	req.ResetBody()
	defer fasthttp.ReleaseRequest(req)
	//replace url variable
	item.prepareRequest(globalCtx, req, buffer)
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	resp.ResetBody()
	defer fasthttp.ReleaseResponse(resp)

	start := time.Now()
	err = httpClient.DoTimeout(req, resp, 60*time.Second)
	result.Duration = time.Since(start)
	result.Status = resp.StatusCode()

	stats.Timing("request", "duration", result.Duration.Milliseconds())
	stats.Increment("request", "total")
	stats.Increment("request", strconv.Itoa(resp.StatusCode()))

	result.RequestSize = req.GetRequestLength()
	result.ResponseSize = resp.GetResponseLength()

	reqBody = req.GetRawBody()
	respBody = resp.GetRawBody()

	if resp.StatusCode() == 0 {
		if err != nil {
			if global.Env().IsDebug {
				log.Error(err, string(respBody))
			}
		}
	} else if resp.StatusCode() != 200 {
		if global.Env().IsDebug {
			log.Error(err, string(respBody))
		}
	}

	//skip verify
	if err != nil {
		result.Error = true
		if global.Env().IsDebug {
			log.Error(err, string(respBody))
		}
		return
	}

	//skip verify
	if resp == nil {
		result.Valid = false
		return
	}

	if global.Env().IsDebug {
		if global.Env().IsDebug {
			log.Debug(string(reqBody))
			log.Debug(string(respBody))
		}
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
		log.Debugf("assert _ctx: %+v", event)
		condition, buildErr := conditions.NewCondition(item.Assert)
		if buildErr != nil {
			log.Error("failed to build conditions whilte assert existed, error: %+v", err)
			return
		}
		if !condition.Check(event) {
			if global.Env().IsDebug {
				log.Error("assert failed")
			}
			result.Valid = false
			return
		}
		return
	} else if item.ResponseAssert != nil {
		if global.Env().IsDebug {
			log.Trace(string(respBody))
		}

		if item.ResponseAssert.Status > 0 {
			if resp.StatusCode() != item.ResponseAssert.Status {
				if global.Env().IsDebug {
					log.Error("invalid status,", item.Request.Url, resp.StatusCode(), len(respBody), string(respBody))
				}
				result.Valid = false
				return
			}
		}

		if item.ResponseAssert.BodySize > 0 {
			if len(respBody) != item.ResponseAssert.BodySize {
				if global.Env().IsDebug {
					log.Trace("invalid response size,", item.Request.Url, resp.StatusCode(), len(respBody), respBody)
				}
				result.Valid = false
				return
			}
		}

		if item.ResponseAssert.Body != "" {
			if len(respBody) != len(item.ResponseAssert.Body) || string(respBody) != item.ResponseAssert.Body {
				if global.Env().IsDebug {
					log.Trace("invalid response,", item.Request.Url, resp.StatusCode(), ",", len(respBody), ",", respBody)
				}
				result.Valid = false
				return
			}
		}
	}
	return
}

func buildCtx(resp *fasthttp.Response, respBody []byte, result *RequestResult) util.MapStr {
	header := map[string]interface{}{}
	resp.Header.VisitAll(func(k, v []byte) {
		header[string(k)] = string(v)
	})
	event := util.MapStr{
		"_ctx": map[string]interface{}{
			"response": map[string]interface{}{
				"status":      resp.StatusCode(),
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

var regex = regexp.MustCompile("(\\$\\[\\[(\\w+?)\\]\\])")
var loadgenPool = bytebufferpool.NewTaggedPool("loadgen", 0, 100*1024*1024, 10000)

func (cfg *LoadGenerator) Run(config AppConfig, countLimit int) {
	stats := &LoadStats{MinRequestTime: time.Minute, StatusCode: map[int]int{}}
	start := time.Now()

	limiter := rate.GetRateLimiter("loadgen", "requests", int(rateLimit), 1, time.Second*1)
	buffer := loadgenPool.Get()
	defer loadgenPool.Put(buffer)
	current := 0
	result := resultPool.Get().(*RequestResult)
	defer resultPool.Put(result)

	// TODO: support concurrent access
	globalCtx := util.MapStr{}

	totalRounds := 0

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {
		if config.RunnerConfig.TotalRounds > 0 && totalRounds >= config.RunnerConfig.TotalRounds {
			goto END
		}
		totalRounds += 1

		buffer.Reset()
		result.Reset()

		for _, v := range config.Requests {

			if rateLimit > 0 {
			RetryRateLimit:
				if !limiter.Allow() {
					time.Sleep(10 * time.Millisecond)
					goto RetryRateLimit
				}
			}

			if config.RunnerConfig.LogRequests {
				log.Infof("[%v] %v, %v - %v", v.Request.Method, v.Request.Url, v.Request.Headers, util.SubString(string(reqBody), 0, 512))
				log.Infof("status: %v, %v, %v", result.Status, err, util.SubString(string(respBody), 0, 512))
			}

			reqBody, respBody, err := doRequest(globalCtx, &v, buffer, result)

			if !result.Valid {
				stats.NumInvalid++
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
			v, ok := stats.StatusCode[result.Status]
			if !ok {
				v = 0
			}
			stats.StatusCode[result.Status] = v + 1

			current++
			if countLimit > 0 && current == countLimit {
				goto END
			}
		}
	}

END:
	cfg.statsAggregator <- stats
}

func (v *RequestItem) prepareRequest(globalCtx util.MapStr, req *fasthttp.Request, bodyBuffer *bytebufferpool.ByteBuffer) {
	if v.Request.DisableHeaderNamesNormalizing {
		req.Header.DisableNormalizing()
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

				v.Request.bodyTemplate.ExecuteFuncStringExtend(bodyBuffer, func(w io.Writer, tag string) (int, error) {
					variable := GetVariable(runtimeVariables, tag)
					return w.Write(util.UnsafeStringToBytes(variable))
				})
			} else {
				bodyBuffer.WriteString(body)
			}
		}
	}

	req.Header.Set("X-PayLoad-Size", util.ToString(bodyBuffer.Len()))

	if bodyBuffer.Len() > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), bodyBuffer.B, fasthttp.CompressBestCompression)
			if err != nil {
				panic(err)
			}

			req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
			req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
			req.Header.Set("X-PayLoad-Compressed", util.ToString(true))

		} else {
			req.SetRawBody(bodyBuffer.B)
		}
	}
}

func (cfg *LoadGenerator) Warmup(config AppConfig) {
	log.Info("warmup started")
	buffer := loadgenPool.Get()
	defer loadgenPool.Put(buffer)
	result := resultPool.Get().(*RequestResult)
	defer resultPool.Put(result)
	globalCtx := util.MapStr{}
	for _, v := range config.Requests {

		reqBody, respBody, err := doRequest(globalCtx, &v, buffer, result)

		log.Infof("[%v] %v -%v", v.Request.Method, v.Request.Url, util.SubString(string(reqBody), 0, 512))
		log.Infof("status: %v,%v,%v", result.Status, err, util.SubString(string(respBody), 0, 512))
		if result.Status >= 400 || result.Status == 0 {
			log.Info("requests seems failed to process, are you sure to continue?\nPress `Ctrl+C` to skip or press 'Enter' to continue...")
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
		}
	}

	log.Info("warmup finished")
}

func (cfg *LoadGenerator) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}
