// Package seiconfig provides unified configuration types, mode-aware defaults,
// validation, and serialization for Sei blockchain nodes.
//
// It serves as the single source of truth for configuration across all Sei
// components: seid, seictl (CLI and sidecar), and the Kubernetes controller.
package seiconfig

import "runtime"

// CurrentVersion is the config schema version produced by this library.
const CurrentVersion = 1

// DefaultSnapshotInterval is the default Tendermint state-sync snapshot
// creation interval (in blocks) used when snapshot generation is enabled.
const DefaultSnapshotInterval = 2000

// Pruning strategy constants.
const (
	PruningDefault    = "default"
	PruningNothing    = "nothing"
	PruningEverything = "everything"
	PruningCustom     = "custom"
)

// SeiConfig is the unified configuration for a Sei node, encompassing all
// settings previously split across config.toml (Tendermint) and app.toml
// (Cosmos SDK + Sei extensions).
type SeiConfig struct {
	Version int      `toml:"version"`
	Mode    NodeMode `toml:"mode"`

	Chain           ChainConfig           `toml:"chain"`
	Network         NetworkConfig         `toml:"network"`
	Consensus       ConsensusConfig       `toml:"consensus"`
	Mempool         MempoolConfig         `toml:"mempool"`
	StateSync       StateSyncConfig       `toml:"state_sync"`
	Storage         StorageConfig         `toml:"storage"`
	TxIndex         TxIndexConfig         `toml:"tx_index"`
	EVM             EVMConfig             `toml:"evm"`
	API             APIConfig             `toml:"api"`
	Metrics         MetricsConfig         `toml:"metrics"`
	Logging         LogConfig             `toml:"logging"`
	WASM            WASMConfig            `toml:"wasm"`
	GigaExecutor    GigaExecutorConfig    `toml:"giga_executor"`
	LightInvariance LightInvarianceConfig `toml:"light_invariance"`
	PrivValidator   PrivValidatorConfig   `toml:"priv_validator"`
	SelfRemediation SelfRemediationConfig `toml:"self_remediation"`
	Genesis         GenesisConfig         `toml:"genesis"`
}

// ---------------------------------------------------------------------------
// Chain — node identity and core chain parameters
// ---------------------------------------------------------------------------

type ChainConfig struct {
	ChainID     string `toml:"chain_id"`
	Moniker     string `toml:"moniker"`
	ProxyApp    string `toml:"proxy_app"`
	ABCI        string `toml:"abci"`
	FilterPeers bool   `toml:"filter_peers"`

	MinGasPrices    string   `toml:"min_gas_prices"`
	HaltHeight      uint64   `toml:"halt_height"`
	HaltTime        uint64   `toml:"halt_time"`
	MinRetainBlocks uint64   `toml:"min_retain_blocks"`
	InterBlockCache bool     `toml:"inter_block_cache"`
	IndexEvents     []string `toml:"index_events"`

	ConcurrencyWorkers int  `toml:"concurrency_workers"`
	OccEnabled         bool `toml:"occ_enabled"`
}

// ---------------------------------------------------------------------------
// Network — RPC and P2P
// ---------------------------------------------------------------------------

type NetworkConfig struct {
	RPC RPCConfig `toml:"rpc"`
	P2P P2PConfig `toml:"p2p"`
}

type RPCConfig struct {
	ListenAddress string   `toml:"listen_address"`
	CORSOrigins   []string `toml:"cors_allowed_origins"`
	CORSMethods   []string `toml:"cors_allowed_methods"`
	CORSHeaders   []string `toml:"cors_allowed_headers"`

	Unsafe             bool `toml:"unsafe"`
	MaxOpenConnections int  `toml:"max_open_connections"`

	MaxSubscriptionClients       int  `toml:"max_subscription_clients"`
	MaxSubscriptionsPerClient    int  `toml:"max_subscriptions_per_client"`
	ExperimentalDisableWebsocket bool `toml:"experimental_disable_websocket"`

	EventLogWindowSize Duration `toml:"event_log_window_size"`
	EventLogMaxItems   int      `toml:"event_log_max_items"`

	TimeoutBroadcastTxCommit Duration `toml:"timeout_broadcast_tx_commit"`
	MaxBodyBytes             int64    `toml:"max_body_bytes"`
	MaxHeaderBytes           int      `toml:"max_header_bytes"`

	TLSCertFile string `toml:"tls_cert_file"`
	TLSKeyFile  string `toml:"tls_key_file"`

	PprofListenAddress string   `toml:"pprof_listen_address"`
	LagThreshold       int64    `toml:"lag_threshold"`
	TimeoutRead        Duration `toml:"timeout_read"`
}

type P2PConfig struct {
	ListenAddress   string `toml:"listen_address"`
	ExternalAddress string `toml:"external_address"`
	BootstrapPeers  string `toml:"bootstrap_peers"`
	PersistentPeers string `toml:"persistent_peers"`
	BlockSyncPeers  string `toml:"blocksync_peers"`

	UPNP                          bool   `toml:"upnp"`
	MaxConnections                uint16 `toml:"max_connections"`
	MaxIncomingConnectionAttempts uint   `toml:"max_incoming_connection_attempts"`
	PexReactor                    bool   `toml:"pex"`
	PrivatePeerIDs                string `toml:"private_peer_ids"`
	AllowDuplicateIP              bool   `toml:"allow_duplicate_ip"`
	UnconditionalPeerIDs          string `toml:"unconditional_peer_ids"`

	FlushThrottleTimeout    Duration `toml:"flush_throttle_timeout"`
	MaxPacketMsgPayloadSize int      `toml:"max_packet_msg_payload_size"`
	SendRate                int64    `toml:"send_rate"`
	RecvRate                int64    `toml:"recv_rate"`
	HandshakeTimeout        Duration `toml:"handshake_timeout"`
	DialTimeout             Duration `toml:"dial_timeout"`
	DialInterval            Duration `toml:"dial_interval"`

	QueueType string `toml:"queue_type"`
}

// ---------------------------------------------------------------------------
// Consensus
// ---------------------------------------------------------------------------

type ConsensusConfig struct {
	WALPath                   string   `toml:"wal_path"`
	CreateEmptyBlocks         bool     `toml:"create_empty_blocks"`
	CreateEmptyBlocksInterval Duration `toml:"create_empty_blocks_interval"`
	GossipTransactionKeyOnly  bool     `toml:"gossip_transaction_key_only"`

	PeerGossipSleepDuration     Duration `toml:"peer_gossip_sleep_duration"`
	PeerQueryMaj23SleepDuration Duration `toml:"peer_query_maj23_sleep_duration"`
	DoubleSignCheckHeight       int64    `toml:"double_sign_check_height"`

	UnsafeProposeTimeoutOverride      Duration `toml:"unsafe_propose_timeout_override"`
	UnsafeProposeTimeoutDeltaOverride Duration `toml:"unsafe_propose_timeout_delta_override"`
	UnsafeVoteTimeoutOverride         Duration `toml:"unsafe_vote_timeout_override"`
	UnsafeVoteTimeoutDeltaOverride    Duration `toml:"unsafe_vote_timeout_delta_override"`
	UnsafeCommitTimeoutOverride       Duration `toml:"unsafe_commit_timeout_override"`
	UnsafeBypassCommitTimeoutOverride *bool    `toml:"unsafe_bypass_commit_timeout_override"`
}

// ---------------------------------------------------------------------------
// Mempool
// ---------------------------------------------------------------------------

type MempoolConfig struct {
	Broadcast             bool     `toml:"broadcast"`
	Size                  int      `toml:"size"`
	MaxTxsBytes           int64    `toml:"max_txs_bytes"`
	CacheSize             int      `toml:"cache_size"`
	DuplicateTxsCacheSize int      `toml:"duplicate_txs_cache_size"`
	KeepInvalidTxsInCache bool     `toml:"keep_invalid_txs_in_cache"`
	MaxTxBytes            int      `toml:"max_tx_bytes"`
	MaxBatchBytes         int      `toml:"max_batch_bytes"`
	TTLDuration           Duration `toml:"ttl_duration"`
	TTLNumBlocks          int64    `toml:"ttl_num_blocks"`
	TxNotifyThreshold     uint64   `toml:"tx_notify_threshold"`

	CheckTxErrorBlacklistEnabled bool `toml:"check_tx_error_blacklist_enabled"`
	CheckTxErrorThreshold        int  `toml:"check_tx_error_threshold"`

	PendingSize         int      `toml:"pending_size"`
	MaxPendingTxsBytes  int64    `toml:"max_pending_txs_bytes"`
	PendingTTLDuration  Duration `toml:"pending_ttl_duration"`
	PendingTTLNumBlocks int64    `toml:"pending_ttl_num_blocks"`

	RemoveExpiredTxsFromQueue bool    `toml:"remove_expired_txs_from_queue"`
	DropPriorityThreshold     float64 `toml:"drop_priority_threshold"`
	DropUtilisationThreshold  float64 `toml:"drop_utilisation_threshold"`
	DropPriorityReservoirSize int     `toml:"drop_priority_reservoir_size"`
}

// ---------------------------------------------------------------------------
// StateSync
// ---------------------------------------------------------------------------

type StateSyncConfig struct {
	Enable     bool     `toml:"enable"`
	UseP2P     bool     `toml:"use_p2p"`
	RPCServers []string `toml:"rpc_servers"`

	TrustHeight int64    `toml:"trust_height"`
	TrustHash   string   `toml:"trust_hash"`
	TrustPeriod Duration `toml:"trust_period"`

	BackfillBlocks   int64    `toml:"backfill_blocks"`
	BackfillDuration Duration `toml:"backfill_duration"`

	DiscoveryTime       Duration `toml:"discovery_time"`
	TempDir             string   `toml:"temp_dir"`
	ChunkRequestTimeout Duration `toml:"chunk_request_timeout"`
	Fetchers            int32    `toml:"fetchers"`

	VerifyLightBlockTimeout   Duration `toml:"verify_light_block_timeout"`
	BlacklistTTL              Duration `toml:"blacklist_ttl"`
	UseLocalSnapshot          bool     `toml:"use_local_snapshot"`
	LightBlockResponseTimeout Duration `toml:"light_block_response_timeout"`
}

// ---------------------------------------------------------------------------
// Storage — DB, pruning, snapshots, SeiDB (state-commit + state-store)
// ---------------------------------------------------------------------------

type StorageConfig struct {
	DBBackend string `toml:"db_backend"`
	DBPath    string `toml:"db_path"`

	PruningStrategy   string `toml:"pruning"`
	PruningKeepRecent string `toml:"pruning_keep_recent"`
	PruningKeepEvery  string `toml:"pruning_keep_every"`
	PruningInterval   string `toml:"pruning_interval"`

	SnapshotInterval   uint64 `toml:"snapshot_interval"`
	SnapshotKeepRecent uint32 `toml:"snapshot_keep_recent"`
	SnapshotDirectory  string `toml:"snapshot_directory"`

	CompactionInterval  uint64 `toml:"compaction_interval"`
	IAVLDisableFastNode bool   `toml:"iavl_disable_fast_node"`

	StateCommit StateCommitConfig `toml:"state_commit"`
	StateStore  StateStoreConfig  `toml:"state_store"`
}

type StateCommitConfig struct {
	Enable            bool      `toml:"enable"`
	Directory         string    `toml:"directory"`
	AsyncCommitBuffer int       `toml:"async_commit_buffer"`
	WriteMode         WriteMode `toml:"write_mode"`
	ReadMode          ReadMode  `toml:"read_mode"`

	MemIAVL MemIAVLConfig `toml:"memiavl"`
}

type MemIAVLConfig struct {
	SnapshotKeepRecent        uint32  `toml:"snapshot_keep_recent"`
	SnapshotInterval          uint32  `toml:"snapshot_interval"`
	SnapshotMinTimeInterval   uint32  `toml:"snapshot_min_time_interval"`
	SnapshotWriterLimit       int     `toml:"snapshot_writer_limit"`
	SnapshotPrefetchThreshold float64 `toml:"snapshot_prefetch_threshold"`
}

type StateStoreConfig struct {
	Enable               bool      `toml:"enable"`
	DBDirectory          string    `toml:"db_directory"`
	Backend              string    `toml:"backend"`
	AsyncWriteBuffer     int       `toml:"async_write_buffer"`
	KeepRecent           int       `toml:"keep_recent"`
	PruneIntervalSeconds int       `toml:"prune_interval_seconds"`
	ImportNumWorkers     int       `toml:"import_num_workers"`
	KeepLastVersion      bool      `toml:"keep_last_version"`
	UseDefaultComparer   bool      `toml:"use_default_comparer"`
	WriteMode            WriteMode `toml:"write_mode"`
	ReadMode             ReadMode  `toml:"read_mode"`
	EVMDBDirectory       string    `toml:"evm_db_directory"`
}

// ---------------------------------------------------------------------------
// TxIndex
// ---------------------------------------------------------------------------

type TxIndexConfig struct {
	Indexer  []string `toml:"indexer"`
	PsqlConn string   `toml:"psql_conn"`
}

// ---------------------------------------------------------------------------
// EVM — RPC server, tracing, query, replay, block test
// ---------------------------------------------------------------------------

type EVMConfig struct {
	HTTPEnabled bool `toml:"http_enabled"`
	HTTPPort    int  `toml:"http_port"`
	WSEnabled   bool `toml:"ws_enabled"`
	WSPort      int  `toml:"ws_port"`

	ReadTimeout       Duration `toml:"read_timeout"`
	ReadHeaderTimeout Duration `toml:"read_header_timeout"`
	WriteTimeout      Duration `toml:"write_timeout"`
	IdleTimeout       Duration `toml:"idle_timeout"`

	SimulationGasLimit   uint64   `toml:"simulation_gas_limit"`
	SimulationEVMTimeout Duration `toml:"simulation_evm_timeout"`

	CORSOrigins string `toml:"cors_origins"`
	WSOrigins   string `toml:"ws_origins"`

	FilterTimeout  Duration `toml:"filter_timeout"`
	CheckTxTimeout Duration `toml:"checktx_timeout"`

	MaxTxPoolTxs uint64 `toml:"max_tx_pool_txs"`
	Slow         bool   `toml:"slow"`

	DenyList []string `toml:"deny_list"`

	MaxLogNoBlock           int64  `toml:"max_log_no_block"`
	MaxBlocksForLog         int64  `toml:"max_blocks_for_log"`
	MaxSubscriptionsNewHead uint64 `toml:"max_subscriptions_new_head"`
	EnableTestAPI           bool   `toml:"enable_test_api"`

	MaxConcurrentTraceCalls      uint64   `toml:"max_concurrent_trace_calls"`
	MaxConcurrentSimulationCalls int      `toml:"max_concurrent_simulation_calls"`
	MaxTraceLookbackBlocks       int64    `toml:"max_trace_lookback_blocks"`
	TraceTimeout                 Duration `toml:"trace_timeout"`

	RPCStatsInterval Duration `toml:"rpc_stats_interval"`
	WorkerPoolSize   int      `toml:"worker_pool_size"`
	WorkerQueueSize  int      `toml:"worker_queue_size"`

	Query     EVMQueryConfig     `toml:"query"`
	Replay    EVMReplayConfig    `toml:"replay"`
	BlockTest EVMBlockTestConfig `toml:"block_test"`
}

type EVMQueryConfig struct {
	GasLimit uint64 `toml:"gas_limit"`
}

type EVMReplayConfig struct {
	Enabled             bool   `toml:"enabled"`
	EthRPC              string `toml:"eth_rpc"`
	EthDataDir          string `toml:"eth_data_dir"`
	ContractStateChecks bool   `toml:"contract_state_checks"`
}

type EVMBlockTestConfig struct {
	Enabled      bool   `toml:"enabled"`
	TestDataPath string `toml:"test_data_path"`
}

// ---------------------------------------------------------------------------
// API — REST, gRPC, gRPC-Web
// ---------------------------------------------------------------------------

type APIConfig struct {
	REST    RESTAPIConfig `toml:"rest"`
	GRPC    GRPCConfig    `toml:"grpc"`
	GRPCWeb GRPCWebConfig `toml:"grpc_web"`
}

type RESTAPIConfig struct {
	Enable             bool   `toml:"enable"`
	Swagger            bool   `toml:"swagger"`
	Address            string `toml:"address"`
	MaxOpenConnections uint   `toml:"max_open_connections"`
	RPCReadTimeout     uint   `toml:"rpc_read_timeout"`
	RPCWriteTimeout    uint   `toml:"rpc_write_timeout"`
	RPCMaxBodyBytes    uint   `toml:"rpc_max_body_bytes"`
	EnableUnsafeCORS   bool   `toml:"enable_unsafe_cors"`
}

type GRPCConfig struct {
	Enable  bool   `toml:"enable"`
	Address string `toml:"address"`
}

type GRPCWebConfig struct {
	Enable           bool   `toml:"enable"`
	Address          string `toml:"address"`
	EnableUnsafeCORS bool   `toml:"enable_unsafe_cors"`
}

// ---------------------------------------------------------------------------
// Metrics — merges Tendermint instrumentation + Cosmos telemetry
// ---------------------------------------------------------------------------

type MetricsConfig struct {
	Enabled              bool   `toml:"enabled"`
	PrometheusListenAddr string `toml:"prometheus_listen_addr"`
	MaxOpenConnections   int    `toml:"max_open_connections"`
	Namespace            string `toml:"namespace"`

	ServiceName             string     `toml:"service_name"`
	EnableHostname          bool       `toml:"enable_hostname"`
	EnableHostnameLabel     bool       `toml:"enable_hostname_label"`
	EnableServiceLabel      bool       `toml:"enable_service_label"`
	PrometheusRetentionTime int64      `toml:"prometheus_retention_time"`
	GlobalLabels            [][]string `toml:"global_labels"`
}

// ---------------------------------------------------------------------------
// Logging
// ---------------------------------------------------------------------------

type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

// ---------------------------------------------------------------------------
// WASM
// ---------------------------------------------------------------------------

type WASMConfig struct {
	QueryGasLimit uint64 `toml:"query_gas_limit"`
	LruSize       uint64 `toml:"lru_size"`
}

// ---------------------------------------------------------------------------
// GigaExecutor
// ---------------------------------------------------------------------------

type GigaExecutorConfig struct {
	Enabled    bool `toml:"enabled"`
	OccEnabled bool `toml:"occ_enabled"`
}

// ---------------------------------------------------------------------------
// LightInvariance
// ---------------------------------------------------------------------------

type LightInvarianceConfig struct {
	SupplyEnabled bool `toml:"supply_enabled"`
}

// ---------------------------------------------------------------------------
// PrivValidator
// ---------------------------------------------------------------------------

type PrivValidatorConfig struct {
	KeyFile           string `toml:"key_file"`
	StateFile         string `toml:"state_file"`
	ListenAddr        string `toml:"listen_addr"`
	ClientCertificate string `toml:"client_certificate_file"`
	ClientKey         string `toml:"client_key_file"`
	RootCA            string `toml:"root_ca_file"`
}

// ---------------------------------------------------------------------------
// SelfRemediation
// ---------------------------------------------------------------------------

type SelfRemediationConfig struct {
	P2PNoPeersRestartWindowSeconds       uint64 `toml:"p2p_no_peers_restart_window_seconds"`
	StatesyncNoPeersRestartWindowSeconds uint64 `toml:"statesync_no_peers_restart_window_seconds"`
	BlocksBehindThreshold                uint64 `toml:"blocks_behind_threshold"`
	BlocksBehindCheckIntervalSeconds     uint64 `toml:"blocks_behind_check_interval_seconds"`
	RestartCooldownSeconds               uint64 `toml:"restart_cooldown_seconds"`
}

// ---------------------------------------------------------------------------
// Genesis (stream import)
// ---------------------------------------------------------------------------

type GenesisConfig struct {
	StreamImport      bool   `toml:"stream_import"`
	GenesisStreamFile string `toml:"genesis_stream_file"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func defaultConcurrencyWorkers() int {
	workers := runtime.NumCPU() * 2
	return max(10, min(workers, 128))
}

func defaultEVMWorkerPoolSize() int {
	return min(64, runtime.NumCPU()*2)
}
