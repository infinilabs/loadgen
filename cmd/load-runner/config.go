package main

import (
	"path/filepath"

	log "github.com/cihub/seelog"
)

type AppConfig struct {
	/*
		Required environments:
		- LR_ELASTICSEARCH_ENDPOINT // ES server endpoint
		- LR_GATEWAY_HOST // Gateway server host
		- LR_GATEWAY_CMD // The command to start gateway server
		- LR_LOADGEN_CMD // The command to start loadgen
		Optional environments:
		- LR_TEST_DIR    // The root directory of all test cases, will automatically convert to absolute path. Default: ./testing
		- LR_GATEWAY_API_HOST // Gateway server api binding host
	*/
	Environments map[string]string `config:"env"`
	Tests        []Test            `config:"tests"`
}

const (
	env_LR_ELASTICSEARCH_ENDPOINT = "LR_ELASTICSEARCH_ENDPOINT"
	env_LR_GATEWAY_HOST           = "LR_GATEWAY_HOST"
	env_LR_GATEWAY_CMD            = "LR_GATEWAY_CMD"
	env_LR_LOADGEN_CMD            = "LR_LOADGEN_CMD"
	env_LR_TEST_DIR               = "LR_TEST_DIR"
	env_LR_GATEWAY_API_HOST       = "LR_GATEWAY_API_HOST"
)

func (cfg *AppConfig) Init() {
	if !cfg.testEnv(env_LR_ELASTICSEARCH_ENDPOINT, env_LR_GATEWAY_HOST, env_LR_GATEWAY_CMD, env_LR_LOADGEN_CMD) {
		panic("invalid environment configurations")
	}
	if cfg.Environments[env_LR_TEST_DIR] == "" {
		cfg.Environments[env_LR_TEST_DIR] = "./testing"
	}
	fullTestDir, err := filepath.Abs(cfg.Environments[env_LR_TEST_DIR])
	if err != nil {
		log.Warnf("failed to get the abs path of test_dir, error: %+v", err)
	} else {
		cfg.Environments[env_LR_TEST_DIR] = fullTestDir
	}
}

func (cfg *AppConfig) testEnv(envVars ...string) bool {
	for _, envVar := range envVars {
		if v, ok := cfg.Environments[envVar]; !ok || v == "" {
			return false
		}
	}
	return true
}

/*
A test case is a standalone directory containing the following configs:
- gateway.yml: The configuration to start the gateway server
- loadgen.yml: The configuration to define the test cases
*/
type Test struct {
	// The directory of the test configurations
	Path string `config:"path"`
}
