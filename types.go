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
	ModeRPC       NodeMode = "rpc"
	ModeIndexer   NodeMode = "indexer"
)

var validModes = map[NodeMode]bool{
	ModeValidator: true,
	ModeFull:      true,
	ModeSeed:      true,
	ModeArchive:   true,
	ModeRPC:       true,
	ModeIndexer:   true,
}

func (m NodeMode) IsValid() bool {
	return validModes[m]
}

func (m NodeMode) IsFullnodeType() bool {
	switch m {
	case ModeFull, ModeArchive, ModeRPC, ModeIndexer:
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
	WriteModeCosmosOnly WriteMode = "cosmos_only"
	WriteModeDualWrite  WriteMode = "dual_write"
	WriteModeSplitWrite WriteMode = "split_write"
	WriteModeEVMOnly    WriteMode = "evm_only"
)

func (m WriteMode) IsValid() bool {
	switch m {
	case WriteModeCosmosOnly, WriteModeDualWrite, WriteModeSplitWrite, WriteModeEVMOnly:
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
