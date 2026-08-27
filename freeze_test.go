package seiconfig

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFreezeHeight_RoundTrip confirms chain.freeze_height renders to the
// app.toml key seid actually reads (freeze-height) and survives a read back.
func TestFreezeHeight_RoundTrip(t *testing.T) {
	home := t.TempDir()
	cfg := DefaultForMode(ModeFull)
	cfg.Chain.FreezeHeight = 12345

	if err := WriteConfigToDir(cfg, home); err != nil {
		t.Fatalf("WriteConfigToDir: %v", err)
	}

	app, err := os.ReadFile(filepath.Join(home, configDir, appTomlFile))
	if err != nil {
		t.Fatalf("ReadFile app.toml: %v", err)
	}
	if !strings.Contains(string(app), "freeze-height") {
		t.Error("app.toml must carry the legacy key freeze-height; seid reads that spelling")
	}

	got, err := ReadConfigFromDir(home)
	if err != nil {
		t.Fatalf("ReadConfigFromDir: %v", err)
	}
	if got.Chain.FreezeHeight != 12345 {
		t.Errorf("freeze_height: got %d, want 12345", got.Chain.FreezeHeight)
	}
}

// TestFreezeHeight_Override covers the path the controller and seictl use: a
// dotted override key resolved against the unified schema.
func TestFreezeHeight_Override(t *testing.T) {
	cfg := Default()
	if err := ApplyOverrides(cfg, map[string]string{
		"chain.freeze_height": "5000000",
	}); err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if cfg.Chain.FreezeHeight != 5000000 {
		t.Errorf("freeze_height: got %d, want 5000000", cfg.Chain.FreezeHeight)
	}
}

// TestFreezeHeight_Validate mirrors seid's own ValidateFreeze rules, so an
// invalid combination is reported here instead of failing at node start.
func TestFreezeHeight_Validate(t *testing.T) {
	tests := []struct {
		name         string
		freezeHeight uint64
		haltHeight   uint64
		haltTime     uint64
		wantErr      bool
	}{
		{name: "unset", wantErr: false},
		{name: "freeze alone", freezeHeight: 100, wantErr: false},
		{name: "halt alone", haltHeight: 100, wantErr: false},
		{name: "freeze with halt height", freezeHeight: 100, haltHeight: 200, wantErr: true},
		{name: "freeze with halt time", freezeHeight: 100, haltTime: 200, wantErr: true},
		{name: "height overflow", freezeHeight: uint64(math.MaxInt64) + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultForMode(ModeFull)
			cfg.Chain.FreezeHeight = tt.freezeHeight
			cfg.Chain.HaltHeight = tt.haltHeight
			cfg.Chain.HaltTime = tt.haltTime

			if got := Validate(cfg).HasErrors(); got != tt.wantErr {
				t.Errorf("HasErrors: got %v, want %v (%v)", got, tt.wantErr, Validate(cfg).Diagnostics)
			}
		})
	}
}
