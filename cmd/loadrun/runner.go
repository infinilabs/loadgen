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
		// Wait for the last process to get fully killed if not existed cleanly
		time.Sleep(time.Second)
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
			msg.Result = result.LoadgenOutput
			msg.Time = result.Time
		}
		msgs[i] = msg
	}
	ok := true
	for _, msg := range msgs {
		log.Infof("[%s][TEST][%s] [%s] duration: %d(ms)\n", msg.Time.Format("2006-01-02 15:04:05"), msg.Status, msg.Path, msg.DurationInMs)
		if msg.Status != "SUCCESS" {
			ok = false
		}
		log.Debug("============================== Loadgen Exit Info [Start] =============================")
		log.Info(util.UnsafeBytesToString(msg.Result))
		log.Debug("============================== Loadgen Exit Info [End] =============================")
	}
	return ok
}

func runTest(appConfig *AppConfig, test Test) (*TestResult, error) {
	// To kill loadgen/gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := path.Join(appConfig.Environments[env_LR_TEST_DIR], test.Path)
	loadgenPath, err := filepath.Abs(appConfig.Environments[env_LR_LOADGEN_CMD])
	if err != nil {
		return nil, err
	}

	loadgenConfigPath := path.Join(testPath, "loadgen.yml")

	log.Debugf("Executing gateway & loadgen within %s", testPath)
	if err := os.Chdir(testPath); err != nil {
		return nil, err
	}

	env := generateEnv(appConfig)
	log.Debugf("Executing gateway/loadgen with environment [%+v]", env)

	loadgenCmdArgs := []string{"-config", loadgenConfigPath, "-log", loadgenLogLevel}
	if test.Compress {
		loadgenCmdArgs = append(loadgenCmdArgs, "--compress")
	}

	if appConfig.Environments[env_LR_GATEWAY_CMD] != "" {
		gatewayConfigPath := path.Join(testPath, "gateway.yml")
		if _, err := os.Stat(gatewayConfigPath); err == nil {
			gatewayOutput := &bytes.Buffer{}
			// Start gateway server
			gatewayPath, err := filepath.Abs(appConfig.Environments[env_LR_GATEWAY_CMD])
			if err != nil {
				return nil, err
			}
			gatewayHost, gatewayApiHost := appConfig.Environments[env_LR_GATEWAY_HOST], appConfig.Environments[env_LR_GATEWAY_API_HOST]
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

	err = loadgenCmd.Run()
	if err != nil {
		log.Debugf("failed to run test case, error: %+v", err)
		testResult.Failed = true
	}
	testResult.LoadgenOutput = loadgenOutput.Bytes()
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

func generateEnv(appConfig *AppConfig) (env []string) {
	for k, v := range appConfig.Environments {
		env = append(env, k+"="+v)
	}
	// Disable greeting messages
	env = append(env, "SILENT_GREETINGS=1")
	return
}
