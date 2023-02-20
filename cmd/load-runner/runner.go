package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
)

type TestResult struct {
	Failed       bool  `json:"failed"`
	DurationInMs int64 `json:"duration_in_ms"`
	Error        error `json:"error"`
}

type TestMsg struct {
	Path string `json:"path"`
	// ABORTED/FAILED/SUCCESS
	Status       string `json:"status"`
	DurationInMs int64  `json:"duration_in_ms"`
}

func startRunner(appConfig *AppConfig) bool {
	defer log.Flush()
	msgs := make([]*TestMsg, len(appConfig.Tests))
	for i, test := range appConfig.Tests {
		result, err := runTest(appConfig, test)
		msg := &TestMsg{
			Path: test.Path,
		}
		if result == nil || err != nil {
			log.Debugf("failed to run test, error: %+v", err)
			msg.Status = "ABORTED"
		} else if result.Failed {
			msg.Status = "FAILED"
		} else {
			msg.Status = "SUCCESS"
		}
		if result != nil {
			msg.DurationInMs = result.DurationInMs
		}
		msgs[i] = msg
	}
	ok := true
	for _, msg := range msgs {
		log.Infof("[TEST][%s] [%s] duration: %d(ms)", msg.Path, msg.Status, msg.DurationInMs)
		if msg.Status != "SUCCESS" {
			ok = false
		}
	}
	return ok
}

func runTest(appConfig *AppConfig, test Test) (*TestResult, error) {
	// To kill loadgen/gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := path.Join(appConfig.Environments[env_LR_TEST_DIR], test.Path)

	loadgenConfigPath, gatewayConfigPath := path.Join(testPath, "loadgen.yml"), path.Join(testPath, "gateway.yml")

	log.Debugf("Executing gateway & loadgen within %s", testPath)
	if err := os.Chdir(testPath); err != nil {
		return nil, err
	}

	loadgenPath := appConfig.Environments[env_LR_LOADGEN_CMD]
	gatewayPath := appConfig.Environments[env_LR_GATEWAY_CMD]
	loadgenCmdArgs := []string{"-config", loadgenConfigPath}
	gatewayCmdArgs := []string{"-config", gatewayConfigPath}
	log.Debugf("Executing loadgen with args [%+v]", loadgenCmdArgs)
	log.Debugf("Executing gateway with args [%+v]", gatewayCmdArgs)
	env := generateEnv(appConfig)
	log.Debugf("Executing gateway/loadgen with environment [%+v]", env)
	loadgenCmd := exec.CommandContext(ctx, loadgenPath, loadgenCmdArgs...)
	loadgenCmd.Env = env
	gatewayCmd := exec.CommandContext(ctx, gatewayPath, gatewayCmdArgs...)
	gatewayCmd.Env = env

	gatewayFailed := int32(0)

	gatewayExited := make(chan int)
	go func() {
		output, err := gatewayCmd.Output()
		if err != nil {
			log.Debugf("gateway server exited: %+v, output: %s", err, string(output))
			log.Debugf("============================== Gateway Exit Info [Start] =============================")
			if osExit, ok := err.(*exec.ExitError); ok {
				log.Debugf(string(osExit.Stderr))
			}
			log.Debugf("============================== Gateway Exit Info [End] =============================")
			atomic.StoreInt32(&gatewayFailed, 1)
		}
		gatewayExited <- 1
	}()

	defer func() {
		log.Debug("waiting for 5s to stop the gateway")
		gatewayCmd.Process.Signal(os.Interrupt)
		timeout := time.NewTimer(5 * time.Second)
		select {
		case <-gatewayExited:
		case <-timeout.C:
		}
	}()

	gatewayReady := false

	for i := 0; i < 10; i += 1 {
		if atomic.LoadInt32(&gatewayFailed) == 1 {
			break
		}
		ncApiCmdArgs := []string{"-z"}
		ncApiCmdArgs = append(ncApiCmdArgs, strings.Split(appConfig.Environments[env_LR_GATEWAY_API_HOST], ":")...)
		ncGatewayCmdArgs := []string{"-z"}
		ncGatewayCmdArgs = append(ncGatewayCmdArgs, strings.Split(appConfig.Environments[env_LR_GATEWAY_HOST], ":")...)
		log.Debugf("Executing nc with args [%+v], [%+v]", ncApiCmdArgs, ncGatewayCmdArgs)
		ncApiCmd := exec.CommandContext(ctx, "nc", ncApiCmdArgs...)
		ncGatewayCmd := exec.CommandContext(ctx, "nc", ncGatewayCmdArgs...)
		apiErr, gatewayErr := ncApiCmd.Run(), ncGatewayCmd.Run()
		if apiErr == nil || gatewayErr == nil {
			gatewayReady = true
			break
		}
		log.Debugf("failed to probe gateway, api: %+v, gateway: %+v", apiErr, gatewayErr)
		time.Sleep(100 * time.Millisecond)
	}

	if !gatewayReady {
		return nil, errors.New("can't start gateway")
	}

	startTime := time.Now()
	testResult := &TestResult{}
	defer func() {
		testResult.DurationInMs = int64(time.Now().Sub(startTime) / time.Millisecond)
	}()

	output, err := loadgenCmd.Output()
	log.Debug("loadgen output: ", string(output))
	if err != nil {
		log.Debugf("failed to run test case, error: %+v", err)
		log.Debugf("============================== Loadgen Exit Info [Start] =============================")
		if osExit, ok := err.(*exec.ExitError); ok {
			log.Debugf(string(osExit.Stderr))
		}
		log.Debugf("============================== Loadgen Exit Info [End] =============================")
		testResult.Failed = true
	}
	return testResult, nil
}

func generateEnv(appConfig *AppConfig) (env []string) {
	for k, v := range appConfig.Environments {
		env = append(env, k+"="+v)
	}
	return
}
