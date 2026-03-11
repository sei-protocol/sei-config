package seiconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultForMode_AllModesValid(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive, ModeRPC, ModeIndexer}
	for _, mode := range modes {
		cfg := DefaultForMode(mode)
		if cfg.Mode != mode {
			t.Errorf("DefaultForMode(%s): got mode %s", mode, cfg.Mode)
		}
		if cfg.Version != CurrentVersion {
			t.Errorf("DefaultForMode(%s): got version %d, want %d", mode, cfg.Version, CurrentVersion)
		}

		result := Validate(cfg)
		if result.HasErrors() {
			t.Errorf("DefaultForMode(%s) produced validation errors: %v", mode, result.Errors())
		}
	}
}

func TestDefaultForMode_ValidatorDisablesServices(t *testing.T) {
	cfg := DefaultForMode(ModeValidator)

	if cfg.API.REST.Enable {
		t.Error("validator should have REST API disabled")
	}
	if cfg.API.GRPC.Enable {
		t.Error("validator should have gRPC disabled")
	}
	if cfg.EVM.HTTPEnabled {
		t.Error("validator should have EVM HTTP disabled")
	}
	if cfg.EVM.WSEnabled {
		t.Error("validator should have EVM WS disabled")
	}
	if cfg.Storage.StateStore.Enable {
		t.Error("validator should have state store disabled")
	}
}

func TestDefaultForMode_SeedHighConnections(t *testing.T) {
	cfg := DefaultForMode(ModeSeed)

	if cfg.Network.P2P.MaxConnections != 1000 {
		t.Errorf("seed max_connections: got %d, want 1000", cfg.Network.P2P.MaxConnections)
	}
	if !cfg.Network.P2P.AllowDuplicateIP {
		t.Error("seed should allow duplicate IPs")
	}
	if cfg.Storage.PruningStrategy != "everything" {
		t.Errorf("seed pruning: got %s, want everything", cfg.Storage.PruningStrategy)
	}
}

func TestDefaultForMode_ArchiveKeepsAll(t *testing.T) {
	cfg := DefaultForMode(ModeArchive)

	if cfg.Storage.PruningStrategy != "nothing" {
		t.Errorf("archive pruning: got %s, want nothing", cfg.Storage.PruningStrategy)
	}
	if cfg.Storage.StateStore.KeepRecent != 0 {
		t.Errorf("archive state_store.keep_recent: got %d, want 0", cfg.Storage.StateStore.KeepRecent)
	}
	if cfg.Chain.MinRetainBlocks != 0 {
		t.Errorf("archive min_retain_blocks: got %d, want 0", cfg.Chain.MinRetainBlocks)
	}
	if cfg.EVM.MaxTraceLookbackBlocks != -1 {
		t.Errorf("archive max_trace_lookback_blocks: got %d, want -1", cfg.EVM.MaxTraceLookbackBlocks)
	}
}

func TestDefaultForMode_FullEnablesServices(t *testing.T) {
	cfg := DefaultForMode(ModeFull)

	if !cfg.API.REST.Enable {
		t.Error("full should have REST API enabled")
	}
	if !cfg.API.GRPC.Enable {
		t.Error("full should have gRPC enabled")
	}
	if !cfg.EVM.HTTPEnabled {
		t.Error("full should have EVM HTTP enabled")
	}
	if cfg.Network.RPC.ListenAddress != "tcp://0.0.0.0:26657" {
		t.Errorf("full RPC listen: got %s, want tcp://0.0.0.0:26657", cfg.Network.RPC.ListenAddress)
	}
}

func TestValidate_InvalidMode(t *testing.T) {
	cfg := Default()
	cfg.Mode = "bogus"
	result := Validate(cfg)
	if !result.HasErrors() {
		t.Error("expected error for invalid mode")
	}
}

func TestValidate_EmptyMinGasPrices(t *testing.T) {
	cfg := Default()
	cfg.Chain.MinGasPrices = ""
	result := Validate(cfg)
	if !result.HasErrors() {
		t.Error("expected error for empty min_gas_prices")
	}
}

func TestValidate_InvalidPruningStrategy(t *testing.T) {
	cfg := Default()
	cfg.Storage.PruningStrategy = "aggressive"
	result := Validate(cfg)
	if !result.HasErrors() {
		t.Error("expected error for invalid pruning strategy")
	}
}

func TestValidate_PruningEverythingWithSnapshots(t *testing.T) {
	cfg := Default()
	cfg.Storage.PruningStrategy = "everything"
	cfg.Storage.SnapshotInterval = 1000
	result := Validate(cfg)
	if !result.HasErrors() {
		t.Error("expected error for snapshots with everything pruning")
	}
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	cfg := Default()
	cfg.Logging.Format = "xml"
	result := Validate(cfg)
	if !result.HasErrors() {
		t.Error("expected error for invalid log format")
	}
}

func TestValidate_EVMOnValidator(t *testing.T) {
	cfg := DefaultForMode(ModeValidator)
	cfg.EVM.HTTPEnabled = true
	result := Validate(cfg)
	hasWarning := false
	for _, d := range result.Diagnostics {
		if d.Severity == SeverityWarning && d.Field == "evm" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning for EVM on validator")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := DefaultForMode(ModeFull)
	// Note: ChainID is stored in genesis.json, not config.toml/app.toml,
	// so it does not round-trip through the legacy two-file format.
	original.Chain.Moniker = "test-node"
	original.EVM.HTTPPort = 9545
	original.Storage.StateStore.KeepRecent = 50000

	if err := WriteConfigToDir(original, dir); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}

	configPath := filepath.Join(dir, "config", "config.toml")
	appPath := filepath.Join(dir, "config", "app.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	if _, err := os.Stat(appPath); err != nil {
		t.Fatalf("app.toml not created: %v", err)
	}

	loaded, err := ReadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}

	if loaded.Chain.Moniker != "test-node" {
		t.Errorf("moniker: got %q, want %q", loaded.Chain.Moniker, "test-node")
	}
	if loaded.EVM.HTTPPort != 9545 {
		t.Errorf("evm.http_port: got %d, want 9545", loaded.EVM.HTTPPort)
	}
	if loaded.Storage.StateStore.KeepRecent != 50000 {
		t.Errorf("state_store.keep_recent: got %d, want 50000", loaded.Storage.StateStore.KeepRecent)
	}
	if loaded.Network.RPC.ListenAddress != "tcp://0.0.0.0:26657" {
		t.Errorf("rpc.listen_address: got %q", loaded.Network.RPC.ListenAddress)
	}
}

func TestWriteReadRoundTrip_AllModes(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive, ModeRPC, ModeIndexer}
	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			dir := t.TempDir()
			original := DefaultForMode(mode)

			if err := WriteConfigToDir(original, dir); err != nil {
				t.Fatalf("WriteConfigToDir: %v", err)
			}

			loaded, err := ReadConfigFromDir(dir)
			if err != nil {
				t.Fatalf("ReadConfigFromDir: %v", err)
			}

			if loaded.Chain.MinGasPrices != original.Chain.MinGasPrices {
				t.Errorf("min_gas_prices: got %q, want %q",
					loaded.Chain.MinGasPrices, original.Chain.MinGasPrices)
			}
			if loaded.Storage.PruningStrategy != original.Storage.PruningStrategy {
				t.Errorf("pruning: got %q, want %q",
					loaded.Storage.PruningStrategy, original.Storage.PruningStrategy)
			}
		})
	}
}

func TestApplyOverrides(t *testing.T) {
	cfg := Default()
	overrides := map[string]string{
		"evm.http_port":             "9545",
		"chain.min_gas_prices":      "0.1usei",
		"storage.pruning":           "custom",
	}

	if err := ApplyOverrides(cfg, overrides); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}

	if cfg.EVM.HTTPPort != 9545 {
		t.Errorf("evm.http_port: got %d, want 9545", cfg.EVM.HTTPPort)
	}
	if cfg.Chain.MinGasPrices != "0.1usei" {
		t.Errorf("chain.min_gas_prices: got %q, want %q", cfg.Chain.MinGasPrices, "0.1usei")
	}
	if cfg.Storage.PruningStrategy != "custom" {
		t.Errorf("storage.pruning: got %q, want %q", cfg.Storage.PruningStrategy, "custom")
	}
}

func TestApplyOverrides_Empty(t *testing.T) {
	cfg := Default()
	original := cfg.EVM.HTTPPort
	if err := ApplyOverrides(cfg, nil); err != nil {
		t.Fatalf("ApplyOverrides(nil): %v", err)
	}
	if cfg.EVM.HTTPPort != original {
		t.Error("nil overrides should not change config")
	}
}

func TestResolveEnv(t *testing.T) {
	cfg := Default()
	t.Setenv("SEI_CHAIN_MIN_GAS_PRICES", "0.5usei")

	warnings := ResolveEnv(cfg)
	if cfg.Chain.MinGasPrices != "0.5usei" {
		t.Errorf("after ResolveEnv: got %q, want %q", cfg.Chain.MinGasPrices, "0.5usei")
	}
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}
}

func TestResolveEnv_LegacyPrefix(t *testing.T) {
	cfg := Default()
	t.Setenv("SEID_CHAIN_MIN_GAS_PRICES", "0.3usei")

	warnings := ResolveEnv(cfg)
	if cfg.Chain.MinGasPrices != "0.3usei" {
		t.Errorf("after ResolveEnv with SEID_: got %q, want %q", cfg.Chain.MinGasPrices, "0.3usei")
	}
	hasDeprecation := false
	for _, w := range warnings {
		if w != "" {
			hasDeprecation = true
		}
	}
	if !hasDeprecation {
		t.Error("expected deprecation warning for SEID_ prefix")
	}
}

func TestResolveEnv_SEIPrecedence(t *testing.T) {
	cfg := Default()
	t.Setenv("SEI_CHAIN_MIN_GAS_PRICES", "0.5usei")
	t.Setenv("SEID_CHAIN_MIN_GAS_PRICES", "0.3usei")

	ResolveEnv(cfg)
	if cfg.Chain.MinGasPrices != "0.5usei" {
		t.Errorf("SEI_ should take precedence: got %q, want %q", cfg.Chain.MinGasPrices, "0.5usei")
	}
}

func TestDuration_MarshalUnmarshal(t *testing.T) {
	d := Dur(10 * time.Second)
	text, err := d.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(text) != "10s" {
		t.Errorf("MarshalText: got %q, want %q", string(text), "10s")
	}

	var d2 Duration
	if err := d2.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if d2.Duration != d.Duration {
		t.Errorf("round-trip: got %v, want %v", d2.Duration, d.Duration)
	}
}

func TestDuration_InvalidParse(t *testing.T) {
	var d Duration
	if err := d.UnmarshalText([]byte("not-a-duration")); err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestNodeMode_Validity(t *testing.T) {
	tests := []struct {
		mode  NodeMode
		valid bool
	}{
		{ModeValidator, true},
		{ModeFull, true},
		{ModeSeed, true},
		{ModeArchive, true},
		{ModeRPC, true},
		{ModeIndexer, true},
		{"bogus", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.mode.IsValid(); got != tt.valid {
			t.Errorf("NodeMode(%q).IsValid() = %v, want %v", tt.mode, got, tt.valid)
		}
	}
}

func TestNodeMode_IsFullnodeType(t *testing.T) {
	fullnodeTypes := []NodeMode{ModeFull, ModeArchive, ModeRPC, ModeIndexer}
	for _, m := range fullnodeTypes {
		if !m.IsFullnodeType() {
			t.Errorf("%s should be fullnode type", m)
		}
	}
	nonFullnodeTypes := []NodeMode{ModeValidator, ModeSeed}
	for _, m := range nonFullnodeTypes {
		if m.IsFullnodeType() {
			t.Errorf("%s should not be fullnode type", m)
		}
	}
}

func TestWriteMode_Validity(t *testing.T) {
	if !WriteModeCosmosOnly.IsValid() {
		t.Error("cosmos_only should be valid")
	}
	if WriteMode("invalid").IsValid() {
		t.Error("'invalid' should not be valid")
	}
}

func TestLegacyTendermintMode_ArchiveMapped(t *testing.T) {
	cfg := DefaultForMode(ModeArchive)
	tm := cfg.toLegacyTendermint()
	if tm.Mode != "full" {
		t.Errorf("archive should map to tendermint mode 'full', got %q", tm.Mode)
	}
}
