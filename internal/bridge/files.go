package bridge

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode/utf8"
)

func (b *Bridge) ReadText(path string) (map[string]any, error) {
	resolved, err := b.readPolicy.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	data, err := readBounded(resolved, b.cfg.Limits.MaxReadBytes)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, errors.New("file is not valid UTF-8; use read_bytes")
	}
	sum := sha256.Sum256(data)
	return map[string]any{
		"path":    resolved,
		"content": string(data),
		"bytes":   len(data),
		"sha256":  fmt.Sprintf("%x", sum[:]),
	}, nil
}

func (b *Bridge) ReadBytes(path string, offset, maxBytes int64) (map[string]any, error) {
	resolved, err := b.readPolicy.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, errors.New("offset must be non-negative")
	}
	if maxBytes <= 0 {
		maxBytes = b.cfg.Limits.MaxReadBytes
	}
	if maxBytes > b.cfg.Limits.MaxReadBytes {
		return nil, errors.New("max_bytes exceeds configured read ceiling")
	}
	f, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("path is a directory")
	}
	if offset > info.Size() {
		return nil, errors.New("offset exceeds file size")
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"path":      resolved,
		"offset":    offset,
		"bytes":     len(data),
		"file_size": info.Size(),
		"eof":       offset+int64(len(data)) >= info.Size(),
		"base64":    base64.StdEncoding.EncodeToString(data),
	}, nil
}

func (b *Bridge) Stat(path string) (map[string]any, error) {
	resolved, err := b.readPolicy.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"path":          resolved,
		"name":          info.Name(),
		"size":          info.Size(),
		"is_directory":  info.IsDir(),
		"mode":          info.Mode().String(),
		"modified_utc":  info.ModTime().UTC().Format(time.RFC3339Nano),
	}, nil
}

func (b *Bridge) ListDirectory(path string, maxEntries int) (map[string]any, error) {
	resolved, err := b.readPolicy.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	if maxEntries <= 0 {
		maxEntries = b.cfg.Limits.MaxDirectoryEntries
	}
	if maxEntries > b.cfg.Limits.MaxDirectoryEntries {
		return nil, errors.New("max_entries exceeds configured directory ceiling")
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	truncated := len(entries) > maxEntries
	if truncated {
		entries = entries[:maxEntries]
	}
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		item := map[string]any{"name": entry.Name(), "is_directory": entry.IsDir()}
		if info, err := entry.Info(); err == nil {
			item["size"] = info.Size()
			item["modified_utc"] = info.ModTime().UTC().Format(time.RFC3339Nano)
		}
		out = append(out, item)
	}
	return map[string]any{"path": resolved, "entries": out, "truncated": truncated}, nil
}

func (b *Bridge) WriteText(path, content string, overwrite bool) (map[string]any, error) {
	data := []byte(content)
	if int64(len(data)) > b.cfg.Limits.MaxWriteBytes {
		return nil, errors.New("content exceeds configured write ceiling")
	}
	resolved, err := b.writePolicy.ResolveForCreate(path)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(resolved); err == nil {
		if info.IsDir() {
			return nil, errors.New("path is a directory")
		}
		if !overwrite {
			return nil, errors.New("destination exists and overwrite is false")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	parent, err := b.writePolicy.ResolveExisting(filepath.Dir(resolved))
	if err != nil {
		return nil, fmt.Errorf("parent directory: %w", err)
	}
	resolved = filepath.Join(parent, filepath.Base(resolved))
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(resolved, flags, 0o600)
	if err != nil {
		return nil, err
	}
	n, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if n != len(data) {
		return nil, io.ErrShortWrite
	}
	sum := sha256.Sum256(data)
	return map[string]any{"path": resolved, "bytes": n, "sha256": fmt.Sprintf("%x", sum[:])}, nil
}

func (b *Bridge) MakeDirectory(path string, parents bool) (map[string]any, error) {
	resolved, err := b.writePolicy.ResolveForCreate(path)
	if err != nil {
		return nil, err
	}
	if parents {
		err = os.MkdirAll(resolved, 0o700)
	} else {
		err = os.Mkdir(resolved, 0o700)
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": resolved, "created": true}, nil
}

func (b *Bridge) MovePath(source, destination string) (map[string]any, error) {
	src, err := b.writePolicy.ResolveExisting(source)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	dst, err := b.writePolicy.ResolveForCreate(destination)
	if err != nil {
		return nil, fmt.Errorf("destination: %w", err)
	}
	if _, err := os.Lstat(dst); err == nil {
		return nil, errors.New("destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	parent, err := b.writePolicy.ResolveExisting(filepath.Dir(dst))
	if err != nil {
		return nil, fmt.Errorf("destination parent: %w", err)
	}
	dst = filepath.Join(parent, filepath.Base(dst))
	if err := os.Rename(src, dst); err != nil {
		return nil, err
	}
	return map[string]any{"source": src, "destination": dst}, nil
}

func readBounded(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("path is a directory")
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("file exceeds configured read ceiling")
	}
	return data, nil
}
