package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"github.com/valyala/fasttemplate"
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

func startRunner(appConfig *AppConfig) {
	envMap := generateEnvironmentMap(appConfig)
	msgs := make([]*TestMsg, len(appConfig.Tests))
	for i, test := range appConfig.Tests {
		result, err := runTest(appConfig, envMap, test)
		msg := &TestMsg{
			Path: test.Path,
		}
		if result == nil || err != nil {
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
	for _, msg := range msgs {
		log.Infof("[TEST][%s] [%s] duration: %d ", msg.Path, msg.Status, msg.DurationInMs)
	}
}

func runTest(appConfig *AppConfig, environmentMap map[string]interface{}, test Test) (*TestResult, error) {
	// To kill loadgen/gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := path.Join(appConfig.Environment.TestDir, test.Path)

	loadgenConfigPath, gatewayConfigPath, err := generateConfig(testPath, environmentMap)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.Remove(loadgenConfigPath); err != nil {
			log.Warn("failed to clean loadgen tempfile: %+v", err)
		}
		if err := os.Remove(gatewayConfigPath); err != nil {
			log.Warn("failed to clean gateway tempfile: %+v", err)
		}
	}()

	log.Debugf("Executing gateway & loadgen within %s", testPath)
	if err := os.Chdir(testPath); err != nil {
		return nil, err
	}

	loadgenPath := appConfig.Environment.Loadgen.Command
	gatewayPath := appConfig.Environment.Gateway.Command
	loadgenCmdArgs := []string{"-config", loadgenConfigPath}
	gatewayCmdArgs := []string{"-config", gatewayConfigPath}
	log.Debugf("Executing loadgen with args [%+v]", loadgenCmdArgs)
	log.Debugf("Executing gateway with args [%+v]", gatewayCmdArgs)
	loadgenCmd := exec.CommandContext(ctx, loadgenPath, loadgenCmdArgs...)
	gatewayCmd := exec.CommandContext(ctx, gatewayPath, gatewayCmdArgs...)

	gatewayFailed := atomic.Bool{}

	go func() {
		output, err := gatewayCmd.Output()
		if err != nil {
			log.Debugf("gateway server exited: %+v, output: %s", err, string(output))
			gatewayFailed.Store(true)
		}
	}()

	gatewayReady := false

	for i := 0; i < 10; i += 1 {
		if gatewayFailed.Load() {
			break
		}
		ncCmd := exec.CommandContext(ctx, "nc", "-z", "localhost", "8000")
		err := ncCmd.Run()
		if err != nil {
			log.Debugf("failed to probe gateway: %+v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		gatewayReady = true
		break
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
		if osExit, ok := err.(*exec.ExitError); ok {
			log.Debugf(string(osExit.Stderr))
		}
		testResult.Failed = true
	}
	return testResult, nil
}

func generateEnvironmentMap(appConfig *AppConfig) map[string]interface{} {
	envMap := map[string]interface{}{}
	env := appConfig.Environment
	envMap["elasticsearch.endpoint"] = env.Elasticsearch.Endpoint
	envMap["gateway.endpoint"] = env.Gateway.Endpoint
	return envMap
}

func generateConfig(testPath string, environmentMap map[string]interface{}) (loadgenConfigPath string, gatewayConfigPath string, err error) {
	loadgenConfigTmplPath := path.Join(testPath, "loadgen.yml")
	gatewayConfigTmplPath := path.Join(testPath, "gateway.yml")

	if _, err := os.Stat(loadgenConfigTmplPath); errors.Is(err, os.ErrNotExist) {
		return "", "", errors.New("missing loadgen.yml")
	}
	if _, err := os.Stat(gatewayConfigTmplPath); errors.Is(err, os.ErrNotExist) {
		return "", "", errors.New("missing gateway.yml")
	}

	loadgenConfigTmplStr, err := os.ReadFile(loadgenConfigTmplPath)
	if err != nil {
		return "", "", err
	}
	gatewayConfigTmplStr, err := os.ReadFile(gatewayConfigTmplPath)
	if err != nil {
		return "", "", err
	}
	loadgenConfigTmpl, err := fasttemplate.NewTemplate(string(loadgenConfigTmplStr), "${{", "}}")
	if err != nil {
		return "", "", err
	}
	gatewayConfigTmpl, err := fasttemplate.NewTemplate(string(gatewayConfigTmplStr), "${{", "}}")
	if err != nil {
		return "", "", err
	}
	prefix := filepath.Base(testPath)
	loadgenConfigFile, err := os.CreateTemp("", prefix+"-loadgen-*.yaml")
	if err != nil {
		return "", "", err
	}
	defer loadgenConfigFile.Close()
	log.Warn(loadgenConfigFile.Name())
	gatewayConfigFile, err := os.CreateTemp("", prefix+"-gateway-*.yaml")
	if err != nil {
		return "", "", err
	}
	defer gatewayConfigFile.Close()

	_, err = loadgenConfigTmpl.Execute(loadgenConfigFile, environmentMap)
	if err != nil {
		return "", "", err
	}
	_, err = gatewayConfigTmpl.Execute(gatewayConfigFile, environmentMap)
	if err != nil {
		return "", "", err
	}

	loadgenConfigPath = loadgenConfigFile.Name()
	gatewayConfigPath = gatewayConfigFile.Name()
	return
}
