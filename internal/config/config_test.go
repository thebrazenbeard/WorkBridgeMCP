package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefaultsAndRejectsUnknownFields(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	payload, _ := json.Marshal(map[string]any{
		"schema":      Schema,
		"read_roots":  []string{root},
		"write_roots": []string{},
		"limits":      map[string]any{},
		"process":     map[string]any{"enabled": false},
		"http":        map[string]any{},
	})
	cfg, err := Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Listen != "127.0.0.1:8765" || cfg.HTTP.Path != "/mcp" ||
		cfg.HTTP.BearerTokenEnv != "WORKBRIDGE_HTTP_TOKEN" {
		t.Fatalf("defaults not applied: %#v", cfg.HTTP)
	}
	_, err = Parse([]byte(`{"schema":"WORKBRIDGE_CONFIG_V1","read_roots":[],"write_roots":[],"limits":{},"process":{"enabled":false},"http":{},"surprise":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field accepted: %v", err)
	}
}

func TestRejectsNonLoopbackHTTPAndRelativeProcessExecutable(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	cfg := Config{
		Schema: Schema,
		Process: ProcessConfig{
			Enabled:            true,
			AllowedExecutables: []string{"cmd.exe"},
			WorkingRoots:       []string{root},
			MaxRuntimeSeconds:  10,
			MaxOutputBytes:     1024,
		},
		Limits: Limits{MaxReadBytes: 1024, MaxWriteBytes: 1024, MaxDirectoryEntries: 10},
		HTTP: HTTPConfig{
			Listen:         "0.0.0.0:8765",
			Path:           "/mcp",
			BearerTokenEnv: "WORKBRIDGE_HTTP_TOKEN",
		},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("non-loopback listen accepted: %v", err)
	}
	cfg.HTTP.Listen = "127.0.0.1:8765"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative executable accepted: %v", err)
	}
}

func TestRejectsInvalidHTTPTokenEnvName(t *testing.T) {
	cfg, err := Parse([]byte(`{
		"schema":"WORKBRIDGE_CONFIG_V1",
		"read_roots":[],
		"write_roots":[],
		"limits":{},
		"process":{"enabled":false},
		"http":{"bearer_token_env":"BAD-NAME"}
	}`))
	if err == nil || cfg != nil || !strings.Contains(err.Error(), "environment-variable") {
		t.Fatalf("invalid env name accepted: cfg=%#v err=%v", cfg, err)
	}
}
