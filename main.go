// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Loadgen is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
	"context"
	_ "embed"
	E "errors"
	"flag"
	"fmt"
	"github.com/jamiealquiza/tachymeter"
	"os"
	"os/signal"
	"strings"
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

var maxDuration int = 10
var goroutines int = 2
var rateLimit int = -1
var reqLimit int = -1
var timeout int = 60
var readTimeout int = 0
var writeTimeout int = 0
var dialTimeout int = 3
var compress bool = false
var mixed bool = false
var dslFileToRun string
var statsAggregator chan *LoadStats

func init() {
	flag.IntVar(&goroutines, "c", 1, "Number of concurrent threads")
	flag.IntVar(&maxDuration, "d", 5, "Duration of tests in seconds")
	flag.IntVar(&rateLimit, "r", -1, "Max requests per second (fixed QPS)")
	flag.IntVar(&reqLimit, "l", -1, "Limit total requests")
	flag.IntVar(&timeout, "timeout", 0, "Request timeout in seconds, default 0s")
	flag.IntVar(&readTimeout, "read-timeout", 0, "Connection read timeout in seconds, default 0s (use -timeout)")
	flag.IntVar(&writeTimeout, "write-timeout", 0, "Connection write timeout in seconds, default 0s (use -timeout)")
	flag.IntVar(&dialTimeout, "dial-timeout", 3, "Connection dial timeout in seconds, default 3s")
	flag.BoolVar(&compress, "compress", false, "Compress requests with gzip")
	flag.BoolVar(&mixed, "mixed", false, "Mixed requests from Yaml/DSL")
	flag.StringVar(&dslFileToRun, "run", "", "DSL config to run tests")
}

func startLoader(cfg *LoaderConfig) *LoadStats {
	defer log.Flush()

	statsAggregator = make(chan *LoadStats, goroutines)
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt)

	flag.Parse()

	if cfg.RunnerConfig.MetricSampleSize <= 0 {
		cfg.RunnerConfig.MetricSampleSize = 10000
	}

	// Initialize tachymeter.
	timer := tachymeter.New(&tachymeter.Config{Size: cfg.RunnerConfig.MetricSampleSize})

	loadGen := NewLoadGenerator(maxDuration, goroutines, statsAggregator, cfg.RunnerConfig.DisableHeaderNamesNormalizing)

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

	// Start wall time for all Goroutines.
	wallTimeStart := time.Now()

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

		go loadGen.Run(cfg, thisDoc, timer)
	}

	responders := 0
	aggStats := LoadStats{MinRequestTime: time.Millisecond, StatusCode: map[int]int{}}

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

	finalDuration := time.Since(wallTimeStart)

	// When finished, set elapsed wall time.
	timer.SetWallTime(finalDuration)

	avgThreadDur := aggStats.TotDuration / time.Duration(responders) //need to average the aggregated duration

	roughReqRate := float64(aggStats.NumRequests) / float64(finalDuration.Seconds())
	roughReqBytesRate := float64(aggStats.TotReqSize) / float64(finalDuration.Seconds())
	roughBytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / float64(finalDuration.Seconds())

	reqRate := float64(aggStats.NumRequests) / avgThreadDur.Seconds()
	avgReqTime := aggStats.TotDuration / time.Duration(aggStats.NumRequests)
	bytesRate := float64(aggStats.TotRespSize+aggStats.TotReqSize) / avgThreadDur.Seconds()

	// Flush before printing stats to avoid logging mixing with stats
	log.Flush()

	if cfg.RunnerConfig.NoSizeStats {
		fmt.Printf("\n%v requests finished in %v\n", aggStats.NumRequests, avgThreadDur)
	} else {
		fmt.Printf("\n%v requests finished in %v, %v sent, %v received\n", aggStats.NumRequests, avgThreadDur, util.ByteValue{float64(aggStats.TotReqSize)}, util.ByteValue{float64(aggStats.TotRespSize)})
	}

	fmt.Println("\n[Loadgen Client Metrics]")

	fmt.Printf("Requests/sec:\t\t%.2f\n", roughReqRate)

	if !cfg.RunnerConfig.BenchmarkOnly && !cfg.RunnerConfig.NoSizeStats {
		fmt.Printf(
			"Request Traffic/sec:\t%v\n"+
				"Total Transfer/sec:\t%v\n",
			util.ByteValue{roughReqBytesRate},
			util.ByteValue{roughBytesRate})
	}

	fmt.Printf("Fastest Request:\t%v\n", aggStats.MinRequestTime)
	fmt.Printf("Slowest Request:\t%v\n", aggStats.MaxRequestTime)

	if cfg.RunnerConfig.AssertError {
		fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
	}

	if cfg.RunnerConfig.AssertInvalid {
		fmt.Printf("Number of Invalid:\t%v\n", aggStats.NumInvalid)
	}

	for k, v := range aggStats.StatusCode {
		fmt.Printf("Status %v:\t\t%v\n", k, v)
	}

	if !cfg.RunnerConfig.BenchmarkOnly && !cfg.RunnerConfig.NoStats {
		// Rate outputs will be accurate.
		fmt.Println("\n[Latency Metrics]")
		fmt.Println(timer.Calc().String())

		fmt.Println("\n[Latency Distribution]")
		fmt.Println(timer.Calc().Histogram.String(30))
	}

	fmt.Printf("\n[Estimated Server Metrics]\nRequests/sec:\t\t%.2f\nAvg Req Time:\t\t%v\n", reqRate, avgReqTime)
	if !cfg.RunnerConfig.BenchmarkOnly && !cfg.RunnerConfig.NoSizeStats {
		fmt.Printf("Transfer/sec:\t\t%v\n", util.ByteValue{bytesRate})
	}

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

	app.IgnoreMainConfigMissing()
	app.Init(nil)

	defer app.Shutdown()
	appConfig := AppConfig{}

	if app.Setup(func() {
		module.RegisterUserPlugin(&stats.StatsDModule{})
		module.Start()

		environments := map[string]string{}
		ok, err := env.ParseConfig("env", &environments)
		if ok && err != nil {
			if ok && err != nil {
				if global.Env().SystemConfig.Configs.PanicOnConfigError {
					panic(err)
				} else {
					log.Error(err)
				}
			}
		}

		// Append system environment variables.
		environs := os.Environ()
		for _, env := range environs {
			kv := strings.Split(env, "=")
			if len(kv) == 2 {
				k, v := kv[0], kv[1]
				if _, ok := environments[k]; ok {
					environments[k] = v
				}
			}
		}

		tests := []Test{}
		ok, err = env.ParseConfig("tests", &tests)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)
			}
		}

		requests := []RequestItem{}
		ok, err = env.ParseConfig("requests", &requests)
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

		runnerConfig := RunnerConfig{
			ValidStatusCodesDuringWarmup: []int{},
		}
		ok, err = env.ParseConfig("runner", &runnerConfig)
		if ok && err != nil {
			if global.Env().SystemConfig.Configs.PanicOnConfigError {
				panic(err)
			} else {
				log.Error(err)
			}
		}

		appConfig.Environments = environments
		appConfig.Tests = tests
		appConfig.Requests = requests
		appConfig.Variable = variables
		appConfig.RunnerConfig = runnerConfig
		appConfig.Init()
	}, func() {
		go func() {
			//dsl go first
			if dslFileToRun != "" {
				log.Debugf("running DSL based requests from %s", dslFileToRun)
				if status := runDSLFile(&appConfig, dslFileToRun); status != 0 {
					os.Exit(status)
				}
				if !mixed {
					os.Exit(0)
					return
				}
			}

			if len(appConfig.Requests) != 0 {
				log.Debugf("running YAML based requests")
				if status := runLoaderConfig(&appConfig.LoaderConfig); status != 0 {
					os.Exit(status)
				}
				if !mixed {
					os.Exit(0)
					return
				}
			}

			//test suit go last
			if len(appConfig.Tests) != 0 {
				log.Debugf("running test suite")
				if !startRunner(&appConfig) {
					os.Exit(1)
				}
				if !mixed {
					os.Exit(0)
					return
				}
			}

			os.Exit(0)
		}()

	}, nil) {
		app.Run()
	}

	time.Sleep(1 * time.Second)

}

func runDSLFile(appConfig *AppConfig, path string) int {

	path = util.TryGetFileAbsPath(path, false)
	dsl, err := env.LoadConfigContents(path)
	if err != nil {
		if global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		} else {
			log.Error(err)
		}
	}
	log.Infof("loading config: %s", path)

	return runDSL(appConfig, dsl)
}

func runDSL(appConfig *AppConfig, dsl string) int {
	loaderConfig := parseDSL(appConfig, dsl)
	return runLoaderConfig(&loaderConfig)
}

func runLoaderConfig(config *LoaderConfig) int {
	err := config.Init()
	if err != nil {
		panic(err)
	}

	aggStats := startLoader(config)
	if aggStats != nil {
		if config.RunnerConfig.AssertInvalid && aggStats.NumInvalid > 0 {
			return 1
		}
		if config.RunnerConfig.AssertError && aggStats.NumErrs > 0 {
			return 2
		}
	}

	return 0
}

// parseDSL parses a DSL string to LoaderConfig.
func parseDSL(appConfig *AppConfig, input string) (output LoaderConfig) {
	output = LoaderConfig{}
	output.RunnerConfig = appConfig.RunnerConfig
	output.Variable = appConfig.Variable

	outputStr, err := loadPlugins([][]byte{loadgenDSL}, input)
	if err != nil {
		if global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		} else {
			log.Error(err)
		}
	}
	log.Debugf("using config:\n%s", outputStr)

	outputParser, err := coreConfig.NewConfigWithYAML([]byte(outputStr), "loadgen-dsl")
	if err != nil {
		if global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		} else {
			log.Error(err)
		}
	}

	if err := outputParser.Unpack(&output); err != nil {
		if global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		} else {
			log.Error(err)
		}
	}

	return
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
		// pipe output
		input = output
	}
	return
}

func callPlugin(ctx context.Context, mod wasmAPI.Module, input string) (output string, err error) {
	alloc := mod.ExportedFunction("allocate")
	free := mod.ExportedFunction("deallocate")
	process := mod.ExportedFunction("process")

	// 1) Plugins do not have access to host memory, so the first step is to copy
	//    the input string to the WASM VM.
	inputSize := uint32(len(input))
	ret, err := alloc.Call(ctx, uint64(inputSize))
	if err != nil {
		return
	}
	inputPtr := ret[0]
	defer free.Call(ctx, inputPtr)
	_, inputAddr, _ := decodePtr(inputPtr)
	mod.Memory().Write(inputAddr, []byte(input))

	// 2) Invoke the `process` function to handle the input string, which returns
	//    a result pointer (referred to as `decodePtr` in the following text)
	//    representing the processing result.
	ret, err = process.Call(ctx, inputPtr)
	if err != nil {
		return
	}
	outputPtr := ret[0]
	defer free.Call(ctx, outputPtr)

	// 3) Check the processing result.
	errors, outputAddr, outputSize := decodePtr(outputPtr)
	bytes, _ := mod.Memory().Read(outputAddr, outputSize)

	if errors {
		err = E.New(string(bytes))
	} else {
		output = string(bytes)
	}
	return
}

// decodePtr decodes error state and memory address of a result pointer.
//
// Some functions may return success or failure as a result. In such cases, a
// special 64-bit pointer is returned. The highest bit in the upper 32 bits
// serves as a boolean value indicating success (1) or failure (0), while the
// remaining 31 bits represent the length of the message. The lower 32 bits of
// the pointer represent the memory address of the specific message.
func decodePtr(ptr uint64) (errors bool, addr, size uint32) {
	const SIZE_MASK uint32 = (^uint32(0)) >> 1
	addr = uint32(ptr)
	size = uint32(ptr>>32) & SIZE_MASK
	errors = (ptr >> 63) != 0
	return
}
