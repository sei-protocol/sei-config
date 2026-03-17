package seiconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	configDir      = "config"
	configTomlFile = "config.toml"
	appTomlFile    = "app.toml"
)

// ReadConfigFromDir reads config.toml and app.toml from homeDir/config/ and
// merges them into a unified SeiConfig.
func ReadConfigFromDir(homeDir string) (*SeiConfig, error) {
	cfgDir := filepath.Join(homeDir, configDir)
	configPath := filepath.Join(cfgDir, configTomlFile)
	appPath := filepath.Join(cfgDir, appTomlFile)

	var tm legacyTendermintConfig
	if _, err := toml.DecodeFile(configPath, &tm); err != nil {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}

	var app legacyAppConfig
	if _, err := toml.DecodeFile(appPath, &app); err != nil {
		return nil, fmt.Errorf("reading %s: %w", appPath, err)
	}

	cfg := fromLegacy(tm, app)
	return cfg, nil
}

// WriteConfigToDir writes the SeiConfig as config.toml and app.toml into
// homeDir/config/. Writes are atomic (temp file + rename) to prevent
// corruption on crash.
func WriteConfigToDir(cfg *SeiConfig, homeDir string) error {
	cfgDir := filepath.Join(homeDir, configDir)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(cfgDir, configTomlFile)
	appPath := filepath.Join(cfgDir, appTomlFile)

	tm := cfg.toLegacyTendermint()
	if err := atomicWriteTOML(configPath, tm); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}

	app := cfg.toLegacyApp()
	if err := atomicWriteTOML(appPath, app); err != nil {
		return fmt.Errorf("writing %s: %w", appPath, err)
	}

	return nil
}

// ApplyOverrides applies a map of dotted-key overrides to a SeiConfig.
// Keys use the unified schema paths (e.g. "evm.http_port", "storage.pruning").
// This is the primary mechanism for the sidecar's ConfigApplyTask and the
// controller's spec.config.overrides.
//
// Each TOML key is resolved to its Go struct field path via the Registry, then
// set directly through reflection — the same path used by ResolveEnv.
func ApplyOverrides(cfg *SeiConfig, overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}

	reg := BuildRegistry()
	for key, val := range overrides {
		f := reg.Field(key)
		if f == nil {
			return fmt.Errorf("unknown override key %q", key)
		}
		if err := setFieldByPath(cfg, f.FieldPath, val); err != nil {
			return fmt.Errorf("applying override %q=%q: %w", key, val, err)
		}
	}
	return nil
}

// atomicWriteTOML encodes v as TOML and writes it atomically to path.
func atomicWriteTOML(path string, v any) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sei-config-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
