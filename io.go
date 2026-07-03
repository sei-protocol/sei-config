package seiconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/go-viper/mapstructure/v2"
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
	if err := decodeTOMLFile(configPath, &tm); err != nil {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}

	var app legacyAppConfig
	if err := decodeTOMLFile(appPath, &app); err != nil {
		return nil, fmt.Errorf("reading %s: %w", appPath, err)
	}

	cfg := fromLegacy(tm, app)
	return cfg, nil
}

// decodeTOMLFile decodes a TOML file into out, coercing quoted scalars the way
// the legacy reader does. The seid/tendermint config templates emit some
// numeric and bool fields quoted (e.g. `duplicate-txs-cache-size = "100000"`,
// `gossip-tx-key-only = "true"`); BurntSushi/toml alone is strict and rejects a
// quoted string into an int/bool field, so we decode to a generic map and then
// weakly-typed-decode into the struct. The TextUnmarshaller hook keeps
// string-encoded types (Duration) parsing.
//
// This approximates how cosmos/Viper tolerates quoted scalars (not full parity
// — Viper's hooks and key handling differ). WeaklyTypedInput widens tolerance
// two ways worth knowing: (a) a non-string scalar bound to a string field is
// stringified (e.g. a stray `true` becomes "1") — accepted as benign since seid
// templates never emit that form; (b) an empty string bound to a numeric/bool
// field would coerce to the zero value — this we reject (see
// rejectEmptyScalarStringHook) so blanking a limit errors rather than silently
// pinning it to zero. Genuinely malformed values (non-numeric strings, bad
// durations, overflow) still error (locked by io_quoted_scalars_test.go).
func decodeTOMLFile(path string, out any) error {
	var raw map[string]any
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return err
	}
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			rejectEmptyScalarStringHook,
			mapstructure.TextUnmarshallerHookFunc(),
		),
		WeaklyTypedInput: true,
		TagName:          "toml",
		Result:           out,
	})
	if err != nil {
		return err
	}
	return dec.Decode(raw)
}

// rejectEmptyScalarStringHook fails an empty-string value bound to a numeric or
// bool field instead of letting WeaklyTypedInput silently coerce it to the zero
// value. Blanking a numeric (a connection limit, a cache size) should error,
// not silently pin it to 0/false. Non-empty strings pass through unchanged to
// the quoted-scalar coercion the template requires.
func rejectEmptyScalarStringHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}
	if s, _ := data.(string); strings.TrimSpace(s) != "" {
		return data, nil
	}
	t := to
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return nil, fmt.Errorf("empty string for %s field", to)
	default:
		return data, nil
	}
}

// WriteConfigToDir writes the SeiConfig as config.toml and app.toml into
// homeDir/config/. Both files are staged (temp + fsync) then committed together,
// so a staging failure leaves the existing files untouched.
func WriteConfigToDir(cfg *SeiConfig, homeDir string) error {
	cfgDir := filepath.Join(homeDir, configDir)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(cfgDir, configTomlFile)
	appPath := filepath.Join(cfgDir, appTomlFile)

	// Stage both files before committing either: encode + write + fsync each to a
	// temp file first, so if any of that fails the home is left untouched.
	stagedConfig, err := stageTOML(cfgDir, cfg.toLegacyTendermint())
	if err != nil {
		return fmt.Errorf("staging %s: %w", configPath, err)
	}
	stagedApp, err := stageTOML(cfgDir, cfg.toLegacyApp())
	if err != nil {
		_ = os.Remove(stagedConfig)
		return fmt.Errorf("staging %s: %w", appPath, err)
	}

	// Commit both renames back-to-back. Residual window: any interruption before
	// the dir fsync completes (a failed/crashed second rename, or a crash after
	// both renames but before the fsync) can leave config.toml new / app.toml old
	// — each file individually valid, cross-file inconsistent.
	if err := os.Rename(stagedConfig, configPath); err != nil {
		_ = os.Remove(stagedConfig)
		_ = os.Remove(stagedApp)
		return fmt.Errorf("committing %s: %w", configPath, err)
	}
	if err := os.Rename(stagedApp, appPath); err != nil {
		_ = os.Remove(stagedApp)
		return fmt.Errorf("committing %s: %w", appPath, err)
	}

	// fsync the config dir so both renames are durable together; a sync failure
	// means the renames may not be on disk, so surface it rather than report
	// success.
	dir, err := os.Open(cfgDir)
	if err != nil {
		return fmt.Errorf("opening config dir for fsync: %w", err)
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return fmt.Errorf("fsyncing config dir: %w", err)
	}
	if err := dir.Close(); err != nil {
		return fmt.Errorf("closing config dir: %w", err)
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

// stageTOML encodes v as TOML into a synced, chmod'd temp file in dir and
// returns its path without renaming it into place. Callers stage several files
// and then commit them together (rename each), so a partial write never lands.
func stageTOML(dir string, v any) (string, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp(dir, ".sei-config-*.tmp")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("setting permissions: %w", err)
	}

	return tmpPath, nil
}
