package main

import (
	"flag"
	"fmt"
	"infini.sh/framework"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	stats "infini.sh/framework/plugins/stats_statsd"
	"infini.sh/loadgen/config"
	"os"
	"os/signal"
	"time"
)

var duration int = 10
var goroutines int = 2
var rateLimit int = -1
var compress bool =false
var statsAggregator chan *LoadStats

func init() {
	flag.IntVar(&goroutines, "c", 1, "Number of concurrent threads")
	flag.IntVar(&duration, "d", 5, "Duration of tests in seconds")
	flag.IntVar(&rateLimit, "r", -1, "Max requests per second (fixed QPS)")
	flag.BoolVar(&compress, "compress", false, "Compress requests with gzip")
}

func startLoader(cfg AppConfig) {

	statsAggregator = make(chan *LoadStats, goroutines)
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt)

	flag.Parse() // Scan the arguments list

	loadGen := NewLoadGenerator(duration, goroutines,statsAggregator)

	loadGen.Warmup(cfg)


	for i := 0; i < goroutines; i++ {
		go loadGen.Run(cfg)
	}

	responders := 0
	aggStats := LoadStats{MinRequestTime: time.Minute,StatusCode: map[int]int{}}

	for responders < goroutines {
		select {
		case <-sigChan:
			loadGen.Stop()
		case stats := <-statsAggregator:
			aggStats.NumErrs += stats.NumErrs
			aggStats.NumInvalid += stats.NumInvalid
			aggStats.NumRequests += stats.NumRequests
			aggStats.TotReqSize += stats.TotReqSize
			aggStats.TotRespSize += stats.TotRespSize
			aggStats.TotDuration += stats.TotDuration
			aggStats.MaxRequestTime = util.MaxDuration(aggStats.MaxRequestTime, stats.MaxRequestTime)
			aggStats.MinRequestTime = util.MinDuration(aggStats.MinRequestTime, stats.MinRequestTime)

			for k,v:=range stats.StatusCode{
				oldV,ok:=aggStats.StatusCode[k]
				if !ok{
					oldV=0
				}
				aggStats.StatusCode[k]=oldV+v
			}

			responders++
		}
	}

	if aggStats.NumRequests == 0 {
		fmt.Println("Error: No statistics collected / no requests found\n")
		return
	}

	avgThreadDur := aggStats.TotDuration / time.Duration(responders) //need to average the aggregated duration

	roughReqRate := float64(aggStats.NumRequests) / float64(duration)
	roughReqBytesRate := float64(aggStats.TotReqSize) / float64(duration)
	roughBytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / float64(duration)
	roughAvgReqTime := (time.Duration(duration)*time.Second) / time.Duration(aggStats.NumRequests)

	reqRate := float64(aggStats.NumRequests) / avgThreadDur.Seconds()
	avgReqTime := aggStats.TotDuration / time.Duration(aggStats.NumRequests)
	bytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / avgThreadDur.Seconds()

	fmt.Printf("\n%v requests in %v, %v sent, %v received\n", aggStats.NumRequests, avgThreadDur, util.ByteValue{float64(aggStats.TotReqSize)},util.ByteValue{float64(aggStats.TotRespSize)})

	fmt.Println("\n[Loadgen Client Metrics]")
	fmt.Printf("Requests/sec:\t\t%.2f\n" +
		"Request Traffic/sec:\t%v\n" +
		"Total Transfer/sec:\t%v\n" +
		"Avg Req Time:\t\t%v\n",
		roughReqRate,
		util.ByteValue{roughReqBytesRate},
		util.ByteValue{roughBytesRate},
		roughAvgReqTime)
	fmt.Printf("Fastest Request:\t%v\n", aggStats.MinRequestTime)
	fmt.Printf("Slowest Request:\t%v\n", aggStats.MaxRequestTime)
	fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
	fmt.Printf("Number of Invalid:\t%v\n", aggStats.NumInvalid)
	for k,v:=range aggStats.StatusCode{
		fmt.Printf("Status %v:\t\t%v\n", k,v)
	}

	fmt.Printf("\n[Estimated Server Metrics]\nRequests/sec:\t\t%.2f\nTransfer/sec:\t\t%v\nAvg Req Time:\t\t%v\n", reqRate, util.ByteValue{bytesRate}, avgReqTime)

	fmt.Println("\n")


}

//func addProcessToCgroup(filepath string, pid int) {
//	file, err := os.OpenFile(filepath, os.O_WRONLY, 0644)
//	if err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//	defer file.Close()
//
//	if _, err := file.WriteString(fmt.Sprintf("%d", pid)); err != nil {
//		fmt.Println("failed to setup cgroup for the container: ", err)
//		os.Exit(1)
//	}
//}
//
//func cgroupSetup(pid int) {
//	for _, c := range []string{"cpu", "memory"} {
//		cpath := fmt.Sprintf("/sys/fs/cgroup/%s/mycontainer/", c)
//		if err := os.MkdirAll(cpath, 0644); err != nil {
//			fmt.Println("failed to create cpu cgroup for my container: ", err)
//			os.Exit(1)
//		}
//		addProcessToCgroup(cpath+"cgroup.procs", pid)
//	}
//}


func main() {

	terminalHeader := ("   __   ___  _      ___  ___   __    __\n")
	terminalHeader += ("  / /  /___\\/_\\    /   \\/ _ \\ /__\\/\\ \\ \\\n")
	terminalHeader += (" / /  //  ///_\\\\  / /\\ / /_\\//_\\ /  \\/ /\n")
	terminalHeader += ("/ /__/ \\_//  _  \\/ /_// /_\\\\//__/ /\\  /\n")
	terminalHeader += ("\\____|___/\\_/ \\_/___,'\\____/\\__/\\_\\ \\/\n\n")

	terminalFooter := ("Thanks for using LOADGEN, have a good day!")

	app := framework.NewApp("loadgen", "A http load generator and testing suit.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	global.RegisterShutdownCallback(func() {
		//os.Exit(1)
	})
	app.Start(func() {

		module.RegisterUserPlugin(stats.StatsDModule{})
		module.Start()

		loaderConfig:= AppConfig{}

		items:=[]RequestItem{}
		ok,err:=env.ParseConfig("requests",&items)
		if ok&&err!=nil{
			panic(err)
		}

		variables:=[]Variable{}
		ok,err=env.ParseConfig("variables",&variables)
		if ok&&err!=nil{
			panic(err)
		}

		loaderConfig.Requests=items
		loaderConfig.Variable=variables
		loaderConfig.Init()

		//TODO warm up
		//TODO show confirm message and confirm

		go func() {
			startLoader(loaderConfig)
		}()

	}, func() {
	})

	time.Sleep(1*time.Second)

}
