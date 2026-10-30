package seiconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// newEVMKeys are the [evm] keys added after nodes already had app.toml files
// rendered without them.
var newEVMKeys = []string{
	"max_filters", "max_block_filter_hashes", "max_subscriptions_logs", "max_log_bytes",
	"enable_simulation", "max_estimate_gas_calls", "max_state_override_accounts", "max_state_override_slots",
	"max_trace_struct_log_bytes", "trace_allowed_tracers", "trace_allow_js_tracers", "enable_parallelized_block_trace",
	"trace_bake_enabled", "trace_bake_workers", "trace_bake_queue_size", "trace_bake_tracers",
	"trace_bake_window_blocks", "trace_bake_use_snapshot", "trace_bake_snapshot_window",
	"rate_limiting_enabled", "ip_rate_limit_rps", "ip_rate_limit_burst", "trusted_proxy_cidrs",
	"batch_request_limit", "batch_response_max_size", "max_request_body_bytes", "max_concurrent_request_bytes",
	"body_read_idle_timeout", "ws_admission_timeout", "max_open_connections",
	"rpc_default_timeout", "rpc_method_timeouts",
}

func TestEVMNewKeys_RoundTrip(t *testing.T) {
	cfg := DefaultForMode(ModeFull)
	e := &cfg.EVM
	e.MaxFilters = 50
	e.MaxBlockFilterHashes = 60
	e.MaxSubscriptionsLogs = 70
	e.MaxLogBytes = 1 << 20
	e.EnableSimulation = false
	e.MaxEstimateGasCalls = 7
	e.MaxStateOverrideAccounts = 8
	e.MaxStateOverrideSlots = 9
	e.MaxTraceStructLogBytes = 0
	e.TraceAllowedTracers = []string{"callTracer"}
	e.TraceAllowJSTracers = true
	e.EnableParallelizedBlockTrace = true
	e.TraceBakeEnabled = true
	e.TraceBakeWorkers = 4
	e.TraceBakeQueueSize = 128
	e.TraceBakeTracers = []string{"callTracer", "prestateTracer"}
	e.TraceBakeWindowBlocks = 5000
	e.TraceBakeUseSnapshot = true
	e.TraceBakeSnapshotWindow = 32
	e.RateLimitingEnabled = false
	e.IPRateLimitRPS = 12.5
	e.IPRateLimitBurst = 500
	e.TrustedProxyCIDRs = []string{"10.0.0.0/8", "fd00::/8"}
	e.BatchRequestLimit = 50
	e.BatchResponseMaxSize = 1000
	e.MaxRequestBodyBytes = 2048
	e.MaxConcurrentRequestBytes = 4096
	e.BodyReadIdleTimeout = Dur(3 * time.Second)
	e.WSAdmissionTimeout = Dur(4 * time.Second)
	e.MaxOpenConnections = 0
	e.RPCDefaultTimeout = Dur(0)
	e.RPCMethodTimeouts = []string{"eth_call=60s", "debug_trace*=0"}

	dir := t.TempDir()
	if err := WriteConfigToDir(cfg, dir); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	loaded, err := ReadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if !reflect.DeepEqual(loaded.EVM, cfg.EVM) {
		t.Errorf("evm round-trip mismatch:\n got  %+v\n want %+v", loaded.EVM, cfg.EVM)
	}
}

// TestEVMNewKeys_AbsentReadAsDefaults covers a node whose app.toml was rendered
// before these keys existed. They must read back at seid's defaults, not the
// zero value: re-rendering zeros would turn off rate limiting and write
// max_filters = 0, which seid refuses to start with.
func TestEVMNewKeys_AbsentReadAsDefaults(t *testing.T) {
	dir := t.TempDir()
	want := DefaultForMode(ModeFull)
	if err := WriteConfigToDir(want, dir); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	stripAppTomlKeys(t, dir, newEVMKeys)

	loaded, err := ReadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if !reflect.DeepEqual(loaded.EVM, want.EVM) {
		t.Errorf("absent keys did not read as defaults:\n got  %+v\n want %+v", loaded.EVM, want.EVM)
	}
	if vr := Validate(loaded); vr.HasErrors() {
		t.Errorf("config read from an older app.toml fails validation: %v", vr.Errors())
	}
}

// TestEVMNewKeys_ExplicitEmptyListNotSeeded guards the seeding: an explicit
// empty list in the file must not be replaced by (or merged with) the default.
func TestEVMNewKeys_ExplicitEmptyListNotSeeded(t *testing.T) {
	cfg := DefaultForMode(ModeFull)
	cfg.EVM.TraceAllowedTracers = []string{}
	cfg.EVM.RPCMethodTimeouts = []string{"eth_call=1s"}

	dir := t.TempDir()
	if err := WriteConfigToDir(cfg, dir); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}
	loaded, err := ReadConfigFromDir(dir)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if got := loaded.EVM.TraceAllowedTracers; got == nil || len(got) != 0 {
		t.Errorf("trace_allowed_tracers: got %#v, want empty non-nil", got)
	}
	if got := loaded.EVM.RPCMethodTimeouts; !reflect.DeepEqual(got, []string{"eth_call=1s"}) {
		t.Errorf("rpc_method_timeouts: got %v, want [eth_call=1s]", got)
	}
}

func TestEVMNewKeys_Overrides(t *testing.T) {
	cfg := DefaultForMode(ModeFull)
	err := ApplyOverrides(cfg, map[string]string{
		"evm.rate_limiting_enabled": "false",
		"evm.ip_rate_limit_rps":     "50.5",
		"evm.ip_rate_limit_burst":   "300",
		"evm.trusted_proxy_cidrs":   "10.0.0.0/8,172.16.0.0/12",
		"evm.rpc_method_timeouts":   "eth_call=10s",
		"evm.ws_admission_timeout":  "15s",
	})
	if err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	e := cfg.EVM
	if e.RateLimitingEnabled {
		t.Error("rate_limiting_enabled: got true, want false")
	}
	if e.IPRateLimitRPS != 50.5 {
		t.Errorf("ip_rate_limit_rps: got %v, want 50.5", e.IPRateLimitRPS)
	}
	if e.IPRateLimitBurst != 300 {
		t.Errorf("ip_rate_limit_burst: got %d, want 300", e.IPRateLimitBurst)
	}
	if !reflect.DeepEqual(e.TrustedProxyCIDRs, []string{"10.0.0.0/8", "172.16.0.0/12"}) {
		t.Errorf("trusted_proxy_cidrs: got %v", e.TrustedProxyCIDRs)
	}
	if !reflect.DeepEqual(e.RPCMethodTimeouts, []string{"eth_call=10s"}) {
		t.Errorf("rpc_method_timeouts: got %v", e.RPCMethodTimeouts)
	}
	if e.WSAdmissionTimeout.Duration != 15*time.Second {
		t.Errorf("ws_admission_timeout: got %s, want 15s", e.WSAdmissionTimeout)
	}
}

func TestValidateEVM_NewKeys(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(e *EVMConfig)
		wantErr bool
	}{
		{"defaults", func(e *EVMConfig) {}, false},
		{"zero max_filters", func(e *EVMConfig) { e.MaxFilters = 0 }, true},
		{"zero max_block_filter_hashes", func(e *EVMConfig) { e.MaxBlockFilterHashes = 0 }, true},
		{"negative max_open_connections", func(e *EVMConfig) { e.MaxOpenConnections = -1 }, true},
		{"negative body_read_idle_timeout", func(e *EVMConfig) { e.BodyReadIdleTimeout = Dur(-time.Second) }, true},
		{"negative rpc_default_timeout", func(e *EVMConfig) { e.RPCDefaultTimeout = Dur(-time.Second) }, true},
		{"js tracer name in allowlist", func(e *EVMConfig) { e.TraceAllowedTracers = []string{"myJSTracer"} }, true},
		{"unknown bake tracer", func(e *EVMConfig) { e.TraceBakeTracers = []string{"nope"} }, true},
		{"empty tracer allowlist", func(e *EVMConfig) { e.TraceAllowedTracers = []string{} }, false},
		{"invalid trusted proxy CIDR", func(e *EVMConfig) { e.TrustedProxyCIDRs = []string{"10.0.0.1"} }, true},
		{"valid trusted proxy CIDRs", func(e *EVMConfig) { e.TrustedProxyCIDRs = []string{"10.0.0.0/8", "::1/128"} }, false},
		{"method timeout missing =", func(e *EVMConfig) { e.RPCMethodTimeouts = []string{"eth_call"} }, true},
		{"method timeout bad duration", func(e *EVMConfig) { e.RPCMethodTimeouts = []string{"eth_call=soon"} }, true},
		{"method timeout negative", func(e *EVMConfig) { e.RPCMethodTimeouts = []string{"eth_call=-1s"} }, true},
		{"method timeout bare wildcard", func(e *EVMConfig) { e.RPCMethodTimeouts = []string{"*=1s"} }, true},
		{"method timeout inner wildcard", func(e *EVMConfig) { e.RPCMethodTimeouts = []string{"debug_*trace=1s"} }, true},
		{"method timeout prefix and unitless", func(e *EVMConfig) {
			e.RPCMethodTimeouts = []string{"debug_trace*=0", "eth_call=5"}
		}, false},
		{"burst below batch limit", func(e *EVMConfig) { e.IPRateLimitBurst = 50; e.BatchRequestLimit = 100 }, true},
		{"burst below batch limit, rate limiting off", func(e *EVMConfig) {
			e.IPRateLimitBurst = 50
			e.BatchRequestLimit = 100
			e.RateLimitingEnabled = false
		}, false},
		{"burst zero disables bucket", func(e *EVMConfig) { e.IPRateLimitBurst = 0 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultForMode(ModeFull)
			tt.mutate(&cfg.EVM)
			if got := Validate(cfg).HasErrors(); got != tt.wantErr {
				t.Errorf("HasErrors: got %v, want %v (diagnostics: %v)", got, tt.wantErr, Validate(cfg).Diagnostics)
			}
		})
	}
}

// stripAppTomlKeys removes the given keys from homeDir/config/app.toml,
// simulating a file rendered before they existed.
func stripAppTomlKeys(t *testing.T, homeDir string, keys []string) {
	t.Helper()
	path := filepath.Join(homeDir, configDir, appTomlFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read app.toml: %v", err)
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	var kept []string
	removed := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		key, _, _ := strings.Cut(strings.TrimSpace(line), "=")
		if drop[strings.TrimSpace(key)] {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed != len(keys) {
		t.Fatalf("stripped %d keys from app.toml, want %d", removed, len(keys))
	}
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatalf("write app.toml: %v", err)
	}
}
