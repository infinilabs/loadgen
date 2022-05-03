package main

import (
	"bufio"
	"crypto/tls"
	"infini.sh/framework/lib/bytebufferpool"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"time"
)

const (
	UserAgent = "loadgen"
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

func NewLoadGenerator(duration int, goroutines int, statsAggregator chan *LoadStats,
) (rt *LoadGenerator) {

	httpClient = fasthttp.Client{
		ReadTimeout:     time.Second * 60,
	    WriteTimeout:    time.Second * 60,
		MaxConnsPerHost: goroutines,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	rt = &LoadGenerator{duration, goroutines, statsAggregator, 0}
	return
}

var httpClient fasthttp.Client

var resultPool = &sync.Pool {
	New: func()interface{} {
	return &RequestResult{}
	},
}

func doRequest(item *RequestItem,result *RequestResult){
	doRequestWithFlag(item,result)
}

func doRequestWithFlag(item *RequestItem,result *RequestResult) (respBody []byte, err error) {
	result.Valid = true
	req := fasthttp.AcquireRequest()
	req.Reset()
	req.ResetBody()
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	resp.ResetBody()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(item.Request.Method)
	req.SetRequestURI(item.Request.Url)

	if item.Request.BasicAuth.Username != "" {
		req.SetBasicAuth(item.Request.BasicAuth.Username, item.Request.BasicAuth.Password)
	}

	if len(item.Request.Headers) > 0 {
		for _, v := range item.Request.Headers {
			for k1, v1 := range v {
				req.Header.Set(k1, v1)
			}
		}
	}

	req.Header.Add("User-Agent", UserAgent)
	bodyBytes:=item.Request.GetBodyBytes()
	req.Header.Set("X-PayLoad-Size",util.ToString(len(bodyBytes)))

	if len(bodyBytes) > 0 {
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), bodyBytes, fasthttp.CompressBestCompression)
			if err != nil {
				panic(err)
			}

			req.Header.Set(fasthttp.HeaderAcceptEncoding, "gzip")
			req.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
			req.Header.Set("X-PayLoad-Compressed",util.ToString(true))

		} else {
			req.SetBody(bodyBytes)
		}
	}

	if global.Env().IsDebug {
		log.Tracef(item.Request.Method)
		log.Tracef(item.Request.Url)
		log.Tracef(string(bodyBytes))
	}

	start := time.Now()

	err = httpClient.DoTimeout(req, resp,60*time.Second)

	result.Duration = time.Since(start)
	result.Status = resp.StatusCode()

	stats.Timing("request", "duration", result.Duration.Milliseconds())
	stats.Increment("request", "total")
	stats.Increment("request", strconv.Itoa(resp.StatusCode()))

	result.RequestSize = req.GetRequestLength()
	result.ResponseSize = resp.GetResponseLength()

	respBody = resp.GetRawBody()

	if resp.StatusCode() == 0 {
		if err != nil {
			if global.Env().IsDebug {
				log.Error(err)
				log.Error(string(respBody))
			}
		}
	} else if resp.StatusCode() != 200 {
		if global.Env().IsDebug {
			log.Error(err)
			log.Error(string(respBody))
		}
	}

	//skip verify
	if err != nil {
		result.Error = true
		if global.Env().IsDebug {
			log.Error(err)
			log.Error(string(respBody))
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
			log.Debug(string(respBody))
		}
	}

	if item.ResponseAssert != nil {
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

var regex = regexp.MustCompile("(\\$\\[\\[(\\w+?)\\]\\])")
var bufferPool =bytebufferpool.NewPool(65536,655360)

func (cfg *LoadGenerator) Run(config AppConfig, countLimit int) {
	stats := &LoadStats{MinRequestTime: time.Minute, StatusCode: map[int]int{}}
	start := time.Now()

	limiter:=rate.GetRateLimiter("loadgen", "requests", int(rateLimit), 1, time.Second*1)
	buffer:=bufferPool.Get()
	defer bufferPool.Put(buffer)
	current := 0
	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {

		result:=resultPool.Get().(*RequestResult)
		defer resultPool.Put(result)

		for _, v := range config.Requests {

			if rateLimit > 0 {
			RetryRateLimit:
				if !limiter.Allow() {
					time.Sleep(10 * time.Millisecond)
					goto RetryRateLimit
				}
			}

			buffer.Reset()
			//replace url variable
			v.Request.bodyBuffer=buffer
			v.prepareRequest(config)

			result.Reset()
			doRequest(&v,result)

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

func (v *RequestItem)prepareRequest(config AppConfig) {
	runtimeVariables:=map[string]string{}
	if v.Request.HasVariable {
		if len(v.Request.RuntimeVariables)>0{
			for _,kv:=range v.Request.RuntimeVariables{
				for  k,v:=range kv{
					runtimeVariables[k]=config.GetVariable(runtimeVariables,v)
					//log.Info("get:",k,":",runtimeVariables[k])
				}
			}
		}

		if util.ContainStr(v.Request.Url, "$") {
			v.Request.Url = config.ReplaceVariable(runtimeVariables,v.Request.Url)
		}
	}

	if v.Request.RepeatBodyNTimes > 0 {
		for i := 0; i < v.Request.RepeatBodyNTimes; i++ {
			if v.Request.HasVariable {

				if len(v.Request.RuntimeBodyLineVariables)>0{
					for _,kv:=range v.Request.RuntimeBodyLineVariables{
						for  k,v:=range kv{
							runtimeVariables[k]=config.GetVariable(runtimeVariables,v)
						}
					}
				}

				body := v.Request.Body
				if util.ContainStr(body, "$") {
					body = config.ReplaceVariable(runtimeVariables,body)
				}
				v.Request.bodyBuffer.Write(util.UnsafeStringToBytes(body))
			} else {
				v.Request.bodyBuffer.Write(util.UnsafeStringToBytes(v.Request.Body))
			}
		}
	} else {
		if v.Request.HasVariable {

			if len(v.Request.RuntimeBodyLineVariables)>0{
				for _,kv:=range v.Request.RuntimeBodyLineVariables{
					for  k,v:=range kv{
						runtimeVariables[k]=config.GetVariable(runtimeVariables,v)
					}
				}
			}

			body := v.Request.Body
			if util.ContainStr(body, "$") {
				body = config.ReplaceVariable(runtimeVariables,body)
			}
			v.Request.bodyBuffer.WriteString(body)
		}else {
			v.Request.bodyBuffer.Write(util.UnsafeStringToBytes(v.Request.Body))
		}
	}
}

func (cfg *LoadGenerator) Warmup(config AppConfig) {
	log.Info("warmup started")
	buffer:=bufferPool.Get()
	defer bufferPool.Put(buffer)
	result:=resultPool.Get().(*RequestResult)
	defer resultPool.Put(result)
	for _, v := range config.Requests {

		result.Reset()
		buffer.Reset()

		v.Request.bodyBuffer=buffer
		v.prepareRequest(config)
		respBody, err := doRequestWithFlag(&v,result)
		log.Infof("[%v] %v -%v", v.Request.Method, v.Request.Url,util.SubString(string(respBody), 0, 256))
		log.Infof("status: %v,%v,%v", result.Status, err, util.SubString(string(respBody), 0, 256))
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
