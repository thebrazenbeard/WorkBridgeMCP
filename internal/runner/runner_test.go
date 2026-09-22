package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func hashForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func platformEcho(t *testing.T) (string, []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := os.Getenv("COMSPEC")
		if path == "" {
			path = `C:\Windows\System32\cmd.exe`
		}
		return path, []string{"/C", "echo", "workbridge"}
	}
	return "/bin/echo", []string{"workbridge"}
}

func TestPinnedExecutableRunsInBoundedWorkingRoot(t *testing.T) {
	exe, args := platformEcho(t)
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("platform echo unavailable: %v", err)
	}
	root := t.TempDir()
	cfg := config.ProcessConfig{
		Enabled: true,
		AllowedExecutables: []config.ExecutableGrant{{
			Name: "echo", Path: exe, SHA256: hashForTest(t, exe),
		}},
		WorkingRoots: []string{root},
		MaxRuntimeSeconds: 5,
		MaxOutputBytes: 1024,
		MaxArgs: 8,
	}
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := r.Run(ctx, "echo", args, root)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.TimedOut {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestRejectsWrongHashAndOutsideWorkingRoot(t *testing.T) {
	exe, _ := platformEcho(t)
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("platform echo unavailable: %v", err)
	}
	root := t.TempDir()
	cfg := config.ProcessConfig{
		Enabled: true,
		AllowedExecutables: []config.ExecutableGrant{{
			Name: "echo", Path: exe, SHA256: string(make([]byte, 64)),
		}},
		WorkingRoots: []string{root},
		MaxRuntimeSeconds: 1,
		MaxOutputBytes: 1024,
		MaxArgs: 8,
	}
	if _, err := New(cfg); err == nil {
		t.Fatal("wrong executable hash accepted")
	}
	cfg.AllowedExecutables[0].SHA256 = hashForTest(t, exe)
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), "echo", nil, filepath.Clean(t.TempDir())); err == nil {
		t.Fatal("outside working root accepted")
	}
}
