package bridge

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func testConfig(root string) *config.Config {
	return &config.Config{
		Schema:     config.Schema,
		ReadRoots:  []string{root},
		WriteRoots: []string{root},
		Limits: config.Limits{
			MaxReadBytes:        1024,
			MaxWriteBytes:       1024,
			MaxDirectoryEntries: 100,
		},
		Process: config.ProcessConfig{
			Enabled:           false,
			MaxRuntimeSeconds: 2,
			MaxOutputBytes:    1024,
		},
		HTTP: config.HTTPConfig{Listen: "127.0.0.1:8765", Path: "/mcp"},
	}
}

func TestReadWriteAndContainment(t *testing.T) {
	root := t.TempDir()
	b, err := New(testConfig(root))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	path := filepath.Join(root, "note.txt")
	if _, err := b.WriteText(path, "hello", false); err != nil {
		t.Fatal(err)
	}
	got, err := b.ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["content"] != "hello" {
		t.Fatalf("content=%v", got["content"])
	}
	if _, err := b.WriteText(path, "again", false); err == nil {
		t.Fatal("overwrite without permission accepted")
	}
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outside, []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadText(outside); err == nil {
		t.Fatal("outside read accepted")
	}
}

func TestProcessDisabled(t *testing.T) {
	root := t.TempDir()
	b, err := New(testConfig(root))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := b.RunProcess(context.Background(), "anything", nil, root, 1); err == nil ||
		!strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled process accepted: %v", err)
	}
}

func TestProcessAllowlist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable test uses /bin/echo")
	}
	root := t.TempDir()
	exe := "/bin/echo"
	if _, err := os.Stat(exe); err != nil {
		t.Skip(err)
	}
	cfg := testConfig(root)
	cfg.Process.Enabled = true
	cfg.Process.AllowedExecutables = []string{exe}
	cfg.Process.WorkingRoots = []string{root}
	b, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	out, err := b.RunProcess(context.Background(), exe, []string{"hello"}, root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out["stdout"].(string), "hello") {
		t.Fatalf("stdout=%q", out["stdout"])
	}
	if _, err := b.RunProcess(context.Background(), "/bin/sh", []string{"-c", "true"}, root, 1); err == nil {
		t.Fatal("unlisted executable accepted")
	}
}
