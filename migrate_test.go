package seiconfig

import (
	"strings"
	"testing"
)

func TestMigrationRegistry_Empty(t *testing.T) {
	r, err := NewMigrationRegistry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.MaxVersion() != 0 {
		t.Errorf("empty registry max version: got %d, want 0", r.MaxVersion())
	}
}

func TestMigrationRegistry_SingleMigration(t *testing.T) {
	r, err := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "test migration",
		Migrate: func(cfg *SeiConfig) error {
			cfg.Version = 2
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.MaxVersion() != 2 {
		t.Errorf("max version: got %d, want 2", r.MaxVersion())
	}
}

func TestMigrationRegistry_ChainedMigrations(t *testing.T) {
	r, err := NewMigrationRegistry(
		Migration{
			FromVersion: 1,
			ToVersion:   2,
			Description: "v1 to v2",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 2
				return nil
			},
		},
		Migration{
			FromVersion: 2,
			ToVersion:   3,
			Description: "v2 to v3",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 3
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.MaxVersion() != 3 {
		t.Errorf("max version: got %d, want 3", r.MaxVersion())
	}
}

func TestMigrationRegistry_RejectsBadToVersion(t *testing.T) {
	_, err := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   3, // should be 2
		Description: "bad",
		Migrate:     func(cfg *SeiConfig) error { return nil },
	})
	if err == nil {
		t.Fatal("expected error for non-sequential ToVersion")
	}
}

func TestMigrationRegistry_RejectsDuplicate(t *testing.T) {
	_, err := NewMigrationRegistry(
		Migration{
			FromVersion: 1,
			ToVersion:   2,
			Description: "first",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 2
				return nil
			},
		},
		Migration{
			FromVersion: 1,
			ToVersion:   2,
			Description: "duplicate",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 2
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("expected error for duplicate FromVersion")
	}
}

func TestMigrationRegistry_RejectsGap(t *testing.T) {
	_, err := NewMigrationRegistry(
		Migration{
			FromVersion: 1,
			ToVersion:   2,
			Description: "v1 to v2",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 2
				return nil
			},
		},
		Migration{
			FromVersion: 3, // gap: missing v2->v3
			ToVersion:   4,
			Description: "v3 to v4",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 4
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("expected error for chain gap")
	}
}

func TestMigrateConfig_NoOp(t *testing.T) {
	r, _ := NewMigrationRegistry()

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	result, err := r.MigrateConfig(cfg, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected no applied migrations, got %d", len(result.Applied))
	}
}

func TestMigrateConfig_SingleStep(t *testing.T) {
	r, _ := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "add new default",
		Migrate: func(cfg *SeiConfig) error {
			cfg.Version = 2
			return nil
		},
	})

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	result, err := r.MigrateConfig(cfg, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 2 {
		t.Errorf("config version: got %d, want 2", cfg.Version)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("applied migrations: got %d, want 1", len(result.Applied))
	}
	if result.Applied[0].Description != "add new default" {
		t.Errorf("description: got %q", result.Applied[0].Description)
	}
	if result.FromVersion != 1 || result.ToVersion != 2 {
		t.Errorf("result: from=%d to=%d", result.FromVersion, result.ToVersion)
	}
}

func TestMigrateConfig_MultiStep(t *testing.T) {
	r, _ := NewMigrationRegistry(
		Migration{
			FromVersion: 1,
			ToVersion:   2,
			Description: "step 1",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 2
				return nil
			},
		},
		Migration{
			FromVersion: 2,
			ToVersion:   3,
			Description: "step 2",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 3
				return nil
			},
		},
		Migration{
			FromVersion: 3,
			ToVersion:   4,
			Description: "step 3",
			Migrate: func(cfg *SeiConfig) error {
				cfg.Version = 4
				return nil
			},
		},
	)

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	result, err := r.MigrateConfig(cfg, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 4 {
		t.Errorf("config version: got %d, want 4", cfg.Version)
	}
	if len(result.Applied) != 3 {
		t.Errorf("applied migrations: got %d, want 3", len(result.Applied))
	}
}

func TestMigrateConfig_RejectsDowngrade(t *testing.T) {
	r, _ := NewMigrationRegistry()

	cfg := Default()
	cfg.Version = 3
	cfg.Mode = ModeValidator

	_, err := r.MigrateConfig(cfg, 2)
	if err == nil {
		t.Fatal("expected error for downgrade")
	}
	if !strings.Contains(err.Error(), "downgrade") {
		t.Errorf("error should mention downgrade: %v", err)
	}
}

func TestMigrateConfig_MissingMigration(t *testing.T) {
	r, _ := NewMigrationRegistry()

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	_, err := r.MigrateConfig(cfg, 2)
	if err == nil {
		t.Fatal("expected error for missing migration")
	}
}

func TestMigrateConfig_MigrationFuncError(t *testing.T) {
	r, _ := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "fails",
		Migrate: func(_ *SeiConfig) error {
			return &migrationTestError{}
		},
	})

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	_, err := r.MigrateConfig(cfg, 2)
	if err == nil {
		t.Fatal("expected error from failing migration")
	}
}

type migrationTestError struct{}

func (e *migrationTestError) Error() string { return "intentional test error" }

func TestMigrateConfig_ValidationAfterMigration(t *testing.T) {
	r, _ := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "sets invalid mode",
		Migrate: func(cfg *SeiConfig) error {
			cfg.Version = 2
			cfg.Mode = "bogus"
			return nil
		},
	})

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	_, err := r.MigrateConfig(cfg, 2)
	if err == nil {
		t.Fatal("expected validation error after migration produces invalid config")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("error should mention validation: %v", err)
	}
}

func TestMigrateConfig_VersionNotUpdated(t *testing.T) {
	r, _ := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "forgets to update version",
		Migrate: func(_ *SeiConfig) error {
			// intentionally does not set cfg.Version = 2
			return nil
		},
	})

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator

	_, err := r.MigrateConfig(cfg, 2)
	if err == nil {
		t.Fatal("expected error when migration doesn't update version")
	}
	if !strings.Contains(err.Error(), "did not update") {
		t.Errorf("error should mention version not updated: %v", err)
	}
}

func TestMigrateConfig_PreservesFieldValues(t *testing.T) {
	r, _ := NewMigrationRegistry(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "schema-only change",
		Migrate: func(cfg *SeiConfig) error {
			cfg.Version = 2
			return nil
		},
	})

	cfg := Default()
	cfg.Version = 1
	cfg.Mode = ModeValidator
	cfg.Chain.Moniker = "my-custom-node"
	cfg.EVM.HTTPPort = 9999

	result, err := r.MigrateConfig(cfg, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Chain.Moniker != "my-custom-node" {
		t.Errorf("moniker was lost: got %q", cfg.Chain.Moniker)
	}
	if cfg.EVM.HTTPPort != 9999 {
		t.Errorf("evm port was lost: got %d", cfg.EVM.HTTPPort)
	}
	if result.ToVersion != 2 {
		t.Errorf("result ToVersion: got %d", result.ToVersion)
	}
}

func TestNeedsUpgrade(t *testing.T) {
	r, _ := NewMigrationRegistry()

	if !r.NeedsUpgrade(1, 2) {
		t.Error("1 → 2 should need upgrade")
	}
	if r.NeedsUpgrade(2, 2) {
		t.Error("2 → 2 should not need upgrade")
	}
	if r.NeedsUpgrade(3, 2) {
		t.Error("3 → 2 should not need upgrade")
	}
}

func TestDefaultMigrations_Valid(t *testing.T) {
	_, err := NewMigrationRegistry(DefaultMigrations()...)
	if err != nil {
		t.Fatalf("DefaultMigrations failed to register: %v", err)
	}
}

// v1ToV2Migration returns the v1→v2 migration from DefaultMigrations for tests
// that exercise the rename transform directly (bypassing post-migration
// validation, which rejects unknown/deprecated WriteMode values).
func v1ToV2Migration(t *testing.T) Migration {
	t.Helper()
	for _, m := range DefaultMigrations() {
		if m.FromVersion == 1 && m.ToVersion == 2 {
			return m
		}
	}
	t.Fatal("DefaultMigrations missing v1→v2 migration")
	return Migration{}
}

// TestMigrateConfig_WriteModeRoundTrip runs the real v1→v2 migration through
// the registry pipeline (including post-migration validation) and asserts the
// deprecated cosmos_only write mode is renamed to memiavl_only in both stores.
func TestMigrateConfig_WriteModeRoundTrip(t *testing.T) {
	r, err := NewMigrationRegistry(DefaultMigrations()...)
	if err != nil {
		t.Fatalf("NewMigrationRegistry: %v", err)
	}

	cfg := DefaultForMode(ModeFull)
	cfg.Version = 1
	cfg.Storage.StateCommit.WriteMode = WriteModeCosmosOnly
	cfg.Storage.StateStore.WriteMode = WriteModeCosmosOnly

	result, err := r.MigrateConfig(cfg, 2)
	if err != nil {
		t.Fatalf("MigrateConfig: %v", err)
	}

	if cfg.Version != 2 {
		t.Errorf("version: got %d, want 2", cfg.Version)
	}
	if cfg.Storage.StateCommit.WriteMode != WriteModeMemiavlOnly {
		t.Errorf("state_commit.write_mode: got %q, want %q",
			cfg.Storage.StateCommit.WriteMode, WriteModeMemiavlOnly)
	}
	if cfg.Storage.StateStore.WriteMode != WriteModeMemiavlOnly {
		t.Errorf("state_store.write_mode: got %q, want %q",
			cfg.Storage.StateStore.WriteMode, WriteModeMemiavlOnly)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("applied migrations: got %d, want 1", len(result.Applied))
	}
}

// TestV1ToV2_WriteModeRename covers every deprecated v1 write mode and asserts
// it maps to the expected v2 value. The unknown-value case asserts pass-through
// (the migration only renames known values; validation handles the rest).
func TestV1ToV2_WriteModeRename(t *testing.T) {
	m := v1ToV2Migration(t)

	tests := []struct {
		name string
		in   WriteMode
		want WriteMode
	}{
		{"cosmos_only renames to memiavl_only", WriteModeCosmosOnly, WriteModeMemiavlOnly},
		{"dual_write renames to migrate_evm", WriteModeDualWrite, WriteModeMigrateEVM},
		{"split_write renames to evm_migrated", WriteModeSplitWrite, WriteModeEVMMigrated},
		{"already-v2 memiavl_only is preserved", WriteModeMemiavlOnly, WriteModeMemiavlOnly},
		{"unknown value passes through unchanged", WriteMode("future_mode"), WriteMode("future_mode")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultForMode(ModeFull)
			cfg.Version = 1
			cfg.Storage.StateCommit.WriteMode = tc.in
			cfg.Storage.StateStore.WriteMode = tc.in

			if err := m.Migrate(cfg); err != nil {
				t.Fatalf("Migrate: %v", err)
			}

			if cfg.Version != 2 {
				t.Errorf("version: got %d, want 2", cfg.Version)
			}
			if cfg.Storage.StateCommit.WriteMode != tc.want {
				t.Errorf("state_commit.write_mode: got %q, want %q",
					cfg.Storage.StateCommit.WriteMode, tc.want)
			}
			if cfg.Storage.StateStore.WriteMode != tc.want {
				t.Errorf("state_store.write_mode: got %q, want %q",
					cfg.Storage.StateStore.WriteMode, tc.want)
			}
		})
	}
}
