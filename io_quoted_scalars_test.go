package seiconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadConfigFromDir_CoercesQuotedScalars reproduces the seid/tendermint
// config template's quoted primitives — e.g. `duplicate-txs-cache-size = "100000"`
// (a Go int) and `broadcast = "true"` (a Go bool) — and asserts ReadConfigFromDir
// coerces them instead of failing a strict decode.
//
// Regression for the v2 ConfigManager differential (PLT-775): a real seid
// config.toml is written with these primitives quoted, and the legacy reader
// (cosmos/Viper) tolerates it via weakly-typed coercion. ReadConfigFromDir must
// do the same, or v2 cannot read a real node's config and fails at boot.
func TestReadConfigFromDir_CoercesQuotedScalars(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, configDir)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Quoted int, quoted bool, and a string-encoded Duration — the three
	// coercion paths the lenient decoder must handle.
	configToml := `
[mempool]
duplicate-txs-cache-size = "100000"
broadcast = "true"
ttl-duration = "1s"
`
	writeFile(t, filepath.Join(cfgDir, configTomlFile), configToml)
	writeFile(t, filepath.Join(cfgDir, appTomlFile), "") // empty app.toml: fields default

	cfg, err := ReadConfigFromDir(home)
	if err != nil {
		t.Fatalf("ReadConfigFromDir failed on quoted scalars: %v", err)
	}
	if got := cfg.Mempool.DuplicateTxsCacheSize; got != 100000 {
		t.Errorf("Mempool.DuplicateTxsCacheSize = %d, want 100000 (quoted int not coerced)", got)
	}
	if !cfg.Mempool.Broadcast {
		t.Errorf("Mempool.Broadcast = false, want true (quoted bool not coerced)")
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
