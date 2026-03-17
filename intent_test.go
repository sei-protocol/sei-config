package seiconfig

import (
	"testing"
)

func TestValidateIntent_ValidModes(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive}
	for _, mode := range modes {
		result := ValidateIntent(ConfigIntent{Mode: mode})
		if !result.Valid {
			t.Errorf("mode %q: expected valid, got diagnostics: %v", mode, result.Diagnostics)
		}
		if result.Mode != mode {
			t.Errorf("mode %q: expected mode in result, got %q", mode, result.Mode)
		}
		if result.Version != CurrentVersion {
			t.Errorf("mode %q: expected version %d, got %d", mode, CurrentVersion, result.Version)
		}
	}
}

func TestValidateIntent_InvalidMode(t *testing.T) {
	result := ValidateIntent(ConfigIntent{Mode: "bogus"})
	if result.Valid {
		t.Fatal("expected invalid result for bogus mode")
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Field == "mode" && d.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error-level diagnostic on 'mode' field")
	}
}

func TestValidateIntent_EmptyModeNonIncremental(t *testing.T) {
	result := ValidateIntent(ConfigIntent{Mode: ""})
	if result.Valid {
		t.Fatal("expected invalid result for empty mode on non-incremental intent")
	}
}

func TestValidateIntent_EmptyModeIncremental(t *testing.T) {
	result := ValidateIntent(ConfigIntent{Mode: "", Incremental: true})
	if !result.Valid {
		t.Errorf("incremental with empty mode should be valid, got: %v", result.Diagnostics)
	}
}

func TestValidateIntent_TargetVersionTooHigh(t *testing.T) {
	result := ValidateIntent(ConfigIntent{
		Mode:          ModeFull,
		TargetVersion: CurrentVersion + 10,
	})
	if result.Valid {
		t.Fatal("expected invalid for version exceeding CurrentVersion")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Field == "targetVersion" && d.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error on targetVersion field")
	}
}

func TestValidateIntent_ExplicitTargetVersion(t *testing.T) {
	result := ValidateIntent(ConfigIntent{
		Mode:          ModeFull,
		TargetVersion: 1,
	})
	if !result.Valid {
		t.Errorf("expected valid for explicit version 1, got: %v", result.Diagnostics)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1 in result, got %d", result.Version)
	}
}

func TestValidateIntent_ZeroVersionDefaultsToCurrent(t *testing.T) {
	result := ValidateIntent(ConfigIntent{
		Mode:          ModeFull,
		TargetVersion: 0,
	})
	if result.Version != CurrentVersion {
		t.Errorf("expected CurrentVersion (%d), got %d", CurrentVersion, result.Version)
	}
}

func TestValidateIntent_UnknownOverrideKey(t *testing.T) {
	result := ValidateIntent(ConfigIntent{
		Mode: ModeFull,
		Overrides: map[string]string{
			"totally.fake.key": "value",
		},
	})
	if result.Valid {
		t.Fatal("expected invalid for unknown override key")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Field == "overrides.totally.fake.key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected diagnostic referencing the unknown override key")
	}
}

func TestValidateIntent_ValidOverrideKey(t *testing.T) {
	result := ValidateIntent(ConfigIntent{
		Mode: ModeFull,
		Overrides: map[string]string{
			"evm.http_port": "9545",
		},
	})
	if !result.Valid {
		t.Errorf("expected valid with known override key, got: %v", result.Diagnostics)
	}
}

// ---------------------------------------------------------------------------
// ResolveIntent tests
// ---------------------------------------------------------------------------

func TestResolveIntent_AllModes(t *testing.T) {
	modes := []NodeMode{ModeValidator, ModeFull, ModeSeed, ModeArchive}
	for _, mode := range modes {
		result, err := ResolveIntent(ConfigIntent{Mode: mode})
		if err != nil {
			t.Errorf("mode %q: unexpected error: %v", mode, err)
			continue
		}
		if !result.Valid {
			t.Errorf("mode %q: expected valid, got diagnostics: %v", mode, result.Diagnostics)
			continue
		}
		if result.Config == nil {
			t.Errorf("mode %q: expected non-nil config", mode)
			continue
		}
		if result.Config.Mode != mode {
			t.Errorf("mode %q: config.Mode = %q", mode, result.Config.Mode)
		}
		if result.Version != CurrentVersion {
			t.Errorf("mode %q: expected version %d, got %d", mode, CurrentVersion, result.Version)
		}
	}
}

func TestResolveIntent_EmptyModeError(t *testing.T) {
	_, err := ResolveIntent(ConfigIntent{Mode: ""})
	if err == nil {
		t.Fatal("expected error for empty mode")
	}
}

func TestResolveIntent_InvalidModeError(t *testing.T) {
	_, err := ResolveIntent(ConfigIntent{Mode: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestResolveIntent_WithOverrides(t *testing.T) {
	result, err := ResolveIntent(ConfigIntent{
		Mode: ModeFull,
		Overrides: map[string]string{
			"evm.http_port": "9545",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got: %v", result.Diagnostics)
	}
	if result.Config.EVM.HTTPPort != 9545 {
		t.Errorf("expected HTTPPort 9545, got %d", result.Config.EVM.HTTPPort)
	}
}

func TestResolveIntent_OverrideThatCausesValidationError(t *testing.T) {
	result, err := ResolveIntent(ConfigIntent{
		Mode: ModeFull,
		Overrides: map[string]string{
			"chain.min_gas_prices": "",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid when override clears a required field")
	}
	if result.Config != nil {
		t.Error("expected nil config when result is invalid")
	}
}

func TestResolveIntent_ExplicitTargetVersion(t *testing.T) {
	result, err := ResolveIntent(ConfigIntent{
		Mode:          ModeValidator,
		TargetVersion: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
}

func TestResolveIntent_ConfigVersionMatchesResultVersion(t *testing.T) {
	result, err := ResolveIntent(ConfigIntent{Mode: ModeFull})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Config.Version != result.Version {
		t.Errorf("result.Config.Version (%d) != result.Version (%d); "+
			"config written to disk would carry the wrong schema version",
			result.Config.Version, result.Version)
	}
}

func TestResolveIntent_ModeDefaultsApplied(t *testing.T) {
	archiveResult, err := ResolveIntent(ConfigIntent{Mode: ModeArchive})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seedResult, err := ResolveIntent(ConfigIntent{Mode: ModeSeed})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if archiveResult.Config.Storage.PruningStrategy == seedResult.Config.Storage.PruningStrategy {
		t.Error("expected different pruning strategies for archive (nothing) vs seed (everything)")
	}
}

// ---------------------------------------------------------------------------
// ResolveIncrementalIntent tests
// ---------------------------------------------------------------------------

func TestResolveIncrementalIntent_NilCurrentError(t *testing.T) {
	_, err := ResolveIncrementalIntent(ConfigIntent{}, nil)
	if err == nil {
		t.Fatal("expected error for nil current config")
	}
}

func TestResolveIncrementalIntent_PatchesExistingConfig(t *testing.T) {
	current := DefaultForMode(ModeFull)
	originalPort := current.EVM.HTTPPort

	result, err := ResolveIncrementalIntent(
		ConfigIntent{
			Overrides: map[string]string{
				"evm.http_port": "9999",
			},
		},
		current,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got: %v", result.Diagnostics)
	}
	if result.Config.EVM.HTTPPort != 9999 {
		t.Errorf("expected patched HTTPPort 9999, got %d", result.Config.EVM.HTTPPort)
	}
	if originalPort == 9999 {
		t.Error("test setup error: default port should not be 9999")
	}
}

func TestResolveIncrementalIntent_ModeOverride(t *testing.T) {
	current := DefaultForMode(ModeFull)
	result, err := ResolveIncrementalIntent(
		ConfigIntent{Mode: ModeArchive},
		current,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Config.Mode != ModeArchive {
		t.Errorf("expected mode archive, got %q", result.Config.Mode)
	}
	if result.Mode != ModeArchive {
		t.Errorf("expected result.Mode archive, got %q", result.Mode)
	}
}

func TestResolveIncrementalIntent_PreservesExistingValues(t *testing.T) {
	current := DefaultForMode(ModeValidator)
	current.Chain.Moniker = "my-validator"

	result, err := ResolveIncrementalIntent(
		ConfigIntent{
			Overrides: map[string]string{
				"evm.http_port": "7777",
			},
		},
		current,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Config.Chain.Moniker != "my-validator" {
		t.Errorf("expected moniker preserved, got %q", result.Config.Chain.Moniker)
	}
	if result.Config.EVM.HTTPPort != 7777 {
		t.Errorf("expected patched port, got %d", result.Config.EVM.HTTPPort)
	}
}

func TestResolveIncrementalIntent_DoesNotMutateCaller(t *testing.T) {
	current := DefaultForMode(ModeFull)
	originalMode := current.Mode
	originalPort := current.EVM.HTTPPort

	_, err := ResolveIncrementalIntent(
		ConfigIntent{
			Mode: ModeArchive,
			Overrides: map[string]string{
				"evm.http_port": "9999",
			},
		},
		current,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if current.Mode != originalMode {
		t.Errorf("caller's Mode was mutated: got %q, want %q", current.Mode, originalMode)
	}
	if current.EVM.HTTPPort != originalPort {
		t.Errorf("caller's HTTPPort was mutated: got %d, want %d", current.EVM.HTTPPort, originalPort)
	}
}

func TestResolveIncrementalIntent_InvalidPatchReturnsInvalid(t *testing.T) {
	current := DefaultForMode(ModeFull)

	result, err := ResolveIncrementalIntent(
		ConfigIntent{
			Overrides: map[string]string{
				"chain.min_gas_prices": "",
			},
		},
		current,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid after clearing required field")
	}
}

// ---------------------------------------------------------------------------
// ConfigResult helpers
// ---------------------------------------------------------------------------

func TestConfigResult_AddError(t *testing.T) {
	r := &ConfigResult{Valid: true}
	r.addError("test.field", "something went wrong")
	if r.Valid {
		t.Error("expected Valid to be false after addError")
	}
	if len(r.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(r.Diagnostics))
	}
	if r.Diagnostics[0].Severity != SeverityError {
		t.Error("expected error severity")
	}
}

func TestConfigResult_AddWarning(t *testing.T) {
	r := &ConfigResult{Valid: true}
	r.addWarning("test.field", "just a heads up")
	if !r.Valid {
		t.Error("warnings should not flip Valid to false")
	}
	if len(r.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(r.Diagnostics))
	}
}

// ---------------------------------------------------------------------------
// isZeroValue
// ---------------------------------------------------------------------------

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero int64", int64(0), true},
		{"non-zero int64", int64(1), false},
		{"zero uint", uint(0), true},
		{"non-zero uint", uint(1), false},
		{"zero uint64", uint64(0), true},
		{"non-zero uint64", uint64(1), false},
		{"zero float64", 0.0, true},
		{"non-zero float64", 3.14, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"unknown type", struct{}{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isZeroValue(tt.val)
			if got != tt.want {
				t.Errorf("isZeroValue(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveTargetVersion
// ---------------------------------------------------------------------------

func TestResolveTargetVersion(t *testing.T) {
	if got := resolveTargetVersion(0); got != CurrentVersion {
		t.Errorf("zero should default to CurrentVersion (%d), got %d", CurrentVersion, got)
	}
	if got := resolveTargetVersion(3); got != 3 {
		t.Errorf("explicit 3 should return 3, got %d", got)
	}
}
