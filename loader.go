package main

import (
	"fmt"
	"infini.sh/framework/core/global"
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
	testUrl            string
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
	testUrl string,
	statsAggregator chan *RequesterStats,
) (rt *LoadCfg) {
	rt = &LoadCfg{duration, goroutines,testUrl,  statsAggregator,0}
	return
}

func escapeUrlStr(in string) string {
	qm := strings.Index(in, "?")
	if qm != -1 {
		qry := in[qm+1:]
		qrys := strings.Split(qry, "&")
		var query string = ""
		var qEscaped string = ""
		var first bool = true
		for _, q := range qrys {
			qSplit := strings.Split(q, "=")
			if len(qSplit) == 2 {
				qEscaped = qSplit[0] + "=" + url.QueryEscape(qSplit[1])
			} else {
				qEscaped = qSplit[0]
			}
			if first {
				first = false
			} else {
				query += "&"
			}
			query += qEscaped

		}
		return in[:qm] + "?" + query
	} else {
		return in
	}
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
	if item.Request.Body!=""{
		req.SetBody([]byte(item.Request.Body))
	}

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
	sid:=util.GetIncrementID("user_id")
	req.Header.Add("User-ID", util.IntToString(int(sid)))
	//if host != "" {
	//	req.Host = host
	//}
	start := time.Now()


	err=httpClient.Do(req, resp)

	if err != nil {
		valid=false

		fmt.Println("redirect?")
		//this is a bit weird. When redirection is prevented, a url.Error is retuned. This creates an issue to distinguish
		//between an invalid URL that was provided and and redirection error.
		rr, ok := err.(*url.Error)
		if !ok {
			fmt.Println("An error occured doing request", err, rr)
			return
		}
		fmt.Println("An error occured doing request", err)
	}
	if resp == nil {
		fmt.Println("empty response")
		return
	}
	resBody:=string(resp.Body())
	if item.Response.Status>0{
		if resp.StatusCode()!=item.Response.Status{
			if global.Env().IsDebug {
				log.Error("invalid status,",item.Request.Url, resp.StatusCode(),len(resBody),resBody)
			}
			//os.Exit(1)
			valid=false
		}
	}

	if item.Response.BodySize>0{
		if len(resBody)!=item.Response.BodySize{
			if global.Env().IsDebug {
				log.Error("invalid response size,",item.Request.Url, resp.StatusCode(),len(resBody),resBody)
				util.FileAppendNewLine("data/invalid_body_size.log",fmt.Sprintf("SID: %v, URL:%v Status: %v Header: %v Header: %v \nBody: %v\n",sid,item.Request.Url,resp.StatusCode(),req.Header.String(),resp.Header.String(),resBody))
			}
			//os.Exit(1)
			valid=false
		}
	}

	if item.Response.Body!=""{
		if string(resp.Body())!=item.Response.Body{

			if global.Env().IsDebug{
				log.Error("invalid response,",item.Request.Url, resp.StatusCode(),",",len(resBody),",",resBody)
				util.FileAppendNewLine("data/invalid_body_content.log",fmt.Sprintf("SID: %v, URL:%v Status: %v Header: %v Header: %v \nBody: %v\n",sid,item.Request.Url,resp.StatusCode(),req.Header.String(),resp.Header.String(),resBody))

			}
			//os.Exit(1)
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

	return
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

	regex,err:=regexp.Compile("(\\$\\[\\[(\\w+?)\\]\\])")

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {

		for _,v:=range config.Requests{

			//replace variable

			if err!=nil{
				panic(err)
			}

			if v.Request.HasVariable{
				allMatchs:=regex.FindAllString(v.Request.Body,-1)
				for _,v1:=range allMatchs{
					vold:=v1
					v1=util.TrimLeftStr(v1,"$[[")
					v1=util.TrimRightStr(v1,"]]")
					variable:=config.GetVariable(v1)
					v.Request.Body=strings.ReplaceAll(v.Request.Body,vold,fmt.Sprintf("%s",util.TrimSpaces(variable)))
				}
				if global.Env().IsDebug{
					log.Debug("replaced body:",v.Request)
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
