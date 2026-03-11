package seiconfig

import (
	"fmt"
	"sort"
)

// Migration transforms a config from one schema version to the next.
// Each migration handles exactly one version transition (e.g. v1 → v2).
type Migration struct {
	// FromVersion is the version this migration reads.
	FromVersion int

	// ToVersion is the version this migration produces. Must equal FromVersion + 1.
	ToVersion int

	// Description is a human-readable summary of what changed.
	Description string

	// Migrate transforms the config in-place. The config's Version field
	// will already be set to FromVersion when this is called. The function
	// must set cfg.Version = ToVersion before returning.
	Migrate func(cfg *SeiConfig) error
}

// MigrationRegistry holds all known migrations and provides sequential
// migration from any version to the current version.
type MigrationRegistry struct {
	migrations map[int]*Migration // keyed by FromVersion
	maxVersion int
}

// NewMigrationRegistry creates a registry and registers the given migrations.
// It validates that migrations form a contiguous chain from their lowest
// version to their highest.
func NewMigrationRegistry(migrations ...Migration) (*MigrationRegistry, error) {
	r := &MigrationRegistry{
		migrations: make(map[int]*Migration),
	}
	for i := range migrations {
		m := &migrations[i]
		if m.ToVersion != m.FromVersion+1 {
			return nil, fmt.Errorf(
				"migration v%d→v%d: ToVersion must equal FromVersion + 1",
				m.FromVersion, m.ToVersion)
		}
		if _, exists := r.migrations[m.FromVersion]; exists {
			return nil, fmt.Errorf(
				"duplicate migration from version %d", m.FromVersion)
		}
		r.migrations[m.FromVersion] = m
		if m.ToVersion > r.maxVersion {
			r.maxVersion = m.ToVersion
		}
	}

	if err := r.validateChain(); err != nil {
		return nil, err
	}

	return r, nil
}

// validateChain ensures migrations form a contiguous sequence.
func (r *MigrationRegistry) validateChain() error {
	if len(r.migrations) == 0 {
		return nil
	}

	versions := make([]int, 0, len(r.migrations))
	for v := range r.migrations {
		versions = append(versions, v)
	}
	sort.Ints(versions)

	for i := 1; i < len(versions); i++ {
		if versions[i] != versions[i-1]+1 {
			return fmt.Errorf(
				"migration chain gap: have v%d→v%d and v%d→v%d but nothing from v%d",
				versions[i-1], versions[i-1]+1,
				versions[i], versions[i]+1,
				versions[i-1]+1)
		}
	}
	return nil
}

// MigrateConfig runs all migrations needed to bring cfg from its current
// version to targetVersion. Returns an error if:
//   - cfg.Version > targetVersion (downgrade not supported)
//   - a required migration is missing
//   - any migration function fails
//   - the final config fails validation
func (r *MigrationRegistry) MigrateConfig(cfg *SeiConfig, targetVersion int) (*MigrateResult, error) {
	result := &MigrateResult{
		FromVersion: cfg.Version,
		ToVersion:   cfg.Version,
	}

	if cfg.Version == targetVersion {
		return result, nil
	}
	if cfg.Version > targetVersion {
		return nil, fmt.Errorf(
			"config version %d is newer than target %d; downgrade is not supported",
			cfg.Version, targetVersion)
	}

	for cfg.Version < targetVersion {
		m, ok := r.migrations[cfg.Version]
		if !ok {
			return nil, fmt.Errorf(
				"no migration registered for version %d → %d",
				cfg.Version, cfg.Version+1)
		}

		if err := m.Migrate(cfg); err != nil {
			return nil, fmt.Errorf(
				"migration v%d→v%d failed: %w",
				m.FromVersion, m.ToVersion, err)
		}

		if cfg.Version != m.ToVersion {
			return nil, fmt.Errorf(
				"migration v%d→v%d did not update cfg.Version (got %d)",
				m.FromVersion, m.ToVersion, cfg.Version)
		}

		result.Applied = append(result.Applied, AppliedMigration{
			FromVersion: m.FromVersion,
			ToVersion:   m.ToVersion,
			Description: m.Description,
		})
	}

	result.ToVersion = cfg.Version

	vr := ValidateWithOpts(cfg, ValidateOpts{MaxVersion: targetVersion})
	if vr.HasErrors() {
		return nil, fmt.Errorf(
			"config fails validation after migration to v%d: %v",
			cfg.Version, vr.Errors())
	}
	result.Diagnostics = vr.Diagnostics

	return result, nil
}

// NeedsUpgrade returns true if the config version is below targetVersion.
func (r *MigrationRegistry) NeedsUpgrade(currentVersion, targetVersion int) bool {
	return currentVersion < targetVersion
}

// MaxVersion returns the highest version any registered migration produces.
func (r *MigrationRegistry) MaxVersion() int {
	return r.maxVersion
}

// MigrateResult captures the outcome of a migration run.
type MigrateResult struct {
	FromVersion int
	ToVersion   int
	Applied     []AppliedMigration
	Diagnostics []Diagnostic
}

// AppliedMigration records a single migration step that was executed.
type AppliedMigration struct {
	FromVersion int
	ToVersion   int
	Description string
}

// ---------------------------------------------------------------------------
// Default migration registry
// ---------------------------------------------------------------------------

// DefaultMigrations returns the set of all known migrations for the sei-config
// schema. Currently empty since v1 is the initial version — migrations will be
// added here as the schema evolves.
//
// Example of a future migration:
//
//	Migration{
//	    FromVersion: 1,
//	    ToVersion:   2,
//	    Description: "Rename evm.checktx_timeout to evm.check_tx_timeout",
//	    Migrate: func(cfg *SeiConfig) error {
//	        // Field was renamed; value is preserved by the struct.
//	        cfg.Version = 2
//	        return nil
//	    },
//	}
func DefaultMigrations() []Migration {
	return []Migration{}
}
