package main

import (
	"crypto/tls"
	"infini.sh/framework/lib/fasthttp"
	"sync"
)

var clientPool = &sync.Pool {
	New: func()interface{} {
	return &fasthttp.Client{
			MaxConnsPerHost: 60000,
			TLSConfig:       &tls.Config{InsecureSkipVerify: true},
		}
	},
}
