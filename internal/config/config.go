package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const Schema = "WORKBRIDGE_CONFIG_V1"

type Limits struct {
	MaxReadBytes        int64 `json:"max_read_bytes"`
	MaxWriteBytes       int64 `json:"max_write_bytes"`
	MaxDirectoryEntries int   `json:"max_directory_entries"`
}

type ProcessConfig struct {
	Enabled            bool     `json:"enabled"`
	AllowedExecutables []string `json:"allowed_executables"`
	WorkingRoots       []string `json:"working_roots"`
	MaxRuntimeSeconds  float64  `json:"max_runtime_seconds"`
	MaxOutputBytes     int64    `json:"max_output_bytes"`
}

type HTTPConfig struct {
	Listen         string `json:"listen"`
	Path           string `json:"path"`
	BearerTokenEnv string `json:"bearer_token_env"`
}

type Config struct {
	Schema     string        `json:"schema"`
	ReadRoots  []string      `json:"read_roots"`
	WriteRoots []string      `json:"write_roots"`
	Limits     Limits        `json:"limits"`
	Process    ProcessConfig `json:"process"`
	HTTP       HTTPConfig    `json:"http"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Config, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("config contains trailing JSON")
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Limits.MaxReadBytes == 0 {
		c.Limits.MaxReadBytes = 1024 * 1024
	}
	if c.Limits.MaxWriteBytes == 0 {
		c.Limits.MaxWriteBytes = 1024 * 1024
	}
	if c.Limits.MaxDirectoryEntries == 0 {
		c.Limits.MaxDirectoryEntries = 500
	}
	if c.Process.MaxRuntimeSeconds == 0 {
		c.Process.MaxRuntimeSeconds = 60
	}
	if c.Process.MaxOutputBytes == 0 {
		c.Process.MaxOutputBytes = 1024 * 1024
	}
	if strings.TrimSpace(c.HTTP.Listen) == "" {
		c.HTTP.Listen = "127.0.0.1:8765"
	}
	if strings.TrimSpace(c.HTTP.Path) == "" {
		c.HTTP.Path = "/mcp"
	}
	if strings.TrimSpace(c.HTTP.BearerTokenEnv) == "" {
		c.HTTP.BearerTokenEnv = "WORKBRIDGE_HTTP_TOKEN"
	}
}

func (c *Config) Validate() error {
	if c.Schema != Schema {
		return fmt.Errorf("schema must be %q", Schema)
	}
	if err := validateRoots("read_roots", c.ReadRoots); err != nil {
		return err
	}
	if err := validateRoots("write_roots", c.WriteRoots); err != nil {
		return err
	}
	if c.Limits.MaxReadBytes < 1 || c.Limits.MaxReadBytes > 64*1024*1024 {
		return errors.New("limits.max_read_bytes must be between 1 and 67108864")
	}
	if c.Limits.MaxWriteBytes < 1 || c.Limits.MaxWriteBytes > 64*1024*1024 {
		return errors.New("limits.max_write_bytes must be between 1 and 67108864")
	}
	if c.Limits.MaxDirectoryEntries < 1 || c.Limits.MaxDirectoryEntries > 10000 {
		return errors.New("limits.max_directory_entries must be between 1 and 10000")
	}
	if err := validateLoopbackListen(c.HTTP.Listen); err != nil {
		return err
	}
	if !strings.HasPrefix(c.HTTP.Path, "/") || c.HTTP.Path == "/" ||
		strings.ContainsAny(c.HTTP.Path, "?#") || strings.HasSuffix(c.HTTP.Path, "/") {
		return errors.New("http.path must be a dedicated absolute path without query, fragment, or trailing slash")
	}
	if !validEnvName(c.HTTP.BearerTokenEnv) {
		return errors.New("http.bearer_token_env must be a valid environment-variable name")
	}
	if c.Process.MaxRuntimeSeconds <= 0 || c.Process.MaxRuntimeSeconds > 900 {
		return errors.New("process.max_runtime_seconds must be > 0 and <= 900")
	}
	if c.Process.MaxOutputBytes < 1 || c.Process.MaxOutputBytes > 16*1024*1024 {
		return errors.New("process.max_output_bytes must be between 1 and 16777216")
	}
	if c.Process.Enabled {
		if len(c.Process.AllowedExecutables) == 0 {
			return errors.New("process.allowed_executables is required when process execution is enabled")
		}
		if len(c.Process.WorkingRoots) == 0 {
			return errors.New("process.working_roots is required when process execution is enabled")
		}
		if err := validateRoots("process.working_roots", c.Process.WorkingRoots); err != nil {
			return err
		}
		for _, executable := range c.Process.AllowedExecutables {
			if !filepath.IsAbs(executable) {
				return fmt.Errorf("process.allowed_executables entry must be absolute: %q", executable)
			}
		}
	}
	return nil
}

func validateRoots(name string, roots []string) error {
	for _, root := range roots {
		if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) {
			return fmt.Errorf("%s entries must be non-empty absolute paths: %q", name, root)
		}
	}
	return nil
}

func validateLoopbackListen(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("http.listen must be host:port: %w", err)
	}
	if port == "" {
		return errors.New("http.listen requires a port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("http.listen must use a literal loopback address")
	}
	return nil
}

func validEnvName(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
