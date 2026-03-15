package seiconfig

import (
	"fmt"
	"slices"
)

// ConfigIntent declares the desired configuration state for a Sei node.
// It is the portable contract between the controller, sidecar, and CLI.
// sei-config owns the full resolution pipeline: intent -> validated SeiConfig.
//
// The controller builds an intent from the CRD spec, the sidecar resolves it.
// The controller never calls DefaultForMode, ApplyOverrides, or Validate
// directly — it just constructs an intent and sends it through.
type ConfigIntent struct {
	// Mode is the node's operating role (validator, full, seed, archive, rpc, indexer).
	Mode NodeMode `json:"mode"`

	// Overrides is a flat map of dotted TOML key paths to string values.
	// These are applied on top of mode defaults.
	Overrides map[string]string `json:"overrides,omitempty"`

	// TargetVersion is the desired config schema version.
	// When zero, uses CurrentVersion (the latest known by this library).
	// Set explicitly when deploying a custom binary that expects a specific
	// config version.
	TargetVersion int `json:"targetVersion,omitempty"`

	// Incremental means "read existing on-disk config and patch it" rather
	// than "generate from mode defaults." Used for day-2 changes.
	Incremental bool `json:"incremental,omitempty"`
}

// ConfigResult is the output of intent resolution. It contains the resolved
// config, diagnostics, and a validity flag.
type ConfigResult struct {
	// Config is the fully resolved SeiConfig. Nil when Valid is false.
	Config *SeiConfig `json:"config,omitempty"`

	// Version is the config schema version of the resolved config.
	Version int `json:"version"`

	// Mode is the mode of the resolved config.
	Mode NodeMode `json:"mode"`

	// Diagnostics contains all validation findings (errors, warnings, info).
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Valid is true when no error-level diagnostics exist.
	Valid bool `json:"valid"`
}

func (r *ConfigResult) addError(field, msg string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{SeverityError, field, msg})
	r.Valid = false
}

func (r *ConfigResult) addWarning(field, msg string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{SeverityWarning, field, msg})
}

// ValidateIntent checks whether a ConfigIntent is well-formed without
// producing a resolved config. This enables dry-run validation by the
// controller before submitting a task to the sidecar.
//
// Checks performed:
//   - Mode is valid
//   - TargetVersion is within the supported range
//   - All override keys exist in the Registry
//   - Version-required fields for the mode are satisfied
func ValidateIntent(intent ConfigIntent) *ConfigResult {
	result := &ConfigResult{
		Version: resolveTargetVersion(intent.TargetVersion),
		Mode:    intent.Mode,
		Valid:   true,
	}

	if !intent.Incremental {
		validateIntentMode(result, intent)
	}
	validateIntentVersion(result, intent)

	registry := BuildRegistry()
	registry.EnrichAll(DefaultEnrichments())
	validateIntentOverrideKeys(result, intent, registry)
	validateIntentRequiredFields(result, intent, registry)

	return result
}

// ResolveIntent produces a fully resolved, validated SeiConfig from an intent.
// This is the primary entry point for non-incremental (bootstrap) config
// generation. The full pipeline is:
//
//  1. Resolve target version
//  2. Generate mode defaults
//  3. Apply overrides
//  4. Validate the result
//  5. Return ConfigResult
func ResolveIntent(intent ConfigIntent) (*ConfigResult, error) {
	result := &ConfigResult{
		Version: resolveTargetVersion(intent.TargetVersion),
		Mode:    intent.Mode,
		Valid:   true,
	}

	if intent.Mode == "" {
		return nil, fmt.Errorf("mode is required for non-incremental config resolution")
	}
	if !intent.Mode.IsValid() {
		return nil, fmt.Errorf("invalid mode %q", intent.Mode)
	}

	cfg := DefaultForMode(intent.Mode)
	cfg.Version = result.Version

	if err := ApplyOverrides(cfg, intent.Overrides); err != nil {
		return nil, fmt.Errorf("applying overrides: %w", err)
	}

	vr := ValidateWithOpts(cfg, ValidateOpts{MaxVersion: result.Version})
	result.Diagnostics = vr.Diagnostics
	result.Valid = !vr.HasErrors()

	if result.Valid {
		result.Config = cfg
	}

	return result, nil
}

// ResolveIncrementalIntent resolves an incremental intent against an existing
// on-disk config. Used for day-2 patches where the base config already exists.
func ResolveIncrementalIntent(intent ConfigIntent, current *SeiConfig) (*ConfigResult, error) {
	if current == nil {
		return nil, fmt.Errorf("current config is required for incremental resolution")
	}

	result := &ConfigResult{
		Version: resolveTargetVersion(intent.TargetVersion),
		Valid:   true,
	}

	copied := *current
	cfg := &copied
	if intent.Mode != "" {
		cfg.Mode = intent.Mode
	}
	result.Mode = cfg.Mode

	if err := ApplyOverrides(cfg, intent.Overrides); err != nil {
		return nil, fmt.Errorf("applying incremental overrides: %w", err)
	}

	vr := ValidateWithOpts(cfg, ValidateOpts{MaxVersion: result.Version})
	result.Diagnostics = vr.Diagnostics
	result.Valid = !vr.HasErrors()

	if result.Valid {
		result.Config = cfg
		result.Version = cfg.Version
	}

	return result, nil
}

func resolveTargetVersion(requested int) int {
	if requested > 0 {
		return requested
	}
	return CurrentVersion
}

func validateIntentMode(result *ConfigResult, intent ConfigIntent) {
	if intent.Mode == "" {
		result.addError("mode", "mode is required for non-incremental config generation")
		return
	}
	if !intent.Mode.IsValid() {
		result.addError("mode", fmt.Sprintf(
			"unknown mode %q; valid modes: validator, full, seed, archive, rpc, indexer", intent.Mode))
	}
}

func validateIntentVersion(result *ConfigResult, intent ConfigIntent) {
	tv := resolveTargetVersion(intent.TargetVersion)
	if tv < 1 {
		result.addError("targetVersion", "target version must be >= 1")
	}
	if tv > CurrentVersion {
		result.addError("targetVersion", fmt.Sprintf(
			"target version %d exceeds maximum supported version %d",
			tv, CurrentVersion))
	}
}

func validateIntentOverrideKeys(result *ConfigResult, intent ConfigIntent, registry *Registry) {
	if len(intent.Overrides) == 0 {
		return
	}
	for key := range intent.Overrides {
		if registry.Field(key) == nil {
			result.addError("overrides."+key, fmt.Sprintf("unknown config field %q", key))
		}
	}
}

func validateIntentRequiredFields(result *ConfigResult, intent ConfigIntent, registry *Registry) {
	if intent.Incremental {
		return
	}

	tv := resolveTargetVersion(intent.TargetVersion)
	for _, field := range registry.Fields() {
		if field.SinceVersion <= 0 || field.SinceVersion > tv {
			continue
		}
		if len(field.RequiredForModes) == 0 {
			continue
		}
		if !slices.Contains(field.RequiredForModes, intent.Mode) {
			continue
		}

		// Field is required for this mode+version. Check if it's in overrides
		// or has a non-zero default.
		if _, ok := intent.Overrides[field.Key]; ok {
			continue
		}

		defaults := registry.DefaultsByMode(intent.Mode)
		if v, ok := defaults[field.Key]; ok && !isZeroValue(v) {
			continue
		}

		result.addError(field.Key, fmt.Sprintf(
			"field %q is required for mode %q in config version %d",
			field.Key, intent.Mode, field.SinceVersion))
	}
}

func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int64:
		return val == 0
	case uint:
		return val == 0
	case uint64:
		return val == 0
	case float64:
		return val == 0
	case bool:
		return !val
	default:
		return false
	}
}
