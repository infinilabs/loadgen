package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
)

type TestResult struct {
	Failed        bool      `json:"failed"`
	Time          time.Time `json:"time"`
	DurationInMs  int64     `json:"duration_in_ms"`
	Error         error     `json:"error"`
	LoadgenOutput []byte    `json:"loadgen_output"`
}

type TestMsg struct {
	Time         time.Time `json:"time"`
	Path         string    `json:"path"`
	Status       string    `json:"status"` // ABORTED/FAILED/SUCCESS
	DurationInMs int64     `json:"duration_in_ms"`
	Result       []byte    `json:"result"`
}

const (
	portTestTimeout = 100 * time.Millisecond
)

func startRunner(appConfig *AppConfig) bool {
	defer log.Flush()
	msgs := make([]*TestMsg, len(appConfig.Tests))
	for i, test := range appConfig.Tests {
		result, err := runTest(appConfig, test)
		msg := &TestMsg{
			Path:   test.Path,
			Result: result.LoadgenOutput,
			Time:   result.Time,
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
		fmt.Printf("[%s][TEST][%s] [%s] duration: %d(ms)\n", msg.Time.Format("2006-01-02 15:04:05"), msg.Status, msg.Path, msg.DurationInMs)
		if msg.Status != "SUCCESS" {
			ok = false
		}
		fmt.Println(string(msg.Result))
	}
	return ok
}

func runTest(appConfig *AppConfig, test Test) (*TestResult, error) {
	// To kill loadgen/gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := path.Join(appConfig.Environments[env_LR_TEST_DIR], test.Path)

	loadgenConfigPath := path.Join(testPath, "loadgen.yml")

	log.Debugf("Executing gateway & loadgen within %s", testPath)
	if err := os.Chdir(testPath); err != nil {
		return nil, err
	}

	env := generateEnv(appConfig)
	log.Debugf("Executing gateway/loadgen with environment [%+v]", env)

	loadgenPath := appConfig.Environments[env_LR_LOADGEN_CMD]
	loadgenCmdArgs := []string{"-config", loadgenConfigPath, "-log", "off", "-silent"}

	if appConfig.Environments[env_LR_GATEWAY_CMD] != "" {
		gatewayConfigPath := path.Join(testPath, "gateway.yml")
		if _, err := os.Stat(gatewayConfigPath); err == nil {
			// Start gateway server
			gatewayPath, gatewayHost, gatewayApiHost := appConfig.Environments[env_LR_GATEWAY_CMD], appConfig.Environments[env_LR_GATEWAY_HOST], appConfig.Environments[env_LR_GATEWAY_API_HOST]
			gatewayCmd, gatewayExited, err := runGateway(ctx, gatewayPath, gatewayConfigPath, gatewayHost, gatewayApiHost, env)
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
			}()
		}
	}

	log.Debugf("Executing loadgen with args [%+v]", loadgenCmdArgs)
	loadgenOutput := &bytes.Buffer{}
	loadgenCmd := exec.CommandContext(ctx, loadgenPath, loadgenCmdArgs...)
	loadgenCmd.Stdout = loadgenOutput
	loadgenCmd.Stderr = loadgenOutput
	loadgenCmd.Env = env

	startTime := time.Now()
	testResult := &TestResult{}
	defer func() {
		testResult.Time = time.Now()
		testResult.DurationInMs = int64(testResult.Time.Sub(startTime) / time.Millisecond)
	}()

	err := loadgenCmd.Run()
	if err != nil {
		log.Debugf("failed to run test case, error: %+v", err)
		testResult.Failed = true
	}
	testResult.LoadgenOutput = loadgenOutput.Bytes()
	return testResult, nil
}

func runGateway(ctx context.Context, gatewayPath, gatewayConfigPath, gatewayHost, gatewayApiHost string, env []string) (*exec.Cmd, chan int, error) {
	gatewayCmdArgs := []string{"-config", gatewayConfigPath}
	log.Debugf("Executing gateway with args [%+v]", gatewayCmdArgs)
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

	gatewayReady := false

	// Check whether gateway is ready.
	for i := 0; i < 10; i += 1 {
		if atomic.LoadInt32(&gatewayFailed) == 1 {
			break
		}
		log.Debugf("Checking whether %s or %s is ready...", gatewayHost, gatewayApiHost)
		if testPort(gatewayHost) || testPort(gatewayApiHost) {
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

func generateEnv(appConfig *AppConfig) (env []string) {
	for k, v := range appConfig.Environments {
		env = append(env, k+"="+v)
	}
	return
}