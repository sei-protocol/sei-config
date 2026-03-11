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
