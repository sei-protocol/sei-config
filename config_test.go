package seiconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testRPCAddr = "tcp://0.0.0.0:26657"

func TestDefaultForMode_AllModesValid(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive}
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
	if cfg.Storage.PruningStrategy != PruningEverything {
		t.Errorf("seed pruning: got %s, want everything", cfg.Storage.PruningStrategy)
	}
}

func TestDefaultForMode_ArchiveKeepsAll(t *testing.T) {
	cfg := DefaultForMode(ModeArchive)

	if cfg.Storage.PruningStrategy != PruningNothing {
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
	if got := cfg.Storage.ReceiptStore.KeepRecent; got != 0 {
		t.Errorf("archive receipt_store.keep_recent: got %d, want 0", got)
	}
	if got := cfg.Storage.ReceiptStore.PruneIntervalSeconds; got != 0 {
		t.Errorf("archive receipt_store.prune_interval_seconds: got %d, want 0", got)
	}
}

func TestWriteArchive_ReceiptStoreTOMLKeys(t *testing.T) {
	// Symmetric tag typos round-trip cleanly but produce a TOML seid rejects;
	// pin the literal upstream key tokens.
	dir := t.TempDir()
	if err := WriteConfigToDir(DefaultForMode(ModeArchive), dir); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "config", "app.toml"))
	if err != nil {
		t.Fatalf("read app.toml: %v", err)
	}
	out := string(raw)

	if !strings.Contains(out, "[receipt-store]") {
		t.Fatalf("app.toml missing [receipt-store] section:\n%s", out)
	}
	requiredKeys := []string{
		"rs-backend = ",
		"db-directory = ",
		"async-write-buffer = ",
		"keep-recent = ",
		"prune-interval-seconds = ",
		"tx-index-backend = ",
	}
	for _, k := range requiredKeys {
		if !strings.Contains(out, k) {
			t.Errorf("app.toml missing receipt-store key %q", k)
		}
	}
	// flagRSMisnamedBackend hard-errors on the unprefixed key at startup.
	if strings.Contains(out, "\nbackend = ") {
		t.Errorf("app.toml emits unprefixed `backend = ` which sei-chain rejects")
	}
}

func TestDefaultForMode_ReceiptStoreDefaults(t *testing.T) {
	rs := DefaultForMode(ModeFull).Storage.ReceiptStore

	if rs.Backend != BackendPebbleDB {
		t.Errorf("receipt_store.backend: got %q, want %q", rs.Backend, BackendPebbleDB)
	}
	if rs.AsyncWriteBuffer != 100 {
		t.Errorf("receipt_store.async_write_buffer: got %d, want 100", rs.AsyncWriteBuffer)
	}
	if rs.KeepRecent != 100_000 {
		t.Errorf("receipt_store.keep_recent: got %d, want 100000", rs.KeepRecent)
	}
	if rs.PruneIntervalSeconds != 600 {
		t.Errorf("receipt_store.prune_interval_seconds: got %d, want 600", rs.PruneIntervalSeconds)
	}
	if rs.TxIndexBackend != BackendPebbleDB {
		t.Errorf("receipt_store.tx_index_backend: got %q, want %q", rs.TxIndexBackend, BackendPebbleDB)
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
	if cfg.Network.RPC.ListenAddress != testRPCAddr {
		t.Errorf("full RPC listen: got %s, want %s", cfg.Network.RPC.ListenAddress, testRPCAddr)
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
	cfg.Storage.PruningStrategy = PruningEverything
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
	original.EVM.EnabledLegacySeiApis = []string{"sei_getLogs", "sei_getBlockByNumber"}
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
	if got := loaded.EVM.EnabledLegacySeiApis; len(got) != 2 ||
		got[0] != "sei_getLogs" || got[1] != "sei_getBlockByNumber" {
		t.Errorf("evm.enabled_legacy_sei_apis: got %v, want [sei_getLogs sei_getBlockByNumber]", got)
	}
	if loaded.Storage.StateStore.KeepRecent != 50000 {
		t.Errorf("state_store.keep_recent: got %d, want 50000", loaded.Storage.StateStore.KeepRecent)
	}
	if loaded.Network.RPC.ListenAddress != testRPCAddr {
		t.Errorf("rpc.listen_address: got %q", loaded.Network.RPC.ListenAddress)
	}
}

func TestWriteReadRoundTrip_AllModes(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive}
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
			if loaded.Storage.ReceiptStore != original.Storage.ReceiptStore {
				t.Errorf("receipt_store: got %+v, want %+v",
					loaded.Storage.ReceiptStore, original.Storage.ReceiptStore)
			}
		})
	}
}

func TestApplyOverrides(t *testing.T) {
	cfg := Default()
	overrides := map[string]string{
		"evm.http_port":        "9545",
		"chain.min_gas_prices": "0.1usei",
		"storage.pruning":      "custom",
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

func TestApplyOverrides_Bool(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"network.p2p.allow_duplicate_ip": "true",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if !cfg.Network.P2P.AllowDuplicateIP {
		t.Error("expected AllowDuplicateIP to be true")
	}

	if err := ApplyOverrides(cfg, map[string]string{
		"network.p2p.allow_duplicate_ip": "false",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.Network.P2P.AllowDuplicateIP {
		t.Error("expected AllowDuplicateIP to be false")
	}
}

func TestApplyOverrides_Uint(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"chain.halt_height": "999999",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.Chain.HaltHeight != 999999 {
		t.Errorf("halt_height: got %d, want 999999", cfg.Chain.HaltHeight)
	}
}

func TestApplyOverrides_Float(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"mempool.drop_priority_threshold": "0.75",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.Mempool.DropPriorityThreshold != 0.75 {
		t.Errorf("drop_priority_threshold: got %f, want 0.75", cfg.Mempool.DropPriorityThreshold)
	}
}

func TestApplyOverrides_Duration(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"network.rpc.timeout_broadcast_tx_commit": "30s",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.Network.RPC.TimeoutBroadcastTxCommit.Duration != 30*time.Second {
		t.Errorf("timeout_broadcast_tx_commit: got %v, want 30s",
			cfg.Network.RPC.TimeoutBroadcastTxCommit.Duration)
	}
}

func TestApplyOverrides_Int64(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"state_sync.backfill_blocks": "500",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.StateSync.BackfillBlocks != 500 {
		t.Errorf("backfill_blocks: got %d, want 500", cfg.StateSync.BackfillBlocks)
	}
}

func TestApplyOverrides_UnknownKey(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"totally.fake.key": "value",
	})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestApplyOverrides_InvalidBool(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"network.p2p.allow_duplicate_ip": "maybe",
	})
	if err == nil {
		t.Fatal("expected error for invalid bool value")
	}
}

func TestApplyOverrides_InvalidInt(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"evm.http_port": "not_a_number",
	})
	if err == nil {
		t.Fatal("expected error for non-numeric int value")
	}
}

func TestApplyOverrides_InvalidDuration(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"network.rpc.timeout_broadcast_tx_commit": "not_a_duration",
	})
	if err == nil {
		t.Fatal("expected error for invalid duration value")
	}
}

func TestApplyOverrides_Uint16Overflow(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"network.p2p.max_connections": "70000",
	})
	if err == nil {
		t.Fatal("expected error for uint16 overflow (65535 max)")
	}
}

func TestApplyOverrides_Int32Overflow(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"state_sync.fetchers": "3000000000",
	})
	if err == nil {
		t.Fatal("expected error for int32 overflow")
	}
}

func TestApplyOverrides_NegativeUint(t *testing.T) {
	cfg := Default()
	err := ApplyOverrides(cfg, map[string]string{
		"chain.halt_height": "-1",
	})
	if err == nil {
		t.Fatal("expected error for negative uint value")
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

func TestApplyOverrides_StringSlice(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"single value", "kv", []string{"kv"}},
		{"multi value", "kv,psql", []string{"kv", "psql"}},
		{"trims whitespace", " kv , psql ", []string{"kv", "psql"}},
		{"empty string yields empty slice", "", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			if err := ApplyOverrides(cfg, map[string]string{
				"tx_index.indexer": tc.in,
			}); err != nil {
				t.Fatalf("ApplyOverrides: %v", err)
			}
			got := cfg.TxIndex.Indexer
			if len(got) != len(tc.want) {
				t.Fatalf("indexer: got %v (len %d), want %v (len %d)",
					got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("indexer[%d]: got %q, want %q", i, got[i], tc.want[i])
				}
			}
			if got == nil {
				t.Error("indexer slice must be non-nil to render into TOML")
			}
		})
	}
}

func TestApplyOverrides_StringSliceRejectsEmptyEntries(t *testing.T) {
	cases := []string{"kv,,psql", ",kv", "kv,", ",,,", "kv, ,psql"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			cfg := Default()
			err := ApplyOverrides(cfg, map[string]string{
				"tx_index.indexer": in,
			})
			if err == nil {
				t.Fatalf("expected error for input %q, got nil", in)
			}
		})
	}
}

func TestApplyOverrides_StringSliceOverwritesDefault(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"tx_index.indexer": "kv",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if len(cfg.TxIndex.Indexer) != 1 || cfg.TxIndex.Indexer[0] != "kv" {
		t.Errorf("indexer: got %v, want [kv]", cfg.TxIndex.Indexer)
	}
}

func TestApplyOverrides_StringSliceRoundTripTOML(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"non-empty list survives round-trip", "kv,psql", []string{"kv", "psql"}},
		{"empty list survives round-trip as []", "", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultForMode(ModeFull)
			if err := ApplyOverrides(cfg, map[string]string{
				"tx_index.indexer": tc.in,
			}); err != nil {
				t.Fatalf("ApplyOverrides: %v", err)
			}
			subdir := t.TempDir()
			if err := WriteConfigToDir(cfg, subdir); err != nil {
				t.Fatalf("WriteConfigToDir: %v", err)
			}
			loaded, err := ReadConfigFromDir(subdir)
			if err != nil {
				t.Fatalf("ReadConfigFromDir: %v", err)
			}
			got := loaded.TxIndex.Indexer
			if len(got) != len(tc.want) {
				t.Fatalf("after round-trip: got %v (len %d), want %v (len %d)",
					got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("indexer[%d]: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
	_ = dir
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

func TestResolveEnv_StringSlice(t *testing.T) {
	cfg := Default()
	t.Setenv("SEI_EVM_ENABLED_LEGACY_SEI_APIS", "sei_getLogs,sei_getBlockByNumber")

	ResolveEnv(cfg)
	got := cfg.EVM.EnabledLegacySeiApis
	if len(got) != 2 || got[0] != "sei_getLogs" || got[1] != "sei_getBlockByNumber" {
		t.Errorf("after ResolveEnv: got %v, want [sei_getLogs sei_getBlockByNumber]", got)
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
		{"rpc", false},
		{"indexer", false},
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
	fullnodeTypes := []NodeMode{ModeFull, ModeArchive}
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
