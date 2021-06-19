package main

//var httpClient= &fasthttp.Client{
//	MaxConnsPerHost: 60000,
//	MaxConnDuration: time.Second * 10,
//	ReadTimeout: time.Second * 30,
//	WriteTimeout: time.Second * 30,
//	TLSConfig:       &tls.Config{InsecureSkipVerify: true},
//}

//var clientPool = &sync.Pool {
//	New: func()interface{} {
//	return &fasthttp.Client{
//			MaxConnsPerHost: 1000,
//			//MaxConnDuration: time.Second * 120,
//			ReadTimeout: time.Second * 60,
//			WriteTimeout: time.Second * 60,
//			TLSConfig:       &tls.Config{InsecureSkipVerify: true},
//		}
//	},
//}
