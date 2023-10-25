package main

import (
	"context"
	_ "embed"
	E "errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	log "github.com/cihub/seelog"
	wasm "github.com/tetratelabs/wazero"
	wasmAPI "github.com/tetratelabs/wazero/api"
	"infini.sh/framework"
	coreConfig "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	stats "infini.sh/framework/plugins/stats_statsd"
	"infini.sh/loadgen/config"
)

//go:embed plugins/loadgen_dsl.wasm
var loadgenDSL []byte

var duration int = 10
var goroutines int = 2
var rateLimit int = -1
var reqLimit int = -1
var timeout int = 60
var readTimeout int = 0
var writeTimeout int = 0
var dialTimeout int = 3
var compress bool = false
var statsAggregator chan *LoadStats

func init() {
	flag.IntVar(&goroutines, "c", 1, "Number of concurrent threads")
	flag.IntVar(&duration, "d", 5, "Duration of tests in seconds")
	flag.IntVar(&rateLimit, "r", -1, "Max requests per second (fixed QPS)")
	flag.IntVar(&reqLimit, "l", -1, "Limit total requests")
	flag.IntVar(&timeout, "timeout", 60, "Request timeout in seconds, default 60s")
	flag.IntVar(&readTimeout, "read-timeout", 0, "Connection read timeout in seconds, default 0s (use -timeout)")
	flag.IntVar(&writeTimeout, "write-timeout", 0, "Connection write timeout in seconds, default 0s (use -timeout)")
	flag.IntVar(&dialTimeout, "dial-timeout", 3, "Connection dial timeout in seconds, default 3s")
	flag.BoolVar(&compress, "compress", false, "Compress requests with gzip")
}

func startLoader(cfg AppConfig) *LoadStats {
	defer log.Flush()

	statsAggregator = make(chan *LoadStats, goroutines)
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt)

	flag.Parse()

	loadGen := NewLoadGenerator(duration, goroutines, statsAggregator, cfg.RunnerConfig.DisableHeaderNamesNormalizing)

	leftDoc := reqLimit

	if !cfg.RunnerConfig.NoWarm {
		reqCount := loadGen.Warmup(cfg)
		leftDoc -= reqCount
	}

	if reqLimit >= 0 && leftDoc <= 0 {
		log.Warn("No request to execute, exit now\n")
		return nil
	}

	var reqPerGoroutines int
	if reqLimit > 0 {
		if goroutines > leftDoc {
			goroutines = leftDoc
		}

		reqPerGoroutines = int((leftDoc + 1) / goroutines)
	}

	for i := 0; i < goroutines; i++ {
		thisDoc := -1
		if reqPerGoroutines > 0 {
			if leftDoc > reqPerGoroutines {
				thisDoc = reqPerGoroutines
			} else {
				thisDoc = leftDoc
			}
			leftDoc -= thisDoc
		}

		go loadGen.Run(cfg, thisDoc)
	}

	responders := 0
	aggStats := LoadStats{MinRequestTime: time.Minute, StatusCode: map[int]int{}}

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

			for k, v := range stats.StatusCode {
				oldV, ok := aggStats.StatusCode[k]
				if !ok {
					oldV = 0
				}
				aggStats.StatusCode[k] = oldV + v
			}

			responders++
		}
	}

	if aggStats.NumRequests == 0 {
		log.Error("Error: No statistics collected / no requests found")
		return nil
	}

	avgThreadDur := aggStats.TotDuration / time.Duration(responders) //need to average the aggregated duration

	roughReqRate := float64(aggStats.NumRequests) / float64(duration)
	roughReqBytesRate := float64(aggStats.TotReqSize) / float64(duration)
	roughBytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / float64(duration)
	roughAvgReqTime := (time.Duration(duration) * time.Second) / time.Duration(aggStats.NumRequests)

	reqRate := float64(aggStats.NumRequests) / avgThreadDur.Seconds()
	avgReqTime := aggStats.TotDuration / time.Duration(aggStats.NumRequests)
	bytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / avgThreadDur.Seconds()

	// Flush before printing stats to avoid logging mixing with stats
	log.Flush()

	fmt.Printf("\n%v requests in %v, %v sent, %v received\n", aggStats.NumRequests, avgThreadDur, util.ByteValue{float64(aggStats.TotReqSize)}, util.ByteValue{float64(aggStats.TotRespSize)})

	fmt.Println("\n[Loadgen Client Metrics]")
	fmt.Printf("Requests/sec:\t\t%.2f\n"+
		"Request Traffic/sec:\t%v\n"+
		"Total Transfer/sec:\t%v\n"+
		"Avg Req Time:\t\t%v\n",
		roughReqRate,
		util.ByteValue{roughReqBytesRate},
		util.ByteValue{roughBytesRate},
		roughAvgReqTime)
	fmt.Printf("Fastest Request:\t%v\n", aggStats.MinRequestTime)
	fmt.Printf("Slowest Request:\t%v\n", aggStats.MaxRequestTime)
	fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
	fmt.Printf("Number of Invalid:\t%v\n", aggStats.NumInvalid)
	for k, v := range aggStats.StatusCode {
		fmt.Printf("Status %v:\t\t%v\n", k, v)
	}

	fmt.Printf("\n[Estimated Server Metrics]\nRequests/sec:\t\t%.2f\nTransfer/sec:\t\t%v\nAvg Req Time:\t\t%v\n", reqRate, util.ByteValue{bytesRate}, avgReqTime)

	fmt.Println("")

	return &aggStats
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

	terminalFooter := ("")

	app := framework.NewApp("loadgen", "A http load generator and testing suite.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.BuildNumber), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	loaderConfig := AppConfig{}

	if app.Setup(func() {
		module.RegisterUserPlugin(&stats.StatsDModule{})
		module.Start()

		var (
			ok  bool
			err error
		)

		items := []RequestItem{}
		ok, err = env.ParseConfig("requests", &items)
		ok, err = env.ParseConfig("requests", &items)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)
			}
		}

		variables := []Variable{}
		ok, err = env.ParseConfig("variables", &variables)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)
			}
		}

		runnerConfig := RunnerConfig{}
		ok, err = env.ParseConfig("runner", &runnerConfig)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)

			}
		}

		loaderConfig.Requests = items
		loaderConfig.Variable = variables
		loaderConfig.RunnerConfig = runnerConfig

		dsl := struct {
			Value string `config:"value"`
		}{}
		ok, err = env.ParseConfig("dsl", &dsl)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)
			}
		}
		if dsl.Value != "" {
			output, err := loadPlugins([][]byte{loadgenDSL}, dsl.Value)
			if err != nil {
				if global.Env().SystemConfig.Configs.PanicOnConfigError {
					panic(err)
				} else {
					log.Error(err)
				}
			}
			if global.Env().IsDebug {
				log.Infof("using config\n%s", output)
			}
			outputParser, err := coreConfig.NewConfigWithYAML([]byte(output), "loadgen-dsl")
			if err != nil {
				if global.Env().SystemConfig.Configs.PanicOnConfigError {
					panic(err)
				} else {
					log.Error(err)
				}
			}
			if err := outputParser.Unpack(&loaderConfig); err != nil {
				if global.Env().SystemConfig.Configs.PanicOnConfigError {
					panic(err)
				} else {
					log.Error(err)
				}
			}
		}

		err = loaderConfig.Init()
		if err != nil {
			panic(err)
		}

	}, func() {

		go func() {
			aggStats := startLoader(loaderConfig)
			if aggStats != nil {
				if loaderConfig.RunnerConfig.AssertInvalid && aggStats.NumInvalid > 0 {
					os.Exit(1)
				}
				if loaderConfig.RunnerConfig.AssertError && aggStats.NumErrs > 0 {
					os.Exit(2)
				}
			}
			os.Exit(0)
		}()

	}, nil) {
		app.Run()
	}

	time.Sleep(1 * time.Second)

}

func loadPlugins(plugins [][]byte, input string) (output string, err error) {
	// init runtime
	ctx := context.Background()
	r := wasm.NewRuntime(ctx)
	defer r.Close(ctx)

	var mod wasmAPI.Module
	for _, plug := range plugins {
		// load plugin
		mod, err = r.Instantiate(ctx, plug)
		if err != nil {
			return
		}
		// call plugin
		output, err = callPlugin(ctx, mod, string(input))
		if err != nil {
			break
		}
	}
	return
}

func callPlugin(ctx context.Context, mod wasmAPI.Module, input string) (output string, err error) {
	alloc := mod.ExportedFunction("allocate")
	free := mod.ExportedFunction("deallocate")
	process := mod.ExportedFunction("process")

	// write input
	inputSize := uint32(len(input))
	ret, err := alloc.Call(ctx, uint64(inputSize))
	if err != nil {
		return
	}
	inputPtr := ret[0]
	defer free.Call(ctx, inputPtr)
	_, inputAddr, _ := decodePtr(inputPtr)
	mod.Memory().Write(inputAddr, []byte(input))

	// prepare memory for results
	ret, err = alloc.Call(ctx, uint64(4))
	if err != nil {
		return
	}
	errorPtr := ret[0]
	defer free.Call(ctx, errorPtr)

	// compile input
	ret, err = process.Call(ctx, inputPtr)
	if err != nil {
		return
	}
	outputPtr := ret[0]
	defer free.Call(ctx, outputPtr)

	// read output
	errors, outputAddr, outputSize := decodePtr(outputPtr)
	bytes, _ := mod.Memory().Read(outputAddr, outputSize)

	if errors {
		err = E.New(string(bytes))
	} else {
		output = string(bytes)
	}
	return
}

func decodePtr(ptr uint64) (errors bool, addr, size uint32) {
	const SIZE_MASK uint32 = (^uint32(0)) >> 1
	addr = uint32(ptr)
	size = uint32(ptr>>32) & SIZE_MASK
	errors = (ptr >> 63) != 0
	return
}
