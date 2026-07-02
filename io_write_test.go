package seiconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteConfigToDir_RoundTrip renders a default config, confirms both legacy
// files land, and reads them back. The default leaves write/read-mode unset, so
// render must omit those keys (letting each seid binary apply its own native
// default — their valid values differ across versions).
func TestWriteConfigToDir_RoundTrip(t *testing.T) {
	home := t.TempDir()
	if err := WriteConfigToDir(DefaultForMode(ModeValidator), home); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}

	for _, name := range []string{configTomlFile, appTomlFile} {
		p := filepath.Join(home, configDir, name)
		if fi, err := os.Stat(p); err != nil || fi.Size() == 0 {
			t.Fatalf("expected non-empty %s: stat err=%v", name, err)
		}
	}

	app, err := os.ReadFile(filepath.Join(home, configDir, appTomlFile))
	if err != nil {
		t.Fatalf("ReadFile app.toml: %v", err)
	}
	for _, key := range []string{"sc-write-mode", "sc-read-mode", "ss-write-mode", "ss-read-mode"} {
		if strings.Contains(string(app), key) {
			t.Errorf("default render must omit %q so the binary applies its own default, but it is present", key)
		}
	}

	got, err := ReadConfigFromDir(home)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if wm := got.Storage.StateCommit.WriteMode; wm != "" {
		t.Errorf("state_commit.write_mode: got %q, want unset", wm)
	}
	if wm := got.Storage.StateStore.WriteMode; wm != "" {
		t.Errorf("state_store.write_mode: got %q, want unset", wm)
	}
}

// TestWriteConfigToDir_ExplicitWriteModeRendered is the escape hatch: a mode set
// explicitly on the model IS rendered (only the unset default is omitted).
func TestWriteConfigToDir_ExplicitWriteModeRendered(t *testing.T) {
	home := t.TempDir()
	cfg := DefaultForMode(ModeValidator)
	cfg.Storage.StateCommit.WriteMode = WriteModeMemiavlOnly
	if err := WriteConfigToDir(cfg, home); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	got, err := ReadConfigFromDir(home)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if wm := got.Storage.StateCommit.WriteMode; wm != WriteModeMemiavlOnly {
		t.Errorf("explicit state_commit.write_mode: got %q, want %q", wm, WriteModeMemiavlOnly)
	}
}

// TestWriteConfigToDir_NoTempResidue asserts a successful write leaves no
// .sei-config-*.tmp staging files behind (they are renamed into place, not
// orphaned). Failure-path cleanup is not exercised here.
func TestWriteConfigToDir_NoTempResidue(t *testing.T) {
	home := t.TempDir()
	if err := WriteConfigToDir(DefaultForMode(ModeFull), home); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(home, configDir))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".sei-config-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("orphaned staging temp file left behind: %s", e.Name())
		}
	}
}

// TestWriteConfigToDir_ReceiptStoreUsesRSBackend guards the receipt-store key
// asymmetry: the backend key is rs-backend (prefixed); the binary rejects an
// unprefixed `backend` under [receipt-store] at startup, so render must never
// emit one.
func TestWriteConfigToDir_ReceiptStoreUsesRSBackend(t *testing.T) {
	home := t.TempDir()
	if err := WriteConfigToDir(DefaultForMode(ModeFull), home); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(home, configDir, appTomlFile))
	if err != nil {
		t.Fatalf("ReadFile app.toml: %v", err)
	}
	app := string(b)
	if !strings.Contains(app, "rs-backend") {
		t.Errorf("app.toml missing rs-backend key")
	}
	for line := range strings.SplitSeq(app, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "backend =") || strings.HasPrefix(s, "backend=") {
			t.Errorf("app.toml emits an unprefixed backend key the binary rejects: %q", s)
		}
	}
}
