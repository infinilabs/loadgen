package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeGatewayConfigFiles materializes a main gateway.yml plus a
// referenced config_template.tpl under a temp directory. The returned
// directory is the temp root (suitable for t.Chdir) and the absolute path
// to the main config.
func writeGatewayConfigFiles(t *testing.T, mainYAML, tplYAML string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "gateway.yml")
	tplPath := filepath.Join(dir, "config_template.tpl")
	if err := os.WriteFile(mainPath, []byte(mainYAML), 0o644); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}
	if err := os.WriteFile(tplPath, []byte(tplYAML), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	return dir, mainPath
}

// Both the api listener (default 0.0.0.0:2900) and the entry contributed by
// configs.template should be probed once the template has been inlined.
func TestParseGatewayListenAddrsWithConfigTemplate(t *testing.T) {
	const mainYAML = `env:
  BINDING_HOST: 127.0.0.1:8001

configs.template:
  - name: "test_entry"
    path: ./config_template.tpl
    variable:
      name: "test_entry"
      binding_host: $[[env.BINDING_HOST]]
`
	const tplYAML = `entry:
  - name: my_entry_$[[name]]
    enabled: true
    network:
      binding: $[[binding_host]]
`
	dir, mainPath := writeGatewayConfigFiles(t, mainYAML, tplYAML)
	t.Chdir(dir)

	addrs, err := parseGatewayListenAddrs(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("probe addrs: %v", addrs)

	want := map[string]bool{
		"127.0.0.1:8001": false,
		"127.0.0.1:2900": false,
	}
	for _, addr := range addrs {
		if _, ok := want[addr]; ok {
			want[addr] = true
		}
	}
	for addr, found := range want {
		if !found {
			t.Errorf("expected probe address %s not found in %v", addr, addrs)
		}
	}
}

// A plain gateway.yml with an api listener and an entry, no $[[ templates.
func TestParseGatewayListenAddrsPlain(t *testing.T) {
	const mainYAML = `api:
  enabled: true
  network:
    binding: 127.0.0.1:9000

entry:
  - name: my_es_entry
    enabled: true
    network:
      binding: 127.0.0.1:8001
`
	dir, mainPath := writeGatewayConfigFiles(t, mainYAML, "")
	t.Chdir(dir)

	addrs, err := parseGatewayListenAddrs(mainPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("probe addrs: %v", addrs)

	want := map[string]bool{
		"127.0.0.1:8001": false,
		"127.0.0.1:9000": false,
	}
	for _, addr := range addrs {
		if _, ok := want[addr]; ok {
			want[addr] = true
		}
	}
	for addr, found := range want {
		if !found {
			t.Errorf("expected probe address %s not found in %v", addr, addrs)
		}
	}
}

// A test suite references gateway.yml with a path relative to loadgen's
// working directory; make sure the internal chdir does not break it.
func TestParseGatewayListenAddrsRelativePath(t *testing.T) {
	const mainYAML = `entry:
  - name: my_es_entry
    enabled: true
    network:
      binding: 127.0.0.1:8001
`
	dir, _ := writeGatewayConfigFiles(t, mainYAML, "")
	t.Chdir(dir)

	addrs, err := parseGatewayListenAddrs("gateway.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("probe addrs: %v", addrs)

	want := map[string]bool{
		"127.0.0.1:8001": false,
	}
	for _, addr := range addrs {
		if _, ok := want[addr]; ok {
			want[addr] = true
		}
	}
	for addr, found := range want {
		if !found {
			t.Errorf("expected probe address %s not found in %v", addr, addrs)
		}
	}
}
