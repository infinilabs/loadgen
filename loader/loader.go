package loader

import (
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"infini.sh/http-loader/util"
)

const (
	USER_AGENT = "go-wrk"
)

type LoadCfg struct {
	duration           int //seconds
	goroutines         int
	testUrl            string
	//reqBody            string
	//method             string
	//host               string
	//header             map[string]string
	statsAggregator    chan *RequesterStats
	//timeoutms          int
	//allowRedirects     bool
	//disableCompression bool
	//disableKeepAlive   bool
	interrupted        int32
	//clientCert         string
	//clientKey          string
	//caCert             string
	//http2              bool
}

// RequesterStats used for colelcting aggregate statistics
type RequesterStats struct {
	TotRespSize    int64
	TotDuration    time.Duration
	MinRequestTime time.Duration
	MaxRequestTime time.Duration
	NumRequests    int
	NumErrs        int
}

func NewLoadCfg(duration int, //seconds
	goroutines int,
	testUrl string,
	//reqBody string,
	//method string,
	//host string,
	//header map[string]string,
	statsAggregator chan *RequesterStats,
	//timeoutms int,
	//allowRedirects bool,
	//disableCompression bool,
	//disableKeepAlive bool,
	//clientCert string,
	//clientKey string,
	//caCert string,
	//http2 bool
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
func DoRequest(httpClient *fasthttp.Client, loadUrl string) (respSize int, duration time.Duration) {

	respSize = -1
	duration = -1

	loadUrl = escapeUrlStr(loadUrl)
	//
	//var buf io.Reader
	//if len(reqBody) > 0 {
	//	buf = bytes.NewBufferString(reqBody)
	//}

	//req, err := http.NewRequest(method, loadUrl, buf)
	//if err != nil {
	//	fmt.Println("An error occured doing request", err)
	//	return
	//}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release


	req.SetRequestURI(loadUrl)
	//req.SetBody([]byte(reqBody))
	//
	//for hk, hv := range header {
	//	req.Header.Add(hk, hv)
	//}

	//req.Header.Add("User-Agent", USER_AGENT)
	//if host != "" {
	//	req.Host = host
	//}
	start := time.Now()


	err:=httpClient.Do(req, resp)

	if err != nil {
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
	//defer func() {
	//	if resp != nil && resp.Body != nil {
	//		resp.Body.Close()
	//	}
	//}()
	//body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("An error occured reading body", err)
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
func (cfg *LoadCfg) RunSingleLoadSession() {
	stats := &RequesterStats{MinRequestTime: time.Minute}
	start := time.Now()

	httpClient, err := client()
	if err != nil {
		log.Fatal(err)
	}

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {
		respSize, reqDur := DoRequest(httpClient,  cfg.testUrl)
		if respSize > 0 {
			stats.TotRespSize += int64(respSize)
			stats.TotDuration += reqDur
			stats.MaxRequestTime = util.MaxDuration(reqDur, stats.MaxRequestTime)
			stats.MinRequestTime = util.MinDuration(reqDur, stats.MinRequestTime)
			stats.NumRequests++
		} else {
			stats.NumErrs++
		}
	}
	cfg.statsAggregator <- stats
}

func (cfg *LoadCfg) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}
