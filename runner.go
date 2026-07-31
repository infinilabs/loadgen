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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	coreConfig "infini.sh/framework/core/config"
	"infini.sh/framework/core/util"
)

type TestResult struct {
	Failed       bool      `json:"failed"`
	Time         time.Time `json:"time"`
	DurationInMs int64     `json:"duration_in_ms"`
	Error        error     `json:"error"`
}

type TestMsg struct {
	Time   time.Time `json:"time"`
	Path   string    `json:"path"`
	Status string    `json:"status"` // ABORTED/FAILED/SUCCESS
	// Why this test abortd, non-empty if Status is aborted.
	AbortMsg     string `json:"abort_msg"`
	DurationInMs int64  `json:"duration_in_ms"`
}

const (
	portTestTimeout = 100 * time.Millisecond
)

func startRunner(config *AppConfig) bool {
	defer log.Flush()

	msgs := make([]*TestMsg, len(config.Tests))
	for i, test := range config.Tests {
		// Wait for the last process to get fully killed if not existed cleanly
		time.Sleep(time.Second)
		result, err := runTest(config, test)
		msg := &TestMsg{
			Path: test.Path,
		}
		if result == nil || err != nil {
			log.Debugf("failed to run test, error: %+v", err)
			msg.Status = "ABORTED"
			msg.AbortMsg = err.Error()
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
		detailedStatus := msg.Status
		if msg.AbortMsg != "" {
			detailedStatus = fmt.Sprintf("%s (%s)", msg.Status, msg.AbortMsg)
		}

		log.Infof("[%s][TEST][%s] [%s] duration: %d(ms)", msg.Time.Format("2006-01-02 15:04:05"), detailedStatus, msg.Path, msg.DurationInMs)
		if msg.Status != "SUCCESS" {
			ok = false
		}
	}
	return ok
}

func runTest(config *AppConfig, test Test) (*TestResult, error) {
	// To kill gateway/other command automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := test.Path
	var gatewayPath string
	if config.Environments[env_LR_GATEWAY_CMD] != "" {
		gatewayPath, _ = filepath.Abs(config.Environments[env_LR_GATEWAY_CMD])
	}

	/*
	 * Pick the test file, it could be either:
	 * 1. loadgen.dsl
	 * 2. loafgen.yml
	 *
	 * If both exist, DSL is preferred.
	 */
	var loaderConfigPath string
	ymlFilePath := path.Join(testPath, "loadgen.yml")
	dslFilePath := path.Join(testPath, "loadgen.dsl")
	ymlFileExists := util.FileExists(ymlFilePath)
	dslFileExists := util.FileExists(dslFilePath)

	if dslFileExists {
		loaderConfigPath = dslFilePath
	} else if ymlFileExists {
		loaderConfigPath = ymlFilePath
	} else {
		return nil, fmt.Errorf("no loadgen test file found under %s, expected a loadgen.dsl or loadgen.yml", testPath)
	}
	loaderConfigPath, _ = filepath.Abs(loaderConfigPath)

	// A gateway.yml is optional: when present, the gateway is started
	// dynamically for this test; otherwise the test runs without a gateway.
	gatewayConfigPath := path.Join(testPath, "gateway.yml")
	if _, err := os.Stat(gatewayConfigPath); err == nil {
		if gatewayPath == "" {
			return nil, errors.New("invalid LR_GATEWAY_CMD, cannot find gateway")
		}

		// Start gateway server
		gatewayOutput := &bytes.Buffer{}
		probeAddrs, err := parseGatewayListenAddrs(gatewayConfigPath)
		if err != nil {
			return nil, err
		}
		env := generateEnv(config)
		log.Debugf("Executing gateway with environment [%+v]", env)
		gatewayCmd, gatewayExited, err := runGateway(ctx, gatewayPath, probeAddrs, testPath, env, gatewayOutput)
		if err != nil {
			return nil, err
		}

		defer func() {
			log.Debug("waiting for 5s to stop the gateway")
			if gatewayCmd != nil && gatewayCmd.Process != nil {
				gatewayCmd.Process.Signal(os.Interrupt)
				timeout := time.NewTimer(5 * time.Second)
				select {
				case <-gatewayExited:
				case <-timeout.C:
				}
			}
			log.Debug("============================== Gateway Exit Info [Start] =============================")
			log.Debug(util.UnsafeBytesToString(gatewayOutput.Bytes()))
			log.Debug("============================== Gateway Exit Info [End] =============================")
		}()
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	startTime := time.Now()
	testResult := &TestResult{}
	defer func() {
		testResult.Time = time.Now()
		testResult.DurationInMs = int64(testResult.Time.Sub(startTime) / time.Millisecond)
	}()

	status := 0
	if strings.HasSuffix(loaderConfigPath, ".dsl") {
		status = runDSLFile(config, loaderConfigPath)
	} else {
		status = runYAMLFile(config, loaderConfigPath)
	}
	if status != 0 {
		testResult.Failed = true
	}
	return testResult, nil
}

func runGateway(ctx context.Context, gatewayPath string, probeAddrs []string, workingDir string, env []string, gatewayOutput *bytes.Buffer) (*exec.Cmd, chan int, error) {
	gatewayCmdArgs := []string{"-log", "debug"}
	log.Debugf("Executing gateway with args [%+v]", gatewayCmdArgs)
	gatewayCmd := exec.CommandContext(ctx, gatewayPath, gatewayCmdArgs...)
	gatewayCmd.Dir = workingDir
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

	// Check whether gateway is ready: every configured listener must accept TCP connections.
	for i := 0; i < 100; i += 1 {
		if atomic.LoadInt32(&gatewayFailed) == 1 {
			break
		}
		allReady := true
		for _, addr := range probeAddrs {
			if !testPort(addr) {
				log.Debugf("gateway %s is not ready yet", addr)
				allReady = false
				break
			}
		}
		if allReady {
			log.Debugf("gateway is started, listening on %v", probeAddrs)
			gatewayReady = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	if !gatewayReady {
		return nil, nil, fmt.Errorf("can't start gateway, output: %s", util.UnsafeBytesToString(gatewayOutput.Bytes()))
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

// Helper function to build the environment variables passed to the gateway
// subprocess. It forwards every variable defined in the loadgen config and
// appends SILENT_GREETINGS to suppress greeting messages.
func generateEnv(config *AppConfig) (env []string) {
	for k, v := range config.Environments {
		env = append(env, k+"="+v)
	}
	// Disable greeting messages
	env = append(env, "SILENT_GREETINGS=1")
	return
}

// gatewayProbeConfig mirrors the network listener sections of a gateway.yml.
// Only the fields needed to locate listening addresses are declared.
type gatewayProbeConfig struct {
	API   gatewayAPIProbeConfig     `config:"api"`
	Entry []gatewayEntryProbeConfig `config:"entry"`
}

type gatewayAPIProbeConfig struct {
	Enabled bool                 `config:"enabled"`
	Network gatewayNetworkConfig `config:"network"`
}

type gatewayEntryProbeConfig struct {
	Enabled bool                 `config:"enabled"`
	Network gatewayNetworkConfig `config:"network"`
}

type gatewayNetworkConfig struct {
	Binding string `config:"binding"`
	Host    string `config:"host"`
	Port    int    `config:"port"`
}

// Helper function to collect the listening addresses declared by an api
// section and all enabled entries into the given set.
func (c *gatewayProbeConfig) collectAddrs(addrs map[string]struct{}) {
	if c.API.Enabled {
		if addr := c.API.Network.addr(); addr != "" {
			addrs[addr] = struct{}{}
		}
	}
	for _, entry := range c.Entry {
		if !entry.Enabled {
			continue
		}
		if addr := entry.Network.addr(); addr != "" {
			addrs[addr] = struct{}{}
		}
	}
}

// Helper function to resolve the listening address following the gateway's
// own precedence: an explicit `binding` (host:port) wins over separate
// `host` and `port` values.
func (n gatewayNetworkConfig) addr() string {
	if n.Binding != "" {
		return n.Binding
	}
	if n.Host != "" || n.Port != 0 {
		return net.JoinHostPort(n.Host, strconv.Itoa(n.Port))
	}
	return ""
}

// Helper function to parse a gateway.yml and return every address the gateway
// is expected to listen on: the api listener plus all enabled entries.
// The config is loaded via the framework's config loader, so `$[[env.X]]`
// templates and `configs.template` references are resolved exactly the same
// way the gateway binary resolves them.
//
// Template paths in gateway.yml (e.g. ./config_template.tpl) are resolved
// against the process working directory, which is the test directory at
// gateway runtime, so the working directory is switched accordingly.
func parseGatewayListenAddrs(gatewayConfigPath string) ([]string, error) {
	// Resolve to an absolute path first: the working directory is switched
	// below, which would otherwise break a relative gatewayConfigPath.
	gatewayConfigPath, err := filepath.Abs(gatewayConfigPath)
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if dir := filepath.Dir(gatewayConfigPath); dir != cwd {
		if err := os.Chdir(dir); err != nil {
			return nil, err
		}
		defer os.Chdir(cwd)
	}

	cfg, err := coreConfig.LoadFile(gatewayConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load gateway config %s, err: %v", gatewayConfigPath, err)
	}

	// Pre-populate the framework defaults: api is enabled and listens on
	// 0.0.0.0:2900 unless the config says otherwise.
	probeCfg := gatewayProbeConfig{
		API: gatewayAPIProbeConfig{
			Enabled: true,
			Network: gatewayNetworkConfig{Binding: "0.0.0.0:2900"},
		},
	}
	if err := cfg.Unpack(&probeCfg); err != nil {
		return nil, fmt.Errorf("failed to unpack gateway config %s, err: %v", gatewayConfigPath, err)
	}

	addrs := map[string]struct{}{}
	probeCfg.collectAddrs(addrs)

	var result []string
	for addr := range addrs {
		result = append(result, addr)
	}

	// A wildcard binding (IPv4 0.0.0.0, IPv6 ::, or an omitted host) is not
	// dialable from outside the process; probe it via loopback instead.
	for i, addr := range result {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			log.Warnf("failed to parse gateway probe address %q: %v", addr, err)
			continue
		}
		if host == "" || host == "0.0.0.0" || host == "::" {
			result[i] = net.JoinHostPort("127.0.0.1", port)
		}
	}
	return result, nil
}
