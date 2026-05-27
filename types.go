package seiconfig

import (
	"fmt"
	"time"
)

// NodeMode represents the operating mode of a Sei node.
type NodeMode string

const (
	ModeValidator NodeMode = "validator"
	ModeFull      NodeMode = "full"
	ModeSeed      NodeMode = "seed"
	ModeArchive   NodeMode = "archive"
)

// BackendPebbleDB is the upstream-accepted name for the pebbledb backend.
const BackendPebbleDB = "pebbledb"

var validModes = map[NodeMode]bool{
	ModeValidator: true,
	ModeFull:      true,
	ModeSeed:      true,
	ModeArchive:   true,
}

func (m NodeMode) IsValid() bool {
	return validModes[m]
}

func (m NodeMode) IsFullnodeType() bool {
	switch m {
	case ModeFull, ModeArchive:
		return true
	default:
		return false
	}
}

func (m NodeMode) String() string {
	return string(m)
}

// Duration wraps time.Duration for human-readable TOML serialization.
// Values are encoded as Go duration strings (e.g. "10s", "100ms", "168h").
type Duration struct {
	time.Duration
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	d.Duration = dur
	return nil
}

func Dur(d time.Duration) Duration {
	return Duration{Duration: d}
}

// WriteMode controls how EVM data writes are routed between backends.
type WriteMode string

const (
	// v2 write modes — FlatKV migration lifecycle (sei-chain ≥ v6.5).
	WriteModeMemiavlOnly        WriteMode = "memiavl_only"
	WriteModeMigrateEVM         WriteMode = "migrate_evm"
	WriteModeEVMMigrated        WriteMode = "evm_migrated"
	WriteModeMigrateAllButBank  WriteMode = "migrate_all_but_bank"
	WriteModeAllMigratedButBank WriteMode = "all_migrated_but_bank"
	WriteModeMigrateBank        WriteMode = "migrate_bank"
	WriteModeFlatKVOnly         WriteMode = "flatkv_only"
	WriteModeTestOnlyDualWrite  WriteMode = "test_only_dual_write"

	// Deprecated: v1 write modes, accepted only during v1→v2 migration.
	WriteModeCosmosOnly WriteMode = "cosmos_only"
	WriteModeDualWrite  WriteMode = "dual_write"
	WriteModeSplitWrite WriteMode = "split_write"
)

func (m WriteMode) IsValid() bool {
	switch m {
	case WriteModeMemiavlOnly, WriteModeMigrateEVM, WriteModeEVMMigrated,
		WriteModeMigrateAllButBank, WriteModeAllMigratedButBank,
		WriteModeMigrateBank, WriteModeFlatKVOnly, WriteModeTestOnlyDualWrite,
		// Deprecated v1 modes remain valid: the stable released seid (v6.5.1)
		// still accepts them and rejects the v2 names. The v1→v2 migration
		// renames them; validation must not reject configs targeting v6.5.1.
		WriteModeCosmosOnly, WriteModeDualWrite, WriteModeSplitWrite:
		return true
	default:
		return false
	}
}

// ReadMode controls how EVM data reads are routed.
type ReadMode string

const (
	ReadModeCosmosOnly ReadMode = "cosmos_only"
	ReadModeEVMFirst   ReadMode = "evm_first"
	ReadModeSplitRead  ReadMode = "split_read"
)

func (m ReadMode) IsValid() bool {
	switch m {
	case ReadModeCosmosOnly, ReadModeEVMFirst, ReadModeSplitRead:
		return true
	default:
		return false
	}
}
