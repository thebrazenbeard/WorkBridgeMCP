package bridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	written   int64
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - b.written
	if remaining > 0 {
		toWrite := p
		if int64(len(toWrite)) > remaining {
			toWrite = toWrite[:remaining]
			b.truncated = true
		}
		n, err := b.buf.Write(toWrite)
		b.written += int64(n)
		if err != nil {
			return original, err
		}
	}
	if int64(original) > remaining {
		b.truncated = true
	}
	return original, nil
}

func (b *Bridge) RunProcess(ctx context.Context, executable string, args []string, cwd string, timeoutSeconds float64) (map[string]any, error) {
	if !b.cfg.Process.Enabled {
		return nil, errors.New("process execution is disabled")
	}
	if len(args) > 128 {
		return nil, errors.New("too many process arguments")
	}
	totalArgBytes := 0
	for _, arg := range args {
		totalArgBytes += len(arg)
	}
	if totalArgBytes > 64*1024 {
		return nil, errors.New("process arguments exceed 64 KiB")
	}
	canonical, err := canonicalExecutable(executable)
	if err != nil {
		return nil, err
	}
	allowed, ok := b.allowedExec[pathKey(canonical)]
	if !ok {
		return nil, errors.New("executable is not allowed by policy")
	}
	workdir, err := b.workingPolicy.ResolveExisting(cwd)
	if err != nil {
		return nil, fmt.Errorf("working directory: %w", err)
	}
	info, err := os.Stat(workdir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("working directory is not a directory")
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = b.cfg.Process.MaxRuntimeSeconds
	}
	if timeoutSeconds > b.cfg.Process.MaxRuntimeSeconds {
		return nil, errors.New("timeout_seconds exceeds configured runtime ceiling")
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds*float64(time.Second)))
	defer cancel()
	cmd := exec.CommandContext(runCtx, allowed, args...)
	cmd.Dir = workdir
	cmd.Env = safeEnvironment()
	stdout := &cappedBuffer{limit: b.cfg.Process.MaxOutputBytes}
	stderr := &cappedBuffer{limit: b.cfg.Process.MaxOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else if timedOut {
			exitCode = -1
		} else {
			return nil, runErr
		}
	}
	return map[string]any{
		"executable":       allowed,
		"args":             args,
		"cwd":              workdir,
		"exit_code":        exitCode,
		"timed_out":        timedOut,
		"runtime_ms":       elapsed.Milliseconds(),
		"stdout":           stdout.buf.String(),
		"stderr":           stderr.buf.String(),
		"stdout_truncated": stdout.truncated,
		"stderr_truncated": stderr.truncated,
	}, nil
}

func safeEnvironment() []string {
	allowed := map[string]struct{}{
		"SYSTEMROOT": {}, "WINDIR": {}, "COMSPEC": {}, "TEMP": {}, "TMP": {},
		"USERPROFILE": {}, "PATH": {}, "PATHEXT": {}, "HOME": {}, "TMPDIR": {},
		"LANG": {}, "LC_ALL": {}, "TERM": {},
	}
	var out []string
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, keep := allowed[strings.ToUpper(key)]; keep {
			out = append(out, entry)
		}
	}
	return out
}
