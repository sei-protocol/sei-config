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
func ApplyOverrides(cfg *SeiConfig, overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}

	// Encode current config to TOML, decode into generic map, apply overrides,
	// then re-decode into SeiConfig. This leverages TOML round-tripping to
	// handle type coercion for all field types.
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config for override application: %w", err)
	}

	var m map[string]any
	if _, err := toml.NewDecoder(&buf).Decode(&m); err != nil {
		return fmt.Errorf("decoding config map: %w", err)
	}

	for key, val := range overrides {
		if err := setNestedKey(m, key, val); err != nil {
			return fmt.Errorf("applying override %q=%q: %w", key, val, err)
		}
	}

	// Re-encode the modified map and decode back into SeiConfig
	var buf2 bytes.Buffer
	if err := toml.NewEncoder(&buf2).Encode(m); err != nil {
		return fmt.Errorf("re-encoding after overrides: %w", err)
	}
	if _, err := toml.NewDecoder(&buf2).Decode(cfg); err != nil {
		return fmt.Errorf("decoding overridden config: %w", err)
	}

	return nil
}

// setNestedKey sets a value in a nested map using a dotted key path.
// It attempts to coerce the string value to match the existing value's type.
func setNestedKey(m map[string]any, dottedKey string, value string) error {
	parts := splitDottedKey(dottedKey)
	if len(parts) == 0 {
		return fmt.Errorf("empty key")
	}

	current := m
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			child := make(map[string]any)
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("key %q is not a section", part)
		}
		current = child
	}

	finalKey := parts[len(parts)-1]
	existing := current[finalKey]
	coerced, err := coerceToType(value, existing)
	if err != nil {
		return fmt.Errorf("coercing value for %q: %w", dottedKey, err)
	}
	current[finalKey] = coerced
	return nil
}

// coerceToType attempts to convert a string value to match the type of an
// existing value. Falls back to string if no existing value or unknown type.
func coerceToType(value string, existing any) (any, error) {
	if existing == nil {
		return value, nil
	}
	switch existing.(type) {
	case int64:
		n, err := parseInt64(value)
		return n, err
	case float64:
		n, err := parseFloat64(value)
		return n, err
	case bool:
		switch value {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid bool: %q", value)
		}
	case string:
		return value, nil
	default:
		return value, nil
	}
}

func splitDottedKey(key string) []string {
	var parts []string
	current := ""
	for _, c := range key {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
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

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
