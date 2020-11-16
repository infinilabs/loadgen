package loader

import (
	"crypto/tls"
	"infini.sh/framework/lib/fasthttp"
)

func client() (*fasthttp.Client, error) {


	client := &fasthttp.Client{
	MaxConnsPerHost: 60000,
	TLSConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	req := fasthttp.AcquireRequest()
	//req.SetBody([]byte(`{"username":"xxxxxx", "password":"xxxxxx"}`))
	//req.Header.SetContentType("application/x-www-form-urlencoded")
	//req.Header.SetMethod("POST")
	//resp := fasthttp.AcquireResponse()
	req.SetRequestURI("https://127.0.0.1:8060/fip/v1/auth/login")
	defer fasthttp.ReleaseRequest(req)
	//defer fasthttp.ReleaseResponse(resp)
	//if err := hc.DoTimeout(req, resp, 20*time.Second); err != nil {
	//}

	//fmt.Println(string(resp.Body()))

	//
	//client := &http.Client{}
	////overriding the default parameters
	//client.Transport = &http.Transport{
	//	DisableCompression:    disableCompression,
	//	DisableKeepAlives:     disableKeepAlive,
	//	ResponseHeaderTimeout: time.Millisecond * time.Duration(timeoutms),
	//}
	//
	//if !allowRedirects {
	//	//returning an error when trying to redirect. This prevents the redirection from happening.
	//	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
	//		return util.NewRedirectError("redirection not allowed")
	//	}
	//}
	//
	//if clientCert == "" && clientKey == "" && caCert == "" {
	//	return client, nil
	//}
	//
	//if clientCert == "" {
	//	return nil, fmt.Errorf("client certificate can't be empty")
	//}
	//
	//if clientKey == "" {
	//	return nil, fmt.Errorf("client key can't be empty")
	//}
	//
	//tlsConfig := &tls.Config{
	//	InsecureSkipVerify: true,
	//}
	//
	//tlsConfig.BuildNameToCertificate()
	//t := &http.Transport{
	//	TLSClientConfig: tlsConfig,
	//}
	//
	//if usehttp2 {
	//	http2.ConfigureTransport(t)
	//}
	//client.Transport = t
	return client, nil
}
