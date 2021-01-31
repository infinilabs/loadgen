/*
Copyright Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

//- request:
//- method: GET
//url: /
//body:
//response:
//status: 200
//body:
//- request:
type Request struct {
	Method string `config:"method"`
	Url    string `config:"url"`
	Body   string `config:"body"`
	Headers []map[string]string `config:"headers"`
	BasicAuth struct{
		Username string `config:"username"`
		Password string `config:"password"`
	} `config:"basic_auth"`
}

type Response struct {
	Status int    `config:"status"`
	Body   string `config:"body"`
	BodySize   int `config:"body_size"`
}

type Item struct {
	Request  Request  `config:"request"`
	Response Response `config:"response"`
}
