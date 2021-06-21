package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"
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
	TotReqSize    int64
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

	httpClient=fasthttp.Client{
		MaxConnsPerHost: goroutines,
		ReadTimeout: time.Second * 60,
		WriteTimeout: time.Second * 60,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	rt = &LoadGenerator{duration, goroutines, statsAggregator, 0}
	return
}

var httpClient fasthttp.Client

func doRequest(item RequestItem) (result RequestResult) {
	result,_,_=doRequestWithFlag(item)
	return result
}
func doRequestWithFlag(item RequestItem) (result RequestResult,respBody []byte,err error) {

	result= RequestResult{}

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

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	//req.Header.Set("Connection", "close")

	if len(item.Request.Body) > 0 {
		reqBytes := []byte(item.Request.Body)
		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), reqBytes, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			req.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.Write(reqBytes)
				w.Flush()
			})
		}
	}

	if global.Env().IsDebug {
		log.Tracef(item.Request.Method)
		log.Tracef(item.Request.Url)
		log.Tracef(item.Request.Body)
	}

	start := time.Now()

	err = httpClient.Do(req, resp)

	result.Duration = time.Since(start)
	result.Status = resp.StatusCode()

	stats.Timing("request", "duration", result.Duration.Milliseconds())
	stats.Increment("request", "total")
	stats.Increment("request", strconv.Itoa(resp.StatusCode()))

	result.RequestSize = req.GetRequestLength()
	result.ResponseSize = resp.GetResponseLength()

	respBody=resp.GetRawBody()

	if resp.StatusCode()==0{
		if err!=nil{
			if global.Env().IsDebug {
				log.Error(err)
				log.Error(string(respBody))
			}
		}
	}else if resp.StatusCode()!=200{
		if global.Env().IsDebug {
			log.Error(err)
			log.Error(string(respBody))
		}
	}

	//skip verify
	if err != nil {
		result.Error=true
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

func (cfg *LoadGenerator) Run(config AppConfig) {
	stats := &LoadStats{MinRequestTime: time.Minute,StatusCode: map[int]int{}}
	start := time.Now()

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {

		for _, v := range config.Requests {

			if rateLimit > 0 {
			RetryRateLimit:
				if !rate.GetRateLimiter("loadgen", "requests", int(rateLimit),1,time.Minute*1).Allow() {
				//if !rate.GetRateLimiterPerSecond("loadgen", "requests", int(rateLimit)).Allow() {
					time.Sleep(10 * time.Millisecond)
					goto RetryRateLimit
				}
			}

			//replace url variable
			v=prepareRequest(v,config)

			result := doRequest(v)

			if !result.Valid{
				stats.NumInvalid++
			}

			if result.Error{
				stats.NumErrs++
			}

			if result.RequestSize>0{
				stats.TotReqSize += int64(result.RequestSize)
			}

			if result.ResponseSize>0{
				stats.TotRespSize += int64(result.ResponseSize)
			}

			////move to async
			stats.TotDuration += result.Duration
			stats.MaxRequestTime = util.MaxDuration(result.Duration, stats.MaxRequestTime)
			stats.MinRequestTime = util.MinDuration(result.Duration, stats.MinRequestTime)
			stats.NumRequests++
			v,ok:=stats.StatusCode[result.Status]
			if !ok{
				v=0
			}
			stats.StatusCode[result.Status]=v+1
		}
	}

	cfg.statsAggregator <- stats
}

func prepareRequest(v RequestItem,config AppConfig) RequestItem {

	if v.Request.HasVariable {
		if util.ContainStr(v.Request.Url, "$") {
			v.Request.Url = config.ReplaceVariable(v.Request.Url)
		}
	}

	if v.Request.RepeatBodyNTimes>0{
		buffer:=bytes.Buffer{}
		for i:=0;i<v.Request.RepeatBodyNTimes;i++{
			if v.Request.HasVariable {
				body:=v.Request.Body
				if util.ContainStr(body, "$") {
					body = config.ReplaceVariable(body)
				}
				buffer.WriteString(body)
			}else{
				buffer.WriteString(v.Request.Body)
			}
		}
		v.Request.Body=buffer.String()
	}else{
		if v.Request.HasVariable {
			body:=v.Request.Body
			if util.ContainStr(body, "$") {
				body = config.ReplaceVariable(body)
			}
			v.Request.Body=body
		}
	}


	return v
}

func (cfg *LoadGenerator) Warmup(config AppConfig) {
	log.Info("warmup started")
	for _, v := range config.Requests {
		v=prepareRequest(v,config)
		result,respBody,err := doRequestWithFlag(v)
		log.Infof("[%v] %v",v.Request.Method,v.Request.Url)
		log.Infof("status: %v,%v,%v",result.Status,err,string(respBody))
		if result.Status>=400||result.Status==0{
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
