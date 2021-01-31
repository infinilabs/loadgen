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

import (
	"errors"
	"golang.org/x/sync/singleflight"
	"log"
	"time"
)

func main1() {
	var singleSetCache singleflight.Group

	getAndSetCache:=func (requestID int,cacheKey string) (string, error) {
		log.Printf("request %v start to get and set cache...",requestID)
		value,_, _ :=singleSetCache.Do(cacheKey, func() (ret interface{}, err error) {//do的入参key，可以直接使用缓存的key，这样同一个缓存，只有一个协程会去读DB
			log.Printf("request %v is setting cache...",requestID)
			time.Sleep(3*time.Second)
			log.Printf("request %v set cache success!",requestID)
			return "VALUE",nil
		})
		return value.(string),nil
	}

	cacheKey:="cacheKey"
	for i:=1;i<10;i++{//模拟多个协程同时请求
		go func(requestID int) {
			value,_:=getAndSetCache(requestID,cacheKey)
			log.Printf("request %v get value: %v",requestID,value)
		}(i)
	}
	time.Sleep(20*time.Second)
}

func main() {
	var singleSetCache singleflight.Group

	getAndSetCache:=func (requestID int,cacheKey string) (string, error) {
		log.Printf("request %v start to get and set cache...",requestID)
		retChan:=singleSetCache.DoChan(cacheKey, func() (ret interface{}, err error) {
			log.Printf("request %v is setting cache...",requestID)
			time.Sleep(3*time.Second)
			log.Printf("request %v set cache success!",requestID)
			return "VALUE",nil
		})

		var ret singleflight.Result

		timeout := time.After(5 * time.Second)

		select {//加入了超时机制
		case <-timeout:
			log.Printf("time out!")
			return "",errors.New("time out")
		case ret =<- retChan://从chan中取出结果
			return ret.Val.(string),ret.Err
		}
		return "",nil
	}

	cacheKey:="cacheKey"
	for i:=1;i<10;i++{
		go func(requestID int) {
			value,_:=getAndSetCache(requestID,cacheKey)
			log.Printf("request %v get value: %v",requestID,value)
		}(i)
	}
	time.Sleep(20*time.Second)
}
