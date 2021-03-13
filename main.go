package main

import (
	"flag"
	"fmt"
	"infini.sh/framework"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/loadgen/config"
	"os"
	"os/signal"
	"time"
)

var duration int = 10 //seconds
var goroutines int = 2
var statsAggregator chan *RequesterStats

func init() {
	flag.IntVar(&goroutines, "c", 10, "Number of goroutines to use (concurrent connections)")
	flag.IntVar(&duration, "d", 10, "Duration of test in seconds")
}

func startLoader(loadgenConfig LoadgenConfig) {

	statsAggregator = make(chan *RequesterStats, goroutines)
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt)

	flag.Parse() // Scan the arguments list

	loadGen := NewLoadCfg(duration, goroutines,statsAggregator)

	for i := 0; i < goroutines; i++ {
		go loadGen.RunSingleLoadSession(loadgenConfig)
	}

	responders := 0
	aggStats := RequesterStats{MinRequestTime: time.Minute}

	for responders < goroutines {
		select {
		case <-sigChan:
			loadGen.Stop()
			fmt.Printf("stopping...\n")
		case stats := <-statsAggregator:
			aggStats.NumErrs += stats.NumErrs
			aggStats.NumInvalid += stats.NumInvalid
			aggStats.NumRequests += stats.NumRequests
			aggStats.TotRespSize += stats.TotRespSize
			aggStats.TotDuration += stats.TotDuration
			aggStats.MaxRequestTime = MaxDuration(aggStats.MaxRequestTime, stats.MaxRequestTime)
			aggStats.MinRequestTime = MinDuration(aggStats.MinRequestTime, stats.MinRequestTime)
			responders++
		}
	}

	if aggStats.NumRequests == 0 {
		fmt.Println("Error: No statistics collected / no requests found\n")
		return
	}

	avgThreadDur := aggStats.TotDuration / time.Duration(responders) //need to average the aggregated duration

	reqRate := float64(aggStats.NumRequests) / avgThreadDur.Seconds()
	avgReqTime := aggStats.TotDuration / time.Duration(aggStats.NumRequests)
	bytesRate := float64(aggStats.TotRespSize) / avgThreadDur.Seconds()
	fmt.Printf("%v requests in %v, %v read\n", aggStats.NumRequests, avgThreadDur, ByteSize{float64(aggStats.TotRespSize)})
	fmt.Printf("Requests/sec:\t\t%.2f\nTransfer/sec:\t\t%v\nAvg Req Time:\t\t%v\n", reqRate, ByteSize{bytesRate}, avgReqTime)
	fmt.Printf("Fastest Request:\t%v\n", aggStats.MinRequestTime)
	fmt.Printf("Slowest Request:\t%v\n", aggStats.MaxRequestTime)
	fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
	fmt.Printf("Number of Invalid:\t%v\n", aggStats.NumInvalid)

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

	terminalHeader := ("")

	terminalFooter := ("Thanks for using LOADGEN, have a good day!")

	app := framework.NewApp("loadgen", "A http load generator and testing suit.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	global.RegisterShutdownCallback(func() {
		//os.Exit(1)
	})
	app.Start(func() {

		module.Start()

		loaderConfig:=LoadgenConfig{}

		items:=[]Item{}
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

		//fmt.Println(items)
		//
		//c,err:=client()
		// DoRequest(c,items[0])

		go func() {
			startLoader(loaderConfig)
		}()

	}, func() {
	})

	time.Sleep(1*time.Second)

}
