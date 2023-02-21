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
		- LR_MINIO_API_HOST // minio server host
		- LR_MINIO_API_USERNAME // minio server username
		- LR_MINIO_API_PASSWORD // minio server password
		- LR_MINIO_TEST_BUCKET // minio testing bucket, need to set as public access
		- LR_GATEWAY_FLOATING_IP_HOST // Gateway server floating IP host
	*/
	Environments map[string]string `config:"env"`
	Tests        []Test            `config:"tests"`
}

const (
	// Required configurations
	env_LR_LOADGEN_CMD = "LR_LOADGEN_CMD"
	env_LR_TEST_DIR    = "LR_TEST_DIR"

	// Gateway-related configurations
	env_LR_GATEWAY_CMD      = "LR_GATEWAY_CMD"
	env_LR_GATEWAY_HOST     = "LR_GATEWAY_HOST"
	env_LR_GATEWAY_API_HOST = "LR_GATEWAY_API_HOST"

	// Standard environments used by `testing` repo
	env_LR_ELASTICSEARCH_ENDPOINT   = "LR_ELASTICSEARCH_ENDPOINT"
	env_LR_MINIO_API_HOST           = "LR_MINIO_API_HOST"
	env_LR_MINIO_API_USERNAME       = "LR_MINIO_API_USERNAME"
	env_LR_MINIO_API_PASSWORD       = "LR_MINIO_API_PASSWORD"
	env_LR_GATEWAY_FLOATING_IP_HOST = "LR_GATEWAY_FLOATING_IP_HOST"
	env_LR_MINIO_TEST_BUCKET        = "LR_MINIO_TEST_BUCKET"
)

func (cfg *AppConfig) Init() {
	if !cfg.testEnv(env_LR_LOADGEN_CMD) {
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
