package mcpserver

import (
	"testing"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/bridge"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func TestServerBuildsAtCapabilityMinimum(t *testing.T) {
	cfg := &config.Config{
		Schema: config.Schema,
		Limits: config.Limits{
			MaxReadBytes:        1024,
			MaxWriteBytes:       1024,
			MaxDirectoryEntries: 10,
		},
		Process: config.ProcessConfig{MaxRuntimeSeconds: 1, MaxOutputBytes: 1024},
		HTTP:    config.HTTPConfig{Listen: "127.0.0.1:8765", Path: "/mcp"},
	}
	b, err := bridge.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if s := New(b, "test"); s == nil {
		t.Fatal("nil MCP server")
	}
}
