package seiconfig

import (
	"strings"
	"testing"
)

func TestBuildRegistry_PopulatesFields(t *testing.T) {
	r := BuildRegistry()
	if r.Len() == 0 {
		t.Fatal("registry should have fields")
	}
	t.Logf("registry has %d fields", r.Len())
}

func TestBuildRegistry_KeysAreDotted(t *testing.T) {
	r := BuildRegistry()
	for _, f := range r.Fields() {
		if f.Key == "" {
			t.Error("field has empty key")
		}
		if f.Section == "" {
			t.Errorf("field %q has empty section", f.Key)
		}
	}
}

func TestBuildRegistry_EnvVarsHaveSEIPrefix(t *testing.T) {
	r := BuildRegistry()
	for _, f := range r.Fields() {
		if !strings.HasPrefix(f.EnvVar, "SEI_") {
			t.Errorf("field %q env var %q missing SEI_ prefix", f.Key, f.EnvVar)
		}
	}
}

func TestBuildRegistry_KnownFieldsExist(t *testing.T) {
	r := BuildRegistry()

	mustExist := []string{
		"version",
		"mode",
		"chain.chain_id",
		"chain.moniker",
		"chain.min_gas_prices",
		"network.rpc.listen_address",
		"network.p2p.max_connections",
		"consensus.create_empty_blocks",
		"mempool.size",
		"state_sync.enable",
		"storage.db_backend",
		"storage.pruning",
		"storage.state_commit.enable",
		"storage.state_store.keep_recent",
		"tx_index.indexer",
		"evm.http_enabled",
		"evm.http_port",
		"api.rest.enable",
		"api.grpc.enable",
		"metrics.enabled",
		"logging.level",
		"wasm.query_gas_limit",
		"giga_executor.enabled",
		"light_invariance.supply_enabled",
		"self_remediation.restart_cooldown_seconds",
	}

	for _, key := range mustExist {
		if r.Field(key) == nil {
			t.Errorf("expected field %q not found in registry", key)
		}
	}
}

func TestBuildRegistry_FieldTypes(t *testing.T) {
	r := BuildRegistry()

	tests := []struct {
		key      string
		wantType FieldType
	}{
		{"chain.moniker", FieldTypeString},
		{"chain.halt_height", FieldTypeUint},
		{"chain.occ_enabled", FieldTypeBool},
		{"evm.http_port", FieldTypeInt},
		{"mempool.drop_priority_threshold", FieldTypeFloat},
		{"network.rpc.timeout_broadcast_tx_commit", FieldTypeDuration},
		{"tx_index.indexer", FieldTypeStringSlice},
	}

	for _, tt := range tests {
		f := r.Field(tt.key)
		if f == nil {
			t.Errorf("field %q not found", tt.key)
			continue
		}
		if f.Type != tt.wantType {
			t.Errorf("field %q: got type %s, want %s", tt.key, f.Type, tt.wantType)
		}
	}
}

func TestBuildRegistry_EnvVarLookup(t *testing.T) {
	r := BuildRegistry()

	f := r.FieldByEnvVar("SEI_EVM_HTTP_PORT")
	if f == nil {
		t.Fatal("SEI_EVM_HTTP_PORT not found by env var lookup")
	}
	if f.Key != "evm.http_port" {
		t.Errorf("env var lookup: got key %q, want %q", f.Key, "evm.http_port")
	}
}

func TestBuildRegistry_NoDuplicateKeys(t *testing.T) {
	r := BuildRegistry()
	seen := make(map[string]bool)
	for _, f := range r.Fields() {
		if seen[f.Key] {
			t.Errorf("duplicate key: %q", f.Key)
		}
		seen[f.Key] = true
	}
}

func TestBuildRegistry_NoDuplicateEnvVars(t *testing.T) {
	r := BuildRegistry()
	seen := make(map[string]bool)
	for _, f := range r.Fields() {
		if seen[f.EnvVar] {
			t.Errorf("duplicate env var: %q (key: %q)", f.EnvVar, f.Key)
		}
		seen[f.EnvVar] = true
	}
}

func TestRegistry_Sections(t *testing.T) {
	r := BuildRegistry()
	sections := r.Sections()

	expectedSections := []string{
		"chain", "network", "consensus", "mempool", "state_sync",
		"storage", "tx_index", "evm", "api", "metrics", "logging",
		"wasm", "giga_executor", "light_invariance", "priv_validator",
		"self_remediation", "genesis",
	}

	sectionSet := make(map[string]bool)
	for _, s := range sections {
		sectionSet[s] = true
	}

	for _, expected := range expectedSections {
		if !sectionSet[expected] {
			t.Errorf("expected section %q not found", expected)
		}
	}
}

func TestRegistry_FieldsInSection(t *testing.T) {
	r := BuildRegistry()
	evmFields := r.FieldsInSection("evm")
	if len(evmFields) == 0 {
		t.Error("evm section should have fields")
	}
	for _, f := range evmFields {
		if f.Section != "evm" {
			t.Errorf("field %q has section %q, expected 'evm'", f.Key, f.Section)
		}
	}
}

func TestRegistry_Enrich(t *testing.T) {
	r := BuildRegistry()

	ok := r.Enrich("evm.http_port",
		WithDescription("Port for EVM HTTP"),
		WithHotReload(),
	)
	if !ok {
		t.Fatal("Enrich returned false for existing key")
	}

	f := r.Field("evm.http_port")
	if f.Description != "Port for EVM HTTP" {
		t.Errorf("description: got %q", f.Description)
	}
	if !f.HotReload {
		t.Error("expected HotReload to be true")
	}
}

func TestRegistry_Enrich_MissingKey(t *testing.T) {
	r := BuildRegistry()
	ok := r.Enrich("nonexistent.key", WithDescription("nope"))
	if ok {
		t.Error("Enrich should return false for missing key")
	}
}

func TestRegistry_EnrichAll(t *testing.T) {
	r := BuildRegistry()
	missing := r.EnrichAll(DefaultEnrichments())
	if len(missing) > 0 {
		t.Errorf("DefaultEnrichments has keys not found in registry: %v", missing)
	}
}

func TestRegistry_EnrichAll_PopulatesDescriptions(t *testing.T) {
	r := BuildRegistry()
	r.EnrichAll(DefaultEnrichments())

	f := r.Field("chain.min_gas_prices")
	if f == nil {
		t.Fatal("chain.min_gas_prices not found")
	}
	if f.Description == "" {
		t.Error("expected description after enrichment")
	}
}

func TestRegistry_HotReloadableFields(t *testing.T) {
	r := BuildRegistry()
	r.EnrichAll(DefaultEnrichments())

	hot := r.HotReloadableFields()
	if len(hot) == 0 {
		t.Error("expected at least one hot-reloadable field")
	}
	for _, f := range hot {
		if !f.HotReload {
			t.Errorf("field %q in hot-reloadable list but HotReload=false", f.Key)
		}
	}
}

func TestRegistry_DefaultsByMode(t *testing.T) {
	r := BuildRegistry()
	defaults := r.DefaultsByMode(ModeValidator)

	if len(defaults) == 0 {
		t.Fatal("DefaultsByMode returned empty map")
	}

	httpEnabled, ok := defaults["evm.http_enabled"]
	if !ok {
		t.Fatal("evm.http_enabled not in defaults")
	}
	if httpEnabled != false {
		t.Errorf("validator evm.http_enabled: got %v, want false", httpEnabled)
	}
}

func TestRegistry_DefaultsByMode_DiffersByMode(t *testing.T) {
	r := BuildRegistry()
	valDefaults := r.DefaultsByMode(ModeValidator)
	fullDefaults := r.DefaultsByMode(ModeFull)

	valHTTP := valDefaults["evm.http_enabled"]
	fullHTTP := fullDefaults["evm.http_enabled"]

	if valHTTP == fullHTTP {
		t.Errorf(
			"expected evm.http_enabled to differ between validator (%v) and full (%v)",
			valHTTP, fullHTTP)
	}
}
