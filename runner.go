/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
)

type TestResult struct {
	Failed       bool      `json:"failed"`
	Time         time.Time `json:"time"`
	DurationInMs int64     `json:"duration_in_ms"`
	Error        error     `json:"error"`
}

type TestMsg struct {
	Time         time.Time `json:"time"`
	Path         string    `json:"path"`
	Status       string    `json:"status"` // ABORTED/FAILED/SUCCESS
	DurationInMs int64     `json:"duration_in_ms"`
}

const (
	portTestTimeout = 100 * time.Millisecond
)

func startRunner(config *AppConfig) bool {
	defer log.Flush()

	cwd, err := os.Getwd()
	if err != nil {
		log.Infof("failed to get working directory, err: %v", err)
		return false
	}
	msgs := make([]*TestMsg, len(config.Tests))
	for i, test := range config.Tests {
		// Wait for the last process to get fully killed if not existed cleanly
		time.Sleep(time.Second)
		result, err := runTest(config, cwd, test)
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
			msg.Time = result.Time
		}
		msgs[i] = msg
	}
	ok := true
	for _, msg := range msgs {
		log.Infof("[%s][TEST][%s] [%s] duration: %d(ms)", msg.Time.Format("2006-01-02 15:04:05"), msg.Status, msg.Path, msg.DurationInMs)
		if msg.Status != "SUCCESS" {
			ok = false
		}
	}
	return ok
}

func runTest(config *AppConfig, cwd string, test Test) (*TestResult, error) {
	// To kill gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := os.Chdir(cwd); err != nil {
		return nil, err
	}

	testPath := path.Join(config.Environments[env_LR_TEST_DIR], test.Path)
	var gatewayPath string
	if config.Environments[env_LR_GATEWAY_CMD] != "" {
		gatewayPath, _ = filepath.Abs(config.Environments[env_LR_GATEWAY_CMD])
	}

	loaderConfigPath := path.Join(testPath, "loadgen.dsl")

	log.Debugf("Executing gateway within %s", testPath)
	if err := os.Chdir(testPath); err != nil {
		return nil, err
	}
	// Revert cwd change
	defer os.Chdir(cwd)

	env := generateEnv(config)
	log.Debugf("Executing gateway with environment [%+v]", env)

	gatewayConfigPath := path.Join(testPath, "gateway.yml")
	if _, err := os.Stat(gatewayConfigPath); err == nil {
		if gatewayPath == "" {
			return nil, errors.New("invalid LR_GATEWAY_CMD, cannot find gateway")
		}
		gatewayOutput := &bytes.Buffer{}
		// Start gateway server
		gatewayHost, gatewayApiHost := config.Environments[env_LR_GATEWAY_HOST], config.Environments[env_LR_GATEWAY_API_HOST]
		gatewayCmd, gatewayExited, err := runGateway(ctx, gatewayPath, gatewayConfigPath, gatewayHost, gatewayApiHost, env, gatewayOutput)
		if err != nil {
			return nil, err
		}

		defer func() {
			log.Debug("waiting for 5s to stop the gateway")
			gatewayCmd.Process.Signal(os.Interrupt)
			timeout := time.NewTimer(5 * time.Second)
			select {
			case <-gatewayExited:
			case <-timeout.C:
			}
			log.Debug("============================== Gateway Exit Info [Start] =============================")
			log.Debug(util.UnsafeBytesToString(gatewayOutput.Bytes()))
			log.Debug("============================== Gateway Exit Info [End] =============================")
		}()
	}

	startTime := time.Now()
	testResult := &TestResult{}
	defer func() {
		testResult.Time = time.Now()
		testResult.DurationInMs = int64(testResult.Time.Sub(startTime) / time.Millisecond)
	}()

	status := runDSL(loaderConfigPath)
	if status != 0 {
		testResult.Failed = true
	}
	return testResult, nil
}

func runGateway(ctx context.Context, gatewayPath, gatewayConfigPath, gatewayHost, gatewayApiHost string, env []string, gatewayOutput *bytes.Buffer) (*exec.Cmd, chan int, error) {
	gatewayCmdArgs := []string{"-config", gatewayConfigPath, "-log", gatewayLogLevel}
	log.Debugf("Executing gateway with args [%+v]", gatewayCmdArgs)
	gatewayCmd := exec.CommandContext(ctx, gatewayPath, gatewayCmdArgs...)
	gatewayCmd.Env = env
	gatewayCmd.Stdout = gatewayOutput
	gatewayCmd.Stderr = gatewayOutput

	gatewayFailed := int32(0)
	gatewayExited := make(chan int)

	go func() {
		err := gatewayCmd.Run()
		if err != nil {
			log.Debugf("gateway server exited with non-zero code: %+v", err)
			atomic.StoreInt32(&gatewayFailed, 1)
		}
		gatewayExited <- 1
	}()

	gatewayReady := false

	// Check whether gateway is ready.
	for i := 0; i < 10; i += 1 {
		if atomic.LoadInt32(&gatewayFailed) == 1 {
			break
		}
		log.Debugf("Checking whether %s or %s is ready...", gatewayHost, gatewayApiHost)
		entryReady, apiReady := testPort(gatewayHost), testPort(gatewayApiHost)
		if entryReady || apiReady {
			log.Debugf("gateway is started, entry: %+v, api: %+v", entryReady, apiReady)
			gatewayReady = true
			break
		}
		log.Debugf("failed to probe gateway, retrying")
		time.Sleep(100 * time.Millisecond)
	}

	if !gatewayReady {
		return nil, nil, errors.New("can't start gateway")
	}

	return gatewayCmd, gatewayExited, nil
}

func testPort(host string) bool {
	conn, err := net.DialTimeout("tcp", host, portTestTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func generateEnv(config *AppConfig) (env []string) {
	for k, v := range config.Environments {
		env = append(env, k+"="+v)
	}
	// Disable greeting messages
	env = append(env, "SILENT_GREETINGS=1")
	return
}
