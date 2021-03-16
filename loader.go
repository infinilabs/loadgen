package main

import (
	"bufio"
	"fmt"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	log "github.com/cihub/seelog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	USER_AGENT = "loadgen"
)

type LoadCfg struct {
	duration           int //seconds
	goroutines         int
	statsAggregator    chan *RequesterStats
	interrupted        int32
}

// RequesterStats used for colelcting aggregate statistics
type RequesterStats struct {
	TotRespSize    int64
	TotDuration    time.Duration
	MinRequestTime time.Duration
	MaxRequestTime time.Duration
	NumRequests    int
	NumErrs        int
	NumInvalid        int
}

func NewLoadCfg(duration int, //seconds
	goroutines int,
	statsAggregator chan *RequesterStats,
) (rt *LoadCfg) {
	rt = &LoadCfg{duration, goroutines,statsAggregator,0}
	return
}

//DoRequest single request implementation. Returns the size of the response and its duration
//On error - returns -1 on both
func DoRequest(httpClient *fasthttp.Client, item Item) (respSize int,err error,valid bool, duration time.Duration) {

	valid=true
	respSize = -1
	duration = -1

	req := fasthttp.AcquireRequest()
	req.Reset()
	resp := fasthttp.AcquireResponse()
	resp.Reset()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release


	req.Header.SetMethod(item.Request.Method)

	req.SetRequestURI(item.Request.Url)

	if item.Request.BasicAuth.Username!=""{
		req.SetBasicAuth(item.Request.BasicAuth.Username,item.Request.BasicAuth.Password)
	}

	if len(item.Request.Headers)>0{
		for _,v:=range item.Request.Headers{
			for k1,v1:=range v{
					req.Header.Set(k1,v1)
			}
		}
	}

	req.Header.Add("User-Agent", USER_AGENT)
	//sid:=util.GetIncrementID("user_id")
	//req.Header.Add("User-ID", util.IntToString(int(sid)))
	//if host != "" {
	//	req.Host = host
	//}

	if useGzip {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	if len(item.Request.Body) > 0 {

		reqBytes:=[]byte(item.Request.Body)
		//if item.Request.Body!=""{
		//	req.SetBody([]byte(item.Request.Body))
		//}

		if useGzip {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), reqBytes, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			//req.SetBody(body)
			req.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.Write(reqBytes)
				w.Flush()
			})

		}
	}

	start := time.Now()

	if global.Env().IsDebug{
		log.Tracef(item.Request.Method)
		log.Tracef(item.Request.Url)
		log.Tracef(item.Request.Body)
	}

	err=httpClient.Do(req, resp)

	stats.Increment("request","total")

	if err != nil {
		valid=false
		stats.Increment("request","invalid")

		//this is a bit weird. When redirection is prevented, a url.Error is retuned. This creates an issue to distinguish
		//between an invalid URL that was provided and and redirection error.
		rr, ok := err.(*url.Error)
		if !ok {
			fmt.Println("An error occured doing request", err, rr)
		}else{
			fmt.Println("An error occured doing request", err)
		}
		return
	}

	if resp == nil {
		stats.Increment("request","invalid")
		return
	}
	resBody:=string(resp.GetRawBody())
	if item.Response.Status>0{
		if resp.StatusCode()!=item.Response.Status{
			if global.Env().IsDebug {
				log.Error("invalid status,",item.Request.Url, resp.StatusCode(),len(resBody),resBody)
			}
			valid=false
		}
	}

	if global.Env().IsDebug{
		log.Trace(resBody)
	}

	if item.Response.BodySize>0{
		if len(resBody)!=item.Response.BodySize{
			if global.Env().IsDebug {
				log.Error("invalid response size,",item.Request.Url, resp.StatusCode(),len(resBody),resBody)
				//util.FileAppendNewLine("data/invalid_body_size.log",fmt.Sprintf("SID: %v, URL:%v Status: %v Header: %v Header: %v \nBody: %v\n",sid,item.Request.Url,resp.StatusCode(),req.Header.String(),resp.Header.String(),resBody))
			}
			valid=false
		}
	}

	if item.Response.Body!=""{
		if string(resBody)!=item.Response.Body{

			if global.Env().IsDebug{
				log.Error("invalid response,",item.Request.Url, resp.StatusCode(),",",len(resBody),",",resBody)
			}
			valid=false
		}
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {
		duration = time.Since(start)
		respSize = int(len(resp.Body())) + int(len(resp.Header.Header()))
	} else if resp.StatusCode() == http.StatusMovedPermanently || resp.StatusCode() == http.StatusTemporaryRedirect {
		duration = time.Since(start)
		respSize = int(len(resp.Body())) + int(len(resp.Header.Header()))
	} else {
		//fmt.Println("received status code", resp.StatusCode, "from", string(resp.Header.Header()), "content", string(body), req)
	}

	stats.Timing("request","duration_in_ms",duration.Milliseconds())

	if valid{
		stats.Increment("request","valid")

	}else{
		stats.Increment("request","invalid")
	}

	return
}

var regex=regexp.MustCompile("(\\$\\[\\[(\\w+?)\\]\\])")

func (config *LoadgenConfig)ReplaceVariable(v string) string {
	allMatchs:=regex.FindAllString(v,-1)
	for _,v1:=range allMatchs{
		vold:=v1
		v1=util.TrimLeftStr(v1,"$[[")
		v1=util.TrimRightStr(v1,"]]")
		variable:=config.GetVariable(v1)
		v=strings.ReplaceAll(v,vold,fmt.Sprintf("%s",util.TrimSpaces(variable)))
	}
	if global.Env().IsDebug{
		log.Debug("replaced body:",v)
	}
	return v
}

//Requester a go function for repeatedly making requests and aggregating statistics as long as required
//When it is done, it sends the results using the statsAggregator channel
func (cfg *LoadCfg) RunSingleLoadSession(config LoadgenConfig) {
	stats := &RequesterStats{MinRequestTime: time.Minute}
	start := time.Now()

	httpClient, err := client()
	if err != nil {
		log.Error(err)
		return
	}

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {

		for _,v:=range config.Requests{

			if rateLimit>0{
				RetryRateLimit:
				if !rate.GetRaterWithDefine("loadgen","requests", int(rateLimit)).Allow(){
					time.Sleep(10*time.Millisecond)
					goto RetryRateLimit
				}
			}


			//replace variable
			if v.Request.HasVariable{
				if util.ContainStr(v.Request.Url,"$"){
					v.Request.Url=config.ReplaceVariable(v.Request.Url)
				}

				if util.ContainStr(v.Request.Body,"$"){
					v.Request.Body=config.ReplaceVariable(v.Request.Body)
				}
			}

			respSize,err1,valid, reqDur := DoRequest(httpClient,  v)
			if !valid{
				stats.NumInvalid++
			}

			if err1!=nil{
				stats.NumErrs++
			}

			if respSize > 0 {
				stats.TotRespSize += int64(respSize)
				stats.TotDuration += reqDur
				stats.MaxRequestTime = MaxDuration(reqDur, stats.MaxRequestTime)
				stats.MinRequestTime = MinDuration(reqDur, stats.MinRequestTime)
				stats.NumRequests++
			}
		}
	}
	cfg.statsAggregator <- stats
}

func (cfg *LoadCfg) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}
