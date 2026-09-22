package mcpserver

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/bridge"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func TestServerCapabilityMinimumExposesOnlyInfo(t *testing.T) {
	cfg := &config.Config{
		Schema: config.Schema,
		Limits: config.Limits{
			MaxReadBytes:        1024,
			MaxWriteBytes:       1024,
			MaxDirectoryEntries: 10,
		},
		Process: config.ProcessConfig{MaxRuntimeSeconds: 1, MaxOutputBytes: 1024},
		HTTP: config.HTTPConfig{
			Listen:         "127.0.0.1:8765",
			Path:           "/mcp",
			BearerTokenEnv: "WORKBRIDGE_HTTP_TOKEN",
		},
	}
	b, err := bridge.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	assertTools(t, New(b, "test"), []string{"workstation_info"})
}

func TestServerRegistersReadWriteToolsButNotProcessWhenDisabled(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	cfg := &config.Config{
		Schema:     config.Schema,
		ReadRoots:  []string{root},
		WriteRoots: []string{root},
		Limits: config.Limits{
			MaxReadBytes:        1024,
			MaxWriteBytes:       1024,
			MaxDirectoryEntries: 10,
		},
		Process: config.ProcessConfig{MaxRuntimeSeconds: 1, MaxOutputBytes: 1024},
		HTTP: config.HTTPConfig{
			Listen:         "127.0.0.1:8765",
			Path:           "/mcp",
			BearerTokenEnv: "WORKBRIDGE_HTTP_TOKEN",
		},
	}
	b, err := bridge.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	assertTools(t, New(b, "test"), []string{
		"list_directory", "make_directory", "read_bytes",
		"read_text", "stat_path", "workstation_info", "write_text",
	})
}

func assertTools(t *testing.T, server *mcp.Server, want []string) {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "workbridge-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("tools=%v want=%v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("tools=%v want=%v", got, want)
		}
	}
}
