package main

import (
	"path/filepath"

	log "github.com/cihub/seelog"
)

type AppConfig struct {
	Environment *Environment `config:"environment"`
	Tests       []Test       `config:"tests"`
}

func (cfg *AppConfig) Init() {
	if cfg.Environment == nil || cfg.Environment.Elasticsearch == nil || cfg.Environment.Gateway == nil || cfg.Environment.Loadgen == nil {
		panic("invalid environment config")
	}
	if cfg.Environment.Elasticsearch.Endpoint == "" {
		panic("invalid es config")
	}
	if cfg.Environment.Gateway.Command == "" || cfg.Environment.Gateway.Endpoint == "" {
		panic("invalid gateway config")
	}
	if cfg.Environment.Loadgen.Command == "" {
		panic("invalid loadgen config")
	}
	if cfg.Environment.TestDir == "" {
		cfg.Environment.TestDir = "./testing"
	}
	fullTestDir, err := filepath.Abs(cfg.Environment.TestDir)
	if err != nil {
		log.Warnf("failed to get the abs path of test_dir, error: %+v", err)
	} else {
		cfg.Environment.TestDir = fullTestDir
	}
}

type Environment struct {
	Elasticsearch *Elasticsearch `config:"elasticsearch"`
	Gateway       *Gateway       `config:"gateway"`
	Loadgen       *Loadgen       `config:"loadgen"`

	// The root directory of all test cases, will automatically convert to absolute path
	TestDir string `config:"test_dir"`
}

type Elasticsearch struct {
	// ES server endpoint, available in template as $[[elasticsearch.endpoint]]
	Endpoint string `config:"endpoint"`
}

type Gateway struct {
	// The command to start gateway server
	Command string `config:"command"`
	// Gateway server endpoint, available in template as $[[gateway.endpoint]]
	Endpoint string `config:"endpoint"`
}

type Loadgen struct {
	// The command to start loadgen
	Command string `config:"command"`
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
