package seiconfig

import (
	"encoding/hex"
	"fmt"
)

// Severity classifies a validation finding.
type Severity int

const (
	SeverityError   Severity = iota // prevents seid from starting
	SeverityWarning                 // logged but does not block startup
	SeverityInfo                    // informational
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// Diagnostic describes a single validation finding.
type Diagnostic struct {
	Severity Severity
	Field    string
	Message  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s", d.Severity, d.Field, d.Message)
}

// ValidationResult holds all findings from a Validate call.
type ValidationResult struct {
	Diagnostics []Diagnostic
}

func (r *ValidationResult) addError(field, msg string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{SeverityError, field, msg})
}

func (r *ValidationResult) addWarning(field, msg string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{SeverityWarning, field, msg})
}

func (r *ValidationResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r *ValidationResult) Errors() []Diagnostic {
	var errs []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			errs = append(errs, d)
		}
	}
	return errs
}

// ValidateOpts controls optional validation behavior.
type ValidateOpts struct {
	// MaxVersion overrides CurrentVersion for the version ceiling check.
	// When zero, CurrentVersion is used.
	MaxVersion int
}

// Validate checks a SeiConfig for correctness and returns all findings.
func Validate(cfg *SeiConfig) *ValidationResult {
	return ValidateWithOpts(cfg, ValidateOpts{})
}

// ValidateWithOpts checks a SeiConfig with configurable options.
func ValidateWithOpts(cfg *SeiConfig, opts ValidateOpts) *ValidationResult {
	r := &ValidationResult{}
	maxVer := CurrentVersion
	if opts.MaxVersion > 0 {
		maxVer = opts.MaxVersion
	}
	validateMode(r, cfg)
	validateVersion(r, cfg, maxVer)
	validateChain(r, cfg)
	validateNetwork(r, cfg)
	validateConsensus(r, cfg)
	validateMempool(r, cfg)
	validateStateSync(r, cfg)
	validateStorage(r, cfg)
	validateEVM(r, cfg)
	validateMetrics(r, cfg)
	validateLogging(r, cfg)
	validateSelfRemediation(r, cfg)
	validateCrossField(r, cfg)
	return r
}

func validateMode(r *ValidationResult, cfg *SeiConfig) {
	if cfg.Mode == "" {
		r.addError("mode", "mode must be set")
		return
	}
	if !cfg.Mode.IsValid() {
		r.addError("mode", fmt.Sprintf(
			"unknown mode %q; valid modes: validator, full, seed, archive", cfg.Mode))
	}
}

func validateVersion(r *ValidationResult, cfg *SeiConfig, maxVersion int) {
	if cfg.Version < 1 {
		r.addError("version", "config version must be >= 1")
	}
	if cfg.Version > maxVersion {
		r.addError("version", fmt.Sprintf(
			"config version %d is newer than supported version %d; "+
				"upgrade or run `seictl config migrate`",
			cfg.Version, maxVersion))
	}
}

func validateChain(r *ValidationResult, cfg *SeiConfig) {
	if cfg.Chain.MinGasPrices == "" {
		r.addError("chain.min_gas_prices", "min_gas_prices must be set")
	}
	if cfg.Chain.ConcurrencyWorkers < -1 {
		r.addError("chain.concurrency_workers", "concurrency_workers must be >= -1")
	}
}

func validateNetwork(r *ValidationResult, cfg *SeiConfig) {
	rpc := &cfg.Network.RPC
	if rpc.MaxOpenConnections < 0 {
		r.addError("network.rpc.max_open_connections", "must be >= 0")
	}
	if rpc.MaxSubscriptionClients < 0 {
		r.addError("network.rpc.max_subscription_clients", "must be >= 0")
	}
	if rpc.MaxSubscriptionsPerClient < 0 {
		r.addError("network.rpc.max_subscriptions_per_client", "must be >= 0")
	}
	if rpc.TimeoutBroadcastTxCommit.Duration < 0 {
		r.addError("network.rpc.timeout_broadcast_tx_commit", "must be >= 0")
	}
	if rpc.MaxBodyBytes < 0 {
		r.addError("network.rpc.max_body_bytes", "must be >= 0")
	}
	if rpc.MaxHeaderBytes < 0 {
		r.addError("network.rpc.max_header_bytes", "must be >= 0")
	}

	p2p := &cfg.Network.P2P
	if p2p.FlushThrottleTimeout.Duration < 0 {
		r.addError("network.p2p.flush_throttle_timeout", "must be >= 0")
	}
	if p2p.MaxPacketMsgPayloadSize < 0 {
		r.addError("network.p2p.max_packet_msg_payload_size", "must be >= 0")
	}
	if p2p.SendRate < 0 {
		r.addError("network.p2p.send_rate", "must be >= 0")
	}
	if p2p.RecvRate < 0 {
		r.addError("network.p2p.recv_rate", "must be >= 0")
	}
}

func validateConsensus(r *ValidationResult, cfg *SeiConfig) {
	c := &cfg.Consensus
	if c.UnsafeProposeTimeoutOverride.Duration < 0 {
		r.addError("consensus.unsafe_propose_timeout_override", "must be >= 0")
	}
	if c.UnsafeCommitTimeoutOverride.Duration < 0 {
		r.addError("consensus.unsafe_commit_timeout_override", "must be >= 0")
	}
	if c.CreateEmptyBlocksInterval.Duration < 0 {
		r.addError("consensus.create_empty_blocks_interval", "must be >= 0")
	}
	if c.PeerGossipSleepDuration.Duration < 0 {
		r.addError("consensus.peer_gossip_sleep_duration", "must be >= 0")
	}
	if c.DoubleSignCheckHeight < 0 {
		r.addError("consensus.double_sign_check_height", "must be >= 0")
	}
}

func validateMempool(r *ValidationResult, cfg *SeiConfig) {
	m := &cfg.Mempool
	if m.Size < 0 {
		r.addError("mempool.size", "must be >= 0")
	}
	if m.MaxTxsBytes < 0 {
		r.addError("mempool.max_txs_bytes", "must be >= 0")
	}
	if m.CacheSize < 0 {
		r.addError("mempool.cache_size", "must be >= 0")
	}
	if m.MaxTxBytes < 0 {
		r.addError("mempool.max_tx_bytes", "must be >= 0")
	}
	if m.TTLDuration.Duration < 0 {
		r.addError("mempool.ttl_duration", "must be >= 0")
	}
	if m.TTLNumBlocks < 0 {
		r.addError("mempool.ttl_num_blocks", "must be >= 0")
	}
	if m.DropUtilisationThreshold < 0 || m.DropUtilisationThreshold > 1.0 {
		r.addError("mempool.drop_utilisation_threshold", "must be between 0.0 and 1.0")
	}
}

func validateStateSync(r *ValidationResult, cfg *SeiConfig) {
	ss := &cfg.StateSync
	if !ss.Enable {
		return
	}
	if !ss.UseP2P && len(ss.RPCServers) < 2 {
		r.addError("state_sync.rpc_servers", "at least two RPC servers are required when not using P2P")
	}
	if ss.TrustPeriod.Duration <= 0 {
		r.addError("state_sync.trust_period", "must be > 0 when state sync is enabled")
	}
	if ss.TrustHeight <= 0 {
		r.addError("state_sync.trust_height", "must be > 0 when state sync is enabled")
	}
	if ss.TrustHash != "" {
		if _, err := hex.DecodeString(ss.TrustHash); err != nil {
			r.addError("state_sync.trust_hash", fmt.Sprintf("invalid hex: %v", err))
		}
	} else {
		r.addError("state_sync.trust_hash", "must be set when state sync is enabled")
	}
	if ss.Fetchers <= 0 {
		r.addError("state_sync.fetchers", "must be > 0")
	}
	if ss.BackfillBlocks < 0 {
		r.addError("state_sync.backfill_blocks", "must be >= 0")
	}
}

func validateStorage(r *ValidationResult, cfg *SeiConfig) {
	s := &cfg.Storage
	switch s.PruningStrategy {
	case PruningDefault, PruningNothing, PruningEverything, PruningCustom:
	default:
		r.addError("storage.pruning", fmt.Sprintf(
			"unknown pruning strategy %q; valid: %s, %s, %s, %s",
			s.PruningStrategy, PruningDefault, PruningNothing, PruningEverything, PruningCustom))
	}

	sc := &s.StateCommit
	if sc.WriteMode != "" && !sc.WriteMode.IsValid() {
		r.addError("storage.state_commit.write_mode", fmt.Sprintf("invalid write_mode: %q", sc.WriteMode))
	}
	if sc.ReadMode != "" && !sc.ReadMode.IsValid() {
		r.addError("storage.state_commit.read_mode", fmt.Sprintf("invalid read_mode: %q", sc.ReadMode))
	}

	ss := &s.StateStore
	if ss.WriteMode != "" && !ss.WriteMode.IsValid() {
		r.addError("storage.state_store.write_mode", fmt.Sprintf("invalid write_mode: %q", ss.WriteMode))
	}
	if ss.ReadMode != "" && !ss.ReadMode.IsValid() {
		r.addError("storage.state_store.read_mode", fmt.Sprintf("invalid read_mode: %q", ss.ReadMode))
	}
	if ss.Backend != "" && ss.Backend != BackendPebbleDB && ss.Backend != "rocksdb" {
		r.addWarning("storage.state_store.backend", fmt.Sprintf(
			"unusual backend %q; expected pebbledb or rocksdb", ss.Backend))
	}
}

func validateEVM(r *ValidationResult, cfg *SeiConfig) {
	e := &cfg.EVM
	if e.HTTPEnabled && e.HTTPPort <= 0 {
		r.addError("evm.http_port", "must be > 0 when HTTP is enabled")
	}
	if e.WSEnabled && e.WSPort <= 0 {
		r.addError("evm.ws_port", "must be > 0 when WS is enabled")
	}
	if cfg.Mode == ModeValidator && (e.HTTPEnabled || e.WSEnabled) {
		r.addWarning("evm", "EVM RPC is enabled on a validator node; this is unusual and may increase attack surface")
	}
}

func validateMetrics(r *ValidationResult, cfg *SeiConfig) {
	if cfg.Metrics.MaxOpenConnections < 0 {
		r.addError("metrics.max_open_connections", "must be >= 0")
	}
}

func validateLogging(r *ValidationResult, cfg *SeiConfig) {
	switch cfg.Logging.Format {
	case "plain", "text", "json":
	case "":
		r.addError("logging.format", "format must be set (plain, text, or json)")
	default:
		r.addError("logging.format", fmt.Sprintf("unknown format %q; valid: plain, text, json", cfg.Logging.Format))
	}
}

func validateSelfRemediation(_ *ValidationResult, _ *SeiConfig) {
	// All fields are uint64, no negative values possible.
}

func validateCrossField(r *ValidationResult, cfg *SeiConfig) {
	if cfg.Storage.PruningStrategy == PruningEverything && cfg.Storage.SnapshotInterval > 0 {
		r.addError("storage", "cannot enable snapshots with 'everything' pruning strategy")
	}
}
