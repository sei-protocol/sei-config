package seiconfig

// Legacy types match the existing config.toml (Tendermint) and app.toml
// (Cosmos SDK + Sei) TOML schemas exactly. These are used during Phase 2
// for reading/writing the two-file layout and will be removed when the
// unified sei.toml format ships in Phase 3.

// ---------------------------------------------------------------------------
// config.toml — Tendermint configuration
// ---------------------------------------------------------------------------

type legacyTendermintConfig struct {
	ProxyApp    string `toml:"proxy-app"`
	Moniker     string `toml:"moniker"`
	Mode        string `toml:"mode"`
	DBBackend   string `toml:"db-backend"`
	DBPath      string `toml:"db-dir"`
	LogLevel    string `toml:"log-level"`
	LogFormat   string `toml:"log-format"`
	Genesis     string `toml:"genesis-file"`
	NodeKey     string `toml:"node-key-file"`
	ABCI        string `toml:"abci"`
	FilterPeers bool   `toml:"filter-peers"`

	RPC             legacyRPC             `toml:"rpc"`
	P2P             legacyP2P             `toml:"p2p"`
	Mempool         legacyMempool         `toml:"mempool"`
	StateSync       legacyStateSync       `toml:"statesync"`
	Consensus       legacyConsensus       `toml:"consensus"`
	TxIndex         legacyTxIndex         `toml:"tx-index"`
	Instrumentation legacyInstrumentation `toml:"instrumentation"`
	PrivValidator   legacyPrivValidator   `toml:"priv-validator"`
	SelfRemediation legacySelfRemediation `toml:"self-remediation"`
}

type legacyRPC struct {
	ListenAddress                string   `toml:"laddr"`
	CORSAllowedOrigins           []string `toml:"cors-allowed-origins"`
	CORSAllowedMethods           []string `toml:"cors-allowed-methods"`
	CORSAllowedHeaders           []string `toml:"cors-allowed-headers"`
	Unsafe                       bool     `toml:"unsafe"`
	MaxOpenConnections           int      `toml:"max-open-connections"`
	MaxSubscriptionClients       int      `toml:"max-subscription-clients"`
	MaxSubscriptionsPerClient    int      `toml:"max-subscriptions-per-client"`
	ExperimentalDisableWebsocket bool     `toml:"experimental-disable-websocket"`
	EventLogWindowSize           Duration `toml:"event-log-window-size"`
	EventLogMaxItems             int      `toml:"event-log-max-items"`
	TimeoutBroadcastTxCommit     Duration `toml:"timeout-broadcast-tx-commit"`
	MaxBodyBytes                 int64    `toml:"max-body-bytes"`
	MaxHeaderBytes               int      `toml:"max-header-bytes"`
	TLSCertFile                  string   `toml:"tls-cert-file"`
	TLSKeyFile                   string   `toml:"tls-key-file"`
	PprofListenAddress           string   `toml:"pprof-laddr"`
	LagThreshold                 int64    `toml:"lag-threshold"`
	TimeoutRead                  Duration `toml:"timeout-read"`
}

type legacyP2P struct {
	ListenAddress                 string   `toml:"laddr"`
	ExternalAddress               string   `toml:"external-address"`
	BootstrapPeers                string   `toml:"bootstrap-peers"`
	PersistentPeers               string   `toml:"persistent-peers"`
	BlockSyncPeers                string   `toml:"blocksync-peers"`
	UPNP                          bool     `toml:"upnp"`
	MaxConnections                uint16   `toml:"max-connections"`
	MaxIncomingConnectionAttempts uint     `toml:"max-incoming-connection-attempts"`
	PexReactor                    bool     `toml:"pex"`
	PrivatePeerIDs                string   `toml:"private-peer-ids"`
	AllowDuplicateIP              bool     `toml:"allow-duplicate-ip"`
	UnconditionalPeerIDs          string   `toml:"unconditional-peer-ids"`
	FlushThrottleTimeout          Duration `toml:"flush-throttle-timeout"`
	MaxPacketMsgPayloadSize       int      `toml:"max-packet-msg-payload-size"`
	SendRate                      int64    `toml:"send-rate"`
	RecvRate                      int64    `toml:"recv-rate"`
	HandshakeTimeout              Duration `toml:"handshake-timeout"`
	DialTimeout                   Duration `toml:"dial-timeout"`
	DialInterval                  Duration `toml:"dial-interval"`
	QueueType                     string   `toml:"queue-type"`
}

type legacyMempool struct {
	Broadcast                    bool     `toml:"broadcast"`
	Size                         int      `toml:"size"`
	MaxTxsBytes                  int64    `toml:"max-txs-bytes"`
	CacheSize                    int      `toml:"cache-size"`
	DuplicateTxsCacheSize        int      `toml:"duplicate-txs-cache-size"`
	KeepInvalidTxsInCache        bool     `toml:"keep-invalid-txs-in-cache"`
	MaxTxBytes                   int      `toml:"max-tx-bytes"`
	MaxBatchBytes                int      `toml:"max-batch-bytes"`
	TTLDuration                  Duration `toml:"ttl-duration"`
	TTLNumBlocks                 int64    `toml:"ttl-num-blocks"`
	TxNotifyThreshold            uint64   `toml:"tx-notify-threshold"`
	CheckTxErrorBlacklistEnabled bool     `toml:"check-tx-error-blacklist-enabled"`
	CheckTxErrorThreshold        int      `toml:"check-tx-error-threshold"`
	PendingSize                  int      `toml:"pending-size"`
	MaxPendingTxsBytes           int64    `toml:"max-pending-txs-bytes"`
	PendingTTLDuration           Duration `toml:"pending-ttl-duration"`
	PendingTTLNumBlocks          int64    `toml:"pending-ttl-num-blocks"`
	RemoveExpiredTxsFromQueue    bool     `toml:"remove-expired-txs-from-queue"`
	DropPriorityThreshold        float64  `toml:"drop-priority-threshold"`
	DropUtilisationThreshold     float64  `toml:"drop-utilisation-threshold"`
	DropPriorityReservoirSize    int      `toml:"drop-priority-reservoir-size"`
}

type legacyStateSync struct {
	Enable                    bool     `toml:"enable"`
	UseP2P                    bool     `toml:"use-p2p"`
	RPCServers                []string `toml:"rpc-servers"`
	TrustHeight               int64    `toml:"trust-height"`
	TrustHash                 string   `toml:"trust-hash"`
	TrustPeriod               Duration `toml:"trust-period"`
	BackfillBlocks            int64    `toml:"backfill-blocks"`
	BackfillDuration          Duration `toml:"backfill-duration"`
	DiscoveryTime             Duration `toml:"discovery-time"`
	TempDir                   string   `toml:"temp-dir"`
	ChunkRequestTimeout       Duration `toml:"chunk-request-timeout"`
	Fetchers                  int32    `toml:"fetchers"`
	VerifyLightBlockTimeout   Duration `toml:"verify-light-block-timeout"`
	BlacklistTTL              Duration `toml:"blacklist-ttl"`
	UseLocalSnapshot          bool     `toml:"use-local-snapshot"`
	LightBlockResponseTimeout Duration `toml:"light-block-response-timeout"`
}

type legacyConsensus struct {
	WALPath                           string   `toml:"wal-file"`
	CreateEmptyBlocks                 bool     `toml:"create-empty-blocks"`
	CreateEmptyBlocksInterval         Duration `toml:"create-empty-blocks-interval"`
	GossipTransactionKeyOnly          bool     `toml:"gossip-tx-key-only"`
	PeerGossipSleepDuration           Duration `toml:"peer-gossip-sleep-duration"`
	PeerQueryMaj23SleepDuration       Duration `toml:"peer-query-maj23-sleep-duration"`
	DoubleSignCheckHeight             int64    `toml:"double-sign-check-height"`
	UnsafeProposeTimeoutOverride      Duration `toml:"unsafe-propose-timeout-override"`
	UnsafeProposeTimeoutDeltaOverride Duration `toml:"unsafe-propose-timeout-delta-override"`
	UnsafeVoteTimeoutOverride         Duration `toml:"unsafe-vote-timeout-override"`
	UnsafeVoteTimeoutDeltaOverride    Duration `toml:"unsafe-vote-timeout-delta-override"`
	UnsafeCommitTimeoutOverride       Duration `toml:"unsafe-commit-timeout-override"`
	UnsafeBypassCommitTimeoutOverride *bool    `toml:"unsafe-bypass-commit-timeout-override"`
}

type legacyTxIndex struct {
	Indexer  []string `toml:"indexer"`
	PsqlConn string   `toml:"psql-conn"`
}

type legacyInstrumentation struct {
	Prometheus           bool   `toml:"prometheus"`
	PrometheusListenAddr string `toml:"prometheus-listen-addr"`
	MaxOpenConnections   int    `toml:"max-open-connections"`
	Namespace            string `toml:"namespace"`
}

type legacyPrivValidator struct {
	KeyFile           string `toml:"key-file"`
	StateFile         string `toml:"state-file"`
	ListenAddr        string `toml:"laddr"`
	ClientCertificate string `toml:"client-certificate-file"`
	ClientKey         string `toml:"client-key-file"`
	RootCA            string `toml:"root-ca-file"`
}

type legacySelfRemediation struct {
	P2PNoPeersRestartWindowSeconds       uint64 `toml:"p2p-no-peers-available-window-seconds"`
	StatesyncNoPeersRestartWindowSeconds uint64 `toml:"statesync-no-peers-available-window-seconds"`
	BlocksBehindThreshold                uint64 `toml:"blocks-behind-threshold"`
	BlocksBehindCheckIntervalSeconds     uint64 `toml:"blocks-behind-check-interval"`
	RestartCooldownSeconds               uint64 `toml:"restart-cooldown-seconds"`
}

// ---------------------------------------------------------------------------
// app.toml — Cosmos SDK + Sei configuration
// ---------------------------------------------------------------------------

type legacyAppConfig struct {
	MinGasPrices        string   `toml:"minimum-gas-prices"`
	Pruning             string   `toml:"pruning"`
	PruningKeepRecent   string   `toml:"pruning-keep-recent"`
	PruningKeepEvery    string   `toml:"pruning-keep-every"`
	PruningInterval     string   `toml:"pruning-interval"`
	HaltHeight          uint64   `toml:"halt-height"`
	HaltTime            uint64   `toml:"halt-time"`
	MinRetainBlocks     uint64   `toml:"min-retain-blocks"`
	InterBlockCache     bool     `toml:"inter-block-cache"`
	IndexEvents         []string `toml:"index-events"`
	IAVLDisableFastNode bool     `toml:"iavl-disable-fastnode"`
	CompactionInterval  uint64   `toml:"compaction-interval"`
	ConcurrencyWorkers  int      `toml:"concurrency-workers"`
	OccEnabled          bool     `toml:"occ-enabled"`

	Telemetry       legacyTelemetry       `toml:"telemetry"`
	API             legacyAPI             `toml:"api"`
	GRPC            legacyGRPC            `toml:"grpc"`
	GRPCWeb         legacyGRPCWeb         `toml:"grpc-web"`
	StateSync       legacyAppStateSync    `toml:"state-sync"`
	StateCommit     legacyStateCommit     `toml:"state-commit"`
	StateStore      legacyStateStore      `toml:"state-store"`
	EVM             legacyEVM             `toml:"evm"`
	WASM            legacyWASM            `toml:"wasm"`
	GigaExecutor    legacyGigaExecutor    `toml:"giga_executor"`
	ETHReplay       legacyETHReplay       `toml:"eth_replay"`
	ETHBlockTest    legacyETHBlockTest    `toml:"eth_block_test"`
	EVMQuery        legacyEVMQuery        `toml:"evm_query"`
	LightInvariance legacyLightInvariance `toml:"light_invariance"`
	Genesis         legacyGenesis         `toml:"genesis"`
	SeiMeta         legacySeiMeta         `toml:"sei"`
}

// legacySeiMeta stores sei-config metadata (mode, version) in app.toml so
// that unified mode values (archive, rpc, indexer) survive legacy round-trips.
// Tendermint's config.toml only understands validator/full/seed.
type legacySeiMeta struct {
	Mode    string `toml:"mode"`
	Version int    `toml:"version"`
}

type legacyTelemetry struct {
	ServiceName             string     `toml:"service-name"`
	Enabled                 bool       `toml:"enabled"`
	EnableHostname          bool       `toml:"enable-hostname"`
	EnableHostnameLabel     bool       `toml:"enable-hostname-label"`
	EnableServiceLabel      bool       `toml:"enable-service-label"`
	PrometheusRetentionTime int64      `toml:"prometheus-retention-time"`
	GlobalLabels            [][]string `toml:"global-labels"`
}

type legacyAPI struct {
	Enable             bool   `toml:"enable"`
	Swagger            bool   `toml:"swagger"`
	Address            string `toml:"address"`
	MaxOpenConnections uint   `toml:"max-open-connections"`
	RPCReadTimeout     uint   `toml:"rpc-read-timeout"`
	RPCWriteTimeout    uint   `toml:"rpc-write-timeout"`
	RPCMaxBodyBytes    uint   `toml:"rpc-max-body-bytes"`
	EnableUnsafeCORS   bool   `toml:"enabled-unsafe-cors"`
}

type legacyGRPC struct {
	Enable  bool   `toml:"enable"`
	Address string `toml:"address"`
}

type legacyGRPCWeb struct {
	Enable           bool   `toml:"enable"`
	Address          string `toml:"address"`
	EnableUnsafeCORS bool   `toml:"enable-unsafe-cors"`
}

type legacyAppStateSync struct {
	SnapshotInterval   uint64 `toml:"snapshot-interval"`
	SnapshotKeepRecent uint32 `toml:"snapshot-keep-recent"`
	SnapshotDirectory  string `toml:"snapshot-directory"`
}

type legacyStateCommit struct {
	Enable            bool   `toml:"sc-enable"`
	Directory         string `toml:"sc-directory"`
	AsyncCommitBuffer int    `toml:"sc-async-commit-buffer"`
	WriteMode         string `toml:"sc-write-mode"`
	ReadMode          string `toml:"sc-read-mode"`

	KeepRecent                uint32  `toml:"sc-keep-recent"`
	SnapshotInterval          uint32  `toml:"sc-snapshot-interval"`
	SnapshotMinTimeInterval   uint32  `toml:"sc-snapshot-min-time-interval"`
	SnapshotWriterLimit       int     `toml:"sc-snapshot-writer-limit"`
	SnapshotPrefetchThreshold float64 `toml:"sc-snapshot-prefetch-threshold"`
}

type legacyStateStore struct {
	Enable               bool   `toml:"ss-enable"`
	DBDirectory          string `toml:"ss-db-directory"`
	Backend              string `toml:"ss-backend"`
	AsyncWriteBuffer     int    `toml:"ss-async-write-buffer"`
	KeepRecent           int    `toml:"ss-keep-recent"`
	PruneIntervalSeconds int    `toml:"ss-prune-interval"`
	ImportNumWorkers     int    `toml:"ss-import-num-workers"`
	KeepLastVersion      bool   `toml:"ss-keep-last-version"`
	UseDefaultComparer   bool   `toml:"ss-use-default-comparer"`
	WriteMode            string `toml:"ss-write-mode"`
	ReadMode             string `toml:"ss-read-mode"`
	EVMDBDirectory       string `toml:"ss-evm-db-directory"`
}

type legacyEVM struct {
	HTTPEnabled                  bool     `toml:"http_enabled"`
	HTTPPort                     int      `toml:"http_port"`
	WSEnabled                    bool     `toml:"ws_enabled"`
	WSPort                       int      `toml:"ws_port"`
	ReadTimeout                  Duration `toml:"read_timeout"`
	ReadHeaderTimeout            Duration `toml:"read_header_timeout"`
	WriteTimeout                 Duration `toml:"write_timeout"`
	IdleTimeout                  Duration `toml:"idle_timeout"`
	SimulationGasLimit           uint64   `toml:"simulation_gas_limit"`
	SimulationEVMTimeout         Duration `toml:"simulation_evm_timeout"`
	CORSOrigins                  string   `toml:"cors_origins"`
	WSOrigins                    string   `toml:"ws_origins"`
	FilterTimeout                Duration `toml:"filter_timeout"`
	CheckTxTimeout               Duration `toml:"checktx_timeout"`
	MaxTxPoolTxs                 uint64   `toml:"max_tx_pool_txs"`
	Slow                         bool     `toml:"slow"`
	DenyList                     []string `toml:"deny_list"`
	MaxLogNoBlock                int64    `toml:"max_log_no_block"`
	MaxBlocksForLog              int64    `toml:"max_blocks_for_log"`
	MaxSubscriptionsNewHead      uint64   `toml:"max_subscriptions_new_head"`
	EnableTestAPI                bool     `toml:"enable_test_api"`
	MaxConcurrentTraceCalls      uint64   `toml:"max_concurrent_trace_calls"`
	MaxConcurrentSimulationCalls int      `toml:"max_concurrent_simulation_calls"`
	MaxTraceLookbackBlocks       int64    `toml:"max_trace_lookback_blocks"`
	TraceTimeout                 Duration `toml:"trace_timeout"`
	RPCStatsInterval             Duration `toml:"rpc_stats_interval"`
	WorkerPoolSize               int      `toml:"worker_pool_size"`
	WorkerQueueSize              int      `toml:"worker_queue_size"`
}

type legacyWASM struct {
	QueryGasLimit uint64 `toml:"query_gas_limit"`
	LruSize       uint64 `toml:"lru_size"`
}

type legacyGigaExecutor struct {
	Enabled    bool `toml:"giga_enabled"`
	OccEnabled bool `toml:"occ_enabled"`
}

type legacyETHReplay struct {
	Enabled             bool   `toml:"eth_replay_enabled"`
	EthRPC              string `toml:"eth_rpc"`
	EthDataDir          string `toml:"eth_data_dir"`
	ContractStateChecks bool   `toml:"eth_replay_contract_state_checks"`
}

type legacyETHBlockTest struct {
	Enabled      bool   `toml:"eth_block_test_enabled"`
	TestDataPath string `toml:"eth_block_test_test_data_path"`
}

type legacyEVMQuery struct {
	GasLimit uint64 `toml:"evm_query_gas_limit"`
}

type legacyLightInvariance struct {
	SupplyEnabled bool `toml:"supply_enabled"`
}

type legacyGenesis struct {
	StreamImport      bool   `toml:"stream-import"`
	GenesisStreamFile string `toml:"genesis-stream-file"`
}

// ---------------------------------------------------------------------------
// Conversion: SeiConfig ↔ legacy types
// ---------------------------------------------------------------------------

func (cfg *SeiConfig) toLegacyTendermint() legacyTendermintConfig {
	// Tendermint treats "archive" as "full"
	tmMode := cfg.Mode.String()
	if tmMode == "archive" || tmMode == "rpc" || tmMode == "indexer" {
		tmMode = "full"
	}

	return legacyTendermintConfig{
		ProxyApp:    cfg.Chain.ProxyApp,
		Moniker:     cfg.Chain.Moniker,
		Mode:        tmMode,
		DBBackend:   cfg.Storage.DBBackend,
		DBPath:      cfg.Storage.DBPath,
		LogLevel:    cfg.Logging.Level,
		LogFormat:   cfg.Logging.Format,
		Genesis:     "config/genesis.json",
		NodeKey:     "config/node_key.json",
		ABCI:        cfg.Chain.ABCI,
		FilterPeers: cfg.Chain.FilterPeers,

		RPC: legacyRPC{
			ListenAddress:                cfg.Network.RPC.ListenAddress,
			CORSAllowedOrigins:           cfg.Network.RPC.CORSOrigins,
			CORSAllowedMethods:           cfg.Network.RPC.CORSMethods,
			CORSAllowedHeaders:           cfg.Network.RPC.CORSHeaders,
			Unsafe:                       cfg.Network.RPC.Unsafe,
			MaxOpenConnections:           cfg.Network.RPC.MaxOpenConnections,
			MaxSubscriptionClients:       cfg.Network.RPC.MaxSubscriptionClients,
			MaxSubscriptionsPerClient:    cfg.Network.RPC.MaxSubscriptionsPerClient,
			ExperimentalDisableWebsocket: cfg.Network.RPC.ExperimentalDisableWebsocket,
			EventLogWindowSize:           cfg.Network.RPC.EventLogWindowSize,
			EventLogMaxItems:             cfg.Network.RPC.EventLogMaxItems,
			TimeoutBroadcastTxCommit:     cfg.Network.RPC.TimeoutBroadcastTxCommit,
			MaxBodyBytes:                 cfg.Network.RPC.MaxBodyBytes,
			MaxHeaderBytes:               cfg.Network.RPC.MaxHeaderBytes,
			TLSCertFile:                  cfg.Network.RPC.TLSCertFile,
			TLSKeyFile:                   cfg.Network.RPC.TLSKeyFile,
			PprofListenAddress:           cfg.Network.RPC.PprofListenAddress,
			LagThreshold:                 cfg.Network.RPC.LagThreshold,
			TimeoutRead:                  cfg.Network.RPC.TimeoutRead,
		},

		P2P: legacyP2P{
			ListenAddress:                 cfg.Network.P2P.ListenAddress,
			ExternalAddress:               cfg.Network.P2P.ExternalAddress,
			BootstrapPeers:                cfg.Network.P2P.BootstrapPeers,
			PersistentPeers:               cfg.Network.P2P.PersistentPeers,
			BlockSyncPeers:                cfg.Network.P2P.BlockSyncPeers,
			UPNP:                          cfg.Network.P2P.UPNP,
			MaxConnections:                cfg.Network.P2P.MaxConnections,
			MaxIncomingConnectionAttempts: cfg.Network.P2P.MaxIncomingConnectionAttempts,
			PexReactor:                    cfg.Network.P2P.PexReactor,
			PrivatePeerIDs:                cfg.Network.P2P.PrivatePeerIDs,
			AllowDuplicateIP:              cfg.Network.P2P.AllowDuplicateIP,
			UnconditionalPeerIDs:          cfg.Network.P2P.UnconditionalPeerIDs,
			FlushThrottleTimeout:          cfg.Network.P2P.FlushThrottleTimeout,
			MaxPacketMsgPayloadSize:       cfg.Network.P2P.MaxPacketMsgPayloadSize,
			SendRate:                      cfg.Network.P2P.SendRate,
			RecvRate:                      cfg.Network.P2P.RecvRate,
			HandshakeTimeout:              cfg.Network.P2P.HandshakeTimeout,
			DialTimeout:                   cfg.Network.P2P.DialTimeout,
			DialInterval:                  cfg.Network.P2P.DialInterval,
			QueueType:                     cfg.Network.P2P.QueueType,
		},

		Mempool: legacyMempool{
			Broadcast:                    cfg.Mempool.Broadcast,
			Size:                         cfg.Mempool.Size,
			MaxTxsBytes:                  cfg.Mempool.MaxTxsBytes,
			CacheSize:                    cfg.Mempool.CacheSize,
			DuplicateTxsCacheSize:        cfg.Mempool.DuplicateTxsCacheSize,
			KeepInvalidTxsInCache:        cfg.Mempool.KeepInvalidTxsInCache,
			MaxTxBytes:                   cfg.Mempool.MaxTxBytes,
			MaxBatchBytes:                cfg.Mempool.MaxBatchBytes,
			TTLDuration:                  cfg.Mempool.TTLDuration,
			TTLNumBlocks:                 cfg.Mempool.TTLNumBlocks,
			TxNotifyThreshold:            cfg.Mempool.TxNotifyThreshold,
			CheckTxErrorBlacklistEnabled: cfg.Mempool.CheckTxErrorBlacklistEnabled,
			CheckTxErrorThreshold:        cfg.Mempool.CheckTxErrorThreshold,
			PendingSize:                  cfg.Mempool.PendingSize,
			MaxPendingTxsBytes:           cfg.Mempool.MaxPendingTxsBytes,
			PendingTTLDuration:           cfg.Mempool.PendingTTLDuration,
			PendingTTLNumBlocks:          cfg.Mempool.PendingTTLNumBlocks,
			RemoveExpiredTxsFromQueue:    cfg.Mempool.RemoveExpiredTxsFromQueue,
			DropPriorityThreshold:        cfg.Mempool.DropPriorityThreshold,
			DropUtilisationThreshold:     cfg.Mempool.DropUtilisationThreshold,
			DropPriorityReservoirSize:    cfg.Mempool.DropPriorityReservoirSize,
		},

		StateSync: legacyStateSync{
			Enable:                    cfg.StateSync.Enable,
			UseP2P:                    cfg.StateSync.UseP2P,
			RPCServers:                cfg.StateSync.RPCServers,
			TrustHeight:               cfg.StateSync.TrustHeight,
			TrustHash:                 cfg.StateSync.TrustHash,
			TrustPeriod:               cfg.StateSync.TrustPeriod,
			BackfillBlocks:            cfg.StateSync.BackfillBlocks,
			BackfillDuration:          cfg.StateSync.BackfillDuration,
			DiscoveryTime:             cfg.StateSync.DiscoveryTime,
			TempDir:                   cfg.StateSync.TempDir,
			ChunkRequestTimeout:       cfg.StateSync.ChunkRequestTimeout,
			Fetchers:                  cfg.StateSync.Fetchers,
			VerifyLightBlockTimeout:   cfg.StateSync.VerifyLightBlockTimeout,
			BlacklistTTL:              cfg.StateSync.BlacklistTTL,
			UseLocalSnapshot:          cfg.StateSync.UseLocalSnapshot,
			LightBlockResponseTimeout: cfg.StateSync.LightBlockResponseTimeout,
		},

		Consensus: legacyConsensus{
			WALPath:                           cfg.Consensus.WALPath,
			CreateEmptyBlocks:                 cfg.Consensus.CreateEmptyBlocks,
			CreateEmptyBlocksInterval:         cfg.Consensus.CreateEmptyBlocksInterval,
			GossipTransactionKeyOnly:          cfg.Consensus.GossipTransactionKeyOnly,
			PeerGossipSleepDuration:           cfg.Consensus.PeerGossipSleepDuration,
			PeerQueryMaj23SleepDuration:       cfg.Consensus.PeerQueryMaj23SleepDuration,
			DoubleSignCheckHeight:             cfg.Consensus.DoubleSignCheckHeight,
			UnsafeProposeTimeoutOverride:      cfg.Consensus.UnsafeProposeTimeoutOverride,
			UnsafeProposeTimeoutDeltaOverride: cfg.Consensus.UnsafeProposeTimeoutDeltaOverride,
			UnsafeVoteTimeoutOverride:         cfg.Consensus.UnsafeVoteTimeoutOverride,
			UnsafeVoteTimeoutDeltaOverride:    cfg.Consensus.UnsafeVoteTimeoutDeltaOverride,
			UnsafeCommitTimeoutOverride:       cfg.Consensus.UnsafeCommitTimeoutOverride,
			UnsafeBypassCommitTimeoutOverride: cfg.Consensus.UnsafeBypassCommitTimeoutOverride,
		},

		TxIndex: legacyTxIndex{
			Indexer:  cfg.TxIndex.Indexer,
			PsqlConn: cfg.TxIndex.PsqlConn,
		},

		Instrumentation: legacyInstrumentation{
			Prometheus:           cfg.Metrics.Enabled,
			PrometheusListenAddr: cfg.Metrics.PrometheusListenAddr,
			MaxOpenConnections:   cfg.Metrics.MaxOpenConnections,
			Namespace:            cfg.Metrics.Namespace,
		},

		PrivValidator: legacyPrivValidator{
			KeyFile:           cfg.PrivValidator.KeyFile,
			StateFile:         cfg.PrivValidator.StateFile,
			ListenAddr:        cfg.PrivValidator.ListenAddr,
			ClientCertificate: cfg.PrivValidator.ClientCertificate,
			ClientKey:         cfg.PrivValidator.ClientKey,
			RootCA:            cfg.PrivValidator.RootCA,
		},

		SelfRemediation: legacySelfRemediation{
			P2PNoPeersRestartWindowSeconds:       cfg.SelfRemediation.P2PNoPeersRestartWindowSeconds,
			StatesyncNoPeersRestartWindowSeconds: cfg.SelfRemediation.StatesyncNoPeersRestartWindowSeconds,
			BlocksBehindThreshold:                cfg.SelfRemediation.BlocksBehindThreshold,
			BlocksBehindCheckIntervalSeconds:     cfg.SelfRemediation.BlocksBehindCheckIntervalSeconds,
			RestartCooldownSeconds:               cfg.SelfRemediation.RestartCooldownSeconds,
		},
	}
}

func (cfg *SeiConfig) toLegacyApp() legacyAppConfig {
	return legacyAppConfig{
		MinGasPrices:        cfg.Chain.MinGasPrices,
		Pruning:             cfg.Storage.PruningStrategy,
		PruningKeepRecent:   cfg.Storage.PruningKeepRecent,
		PruningKeepEvery:    cfg.Storage.PruningKeepEvery,
		PruningInterval:     cfg.Storage.PruningInterval,
		HaltHeight:          cfg.Chain.HaltHeight,
		HaltTime:            cfg.Chain.HaltTime,
		MinRetainBlocks:     cfg.Chain.MinRetainBlocks,
		InterBlockCache:     cfg.Chain.InterBlockCache,
		IndexEvents:         cfg.Chain.IndexEvents,
		IAVLDisableFastNode: cfg.Storage.IAVLDisableFastNode,
		CompactionInterval:  cfg.Storage.CompactionInterval,
		ConcurrencyWorkers:  cfg.Chain.ConcurrencyWorkers,
		OccEnabled:          cfg.Chain.OccEnabled,

		Telemetry: legacyTelemetry{
			ServiceName:             cfg.Metrics.ServiceName,
			Enabled:                 cfg.Metrics.Enabled,
			EnableHostname:          cfg.Metrics.EnableHostname,
			EnableHostnameLabel:     cfg.Metrics.EnableHostnameLabel,
			EnableServiceLabel:      cfg.Metrics.EnableServiceLabel,
			PrometheusRetentionTime: cfg.Metrics.PrometheusRetentionTime,
			GlobalLabels:            cfg.Metrics.GlobalLabels,
		},

		API: legacyAPI{
			Enable:             cfg.API.REST.Enable,
			Swagger:            cfg.API.REST.Swagger,
			Address:            cfg.API.REST.Address,
			MaxOpenConnections: cfg.API.REST.MaxOpenConnections,
			RPCReadTimeout:     cfg.API.REST.RPCReadTimeout,
			RPCWriteTimeout:    cfg.API.REST.RPCWriteTimeout,
			RPCMaxBodyBytes:    cfg.API.REST.RPCMaxBodyBytes,
			EnableUnsafeCORS:   cfg.API.REST.EnableUnsafeCORS,
		},

		GRPC: legacyGRPC{
			Enable:  cfg.API.GRPC.Enable,
			Address: cfg.API.GRPC.Address,
		},

		GRPCWeb: legacyGRPCWeb{
			Enable:           cfg.API.GRPCWeb.Enable,
			Address:          cfg.API.GRPCWeb.Address,
			EnableUnsafeCORS: cfg.API.GRPCWeb.EnableUnsafeCORS,
		},

		StateSync: legacyAppStateSync{
			SnapshotInterval:   cfg.Storage.SnapshotInterval,
			SnapshotKeepRecent: cfg.Storage.SnapshotKeepRecent,
			SnapshotDirectory:  cfg.Storage.SnapshotDirectory,
		},

		StateCommit: legacyStateCommit{
			Enable:                    cfg.Storage.StateCommit.Enable,
			Directory:                 cfg.Storage.StateCommit.Directory,
			AsyncCommitBuffer:         cfg.Storage.StateCommit.AsyncCommitBuffer,
			WriteMode:                 string(cfg.Storage.StateCommit.WriteMode),
			ReadMode:                  string(cfg.Storage.StateCommit.ReadMode),
			KeepRecent:                cfg.Storage.StateCommit.MemIAVL.SnapshotKeepRecent,
			SnapshotInterval:          cfg.Storage.StateCommit.MemIAVL.SnapshotInterval,
			SnapshotMinTimeInterval:   cfg.Storage.StateCommit.MemIAVL.SnapshotMinTimeInterval,
			SnapshotWriterLimit:       cfg.Storage.StateCommit.MemIAVL.SnapshotWriterLimit,
			SnapshotPrefetchThreshold: cfg.Storage.StateCommit.MemIAVL.SnapshotPrefetchThreshold,
		},

		StateStore: legacyStateStore{
			Enable:               cfg.Storage.StateStore.Enable,
			DBDirectory:          cfg.Storage.StateStore.DBDirectory,
			Backend:              cfg.Storage.StateStore.Backend,
			AsyncWriteBuffer:     cfg.Storage.StateStore.AsyncWriteBuffer,
			KeepRecent:           cfg.Storage.StateStore.KeepRecent,
			PruneIntervalSeconds: cfg.Storage.StateStore.PruneIntervalSeconds,
			ImportNumWorkers:     cfg.Storage.StateStore.ImportNumWorkers,
			KeepLastVersion:      cfg.Storage.StateStore.KeepLastVersion,
			UseDefaultComparer:   cfg.Storage.StateStore.UseDefaultComparer,
			WriteMode:            string(cfg.Storage.StateStore.WriteMode),
			ReadMode:             string(cfg.Storage.StateStore.ReadMode),
			EVMDBDirectory:       cfg.Storage.StateStore.EVMDBDirectory,
		},

		EVM: legacyEVM{
			HTTPEnabled:                  cfg.EVM.HTTPEnabled,
			HTTPPort:                     cfg.EVM.HTTPPort,
			WSEnabled:                    cfg.EVM.WSEnabled,
			WSPort:                       cfg.EVM.WSPort,
			ReadTimeout:                  cfg.EVM.ReadTimeout,
			ReadHeaderTimeout:            cfg.EVM.ReadHeaderTimeout,
			WriteTimeout:                 cfg.EVM.WriteTimeout,
			IdleTimeout:                  cfg.EVM.IdleTimeout,
			SimulationGasLimit:           cfg.EVM.SimulationGasLimit,
			SimulationEVMTimeout:         cfg.EVM.SimulationEVMTimeout,
			CORSOrigins:                  cfg.EVM.CORSOrigins,
			WSOrigins:                    cfg.EVM.WSOrigins,
			FilterTimeout:                cfg.EVM.FilterTimeout,
			CheckTxTimeout:               cfg.EVM.CheckTxTimeout,
			MaxTxPoolTxs:                 cfg.EVM.MaxTxPoolTxs,
			Slow:                         cfg.EVM.Slow,
			DenyList:                     cfg.EVM.DenyList,
			MaxLogNoBlock:                cfg.EVM.MaxLogNoBlock,
			MaxBlocksForLog:              cfg.EVM.MaxBlocksForLog,
			MaxSubscriptionsNewHead:      cfg.EVM.MaxSubscriptionsNewHead,
			EnableTestAPI:                cfg.EVM.EnableTestAPI,
			MaxConcurrentTraceCalls:      cfg.EVM.MaxConcurrentTraceCalls,
			MaxConcurrentSimulationCalls: cfg.EVM.MaxConcurrentSimulationCalls,
			MaxTraceLookbackBlocks:       cfg.EVM.MaxTraceLookbackBlocks,
			TraceTimeout:                 cfg.EVM.TraceTimeout,
			RPCStatsInterval:             cfg.EVM.RPCStatsInterval,
			WorkerPoolSize:               cfg.EVM.WorkerPoolSize,
			WorkerQueueSize:              cfg.EVM.WorkerQueueSize,
		},

		WASM: legacyWASM{
			QueryGasLimit: cfg.WASM.QueryGasLimit,
			LruSize:       cfg.WASM.LruSize,
		},

		GigaExecutor: legacyGigaExecutor{
			Enabled:    cfg.GigaExecutor.Enabled,
			OccEnabled: cfg.GigaExecutor.OccEnabled,
		},

		ETHReplay: legacyETHReplay{
			Enabled:             cfg.EVM.Replay.Enabled,
			EthRPC:              cfg.EVM.Replay.EthRPC,
			EthDataDir:          cfg.EVM.Replay.EthDataDir,
			ContractStateChecks: cfg.EVM.Replay.ContractStateChecks,
		},

		ETHBlockTest: legacyETHBlockTest{
			Enabled:      cfg.EVM.BlockTest.Enabled,
			TestDataPath: cfg.EVM.BlockTest.TestDataPath,
		},

		EVMQuery: legacyEVMQuery{
			GasLimit: cfg.EVM.Query.GasLimit,
		},

		LightInvariance: legacyLightInvariance{
			SupplyEnabled: cfg.LightInvariance.SupplyEnabled,
		},

		Genesis: legacyGenesis{
			StreamImport:      cfg.Genesis.StreamImport,
			GenesisStreamFile: cfg.Genesis.GenesisStreamFile,
		},

		SeiMeta: legacySeiMeta{
			Mode:    string(cfg.Mode),
			Version: cfg.Version,
		},
	}
}

func fromLegacy(tm legacyTendermintConfig, app legacyAppConfig) *SeiConfig {
	// Prefer the sei metadata section (preserves archive/rpc/indexer) over the
	// Tendermint mode field (which only supports validator/full/seed).
	mode := NodeMode(app.SeiMeta.Mode)
	if !mode.IsValid() {
		mode = NodeMode(tm.Mode)
		if !mode.IsValid() {
			mode = ModeFull
		}
	}
	version := app.SeiMeta.Version
	if version < 1 {
		version = CurrentVersion
	}

	return &SeiConfig{
		Version: version,
		Mode:    mode,

		Chain: ChainConfig{
			Moniker:            tm.Moniker,
			ProxyApp:           tm.ProxyApp,
			ABCI:               tm.ABCI,
			FilterPeers:        tm.FilterPeers,
			MinGasPrices:       app.MinGasPrices,
			HaltHeight:         app.HaltHeight,
			HaltTime:           app.HaltTime,
			MinRetainBlocks:    app.MinRetainBlocks,
			InterBlockCache:    app.InterBlockCache,
			IndexEvents:        app.IndexEvents,
			ConcurrencyWorkers: app.ConcurrencyWorkers,
			OccEnabled:         app.OccEnabled,
		},

		Network: NetworkConfig{
			RPC: RPCConfig{
				ListenAddress:                tm.RPC.ListenAddress,
				CORSOrigins:                  tm.RPC.CORSAllowedOrigins,
				CORSMethods:                  tm.RPC.CORSAllowedMethods,
				CORSHeaders:                  tm.RPC.CORSAllowedHeaders,
				Unsafe:                       tm.RPC.Unsafe,
				MaxOpenConnections:           tm.RPC.MaxOpenConnections,
				MaxSubscriptionClients:       tm.RPC.MaxSubscriptionClients,
				MaxSubscriptionsPerClient:    tm.RPC.MaxSubscriptionsPerClient,
				ExperimentalDisableWebsocket: tm.RPC.ExperimentalDisableWebsocket,
				EventLogWindowSize:           tm.RPC.EventLogWindowSize,
				EventLogMaxItems:             tm.RPC.EventLogMaxItems,
				TimeoutBroadcastTxCommit:     tm.RPC.TimeoutBroadcastTxCommit,
				MaxBodyBytes:                 tm.RPC.MaxBodyBytes,
				MaxHeaderBytes:               tm.RPC.MaxHeaderBytes,
				TLSCertFile:                  tm.RPC.TLSCertFile,
				TLSKeyFile:                   tm.RPC.TLSKeyFile,
				PprofListenAddress:           tm.RPC.PprofListenAddress,
				LagThreshold:                 tm.RPC.LagThreshold,
				TimeoutRead:                  tm.RPC.TimeoutRead,
			},
			P2P: P2PConfig{
				ListenAddress:                 tm.P2P.ListenAddress,
				ExternalAddress:               tm.P2P.ExternalAddress,
				BootstrapPeers:                tm.P2P.BootstrapPeers,
				PersistentPeers:               tm.P2P.PersistentPeers,
				BlockSyncPeers:                tm.P2P.BlockSyncPeers,
				UPNP:                          tm.P2P.UPNP,
				MaxConnections:                tm.P2P.MaxConnections,
				MaxIncomingConnectionAttempts: tm.P2P.MaxIncomingConnectionAttempts,
				PexReactor:                    tm.P2P.PexReactor,
				PrivatePeerIDs:                tm.P2P.PrivatePeerIDs,
				AllowDuplicateIP:              tm.P2P.AllowDuplicateIP,
				UnconditionalPeerIDs:          tm.P2P.UnconditionalPeerIDs,
				FlushThrottleTimeout:          tm.P2P.FlushThrottleTimeout,
				MaxPacketMsgPayloadSize:       tm.P2P.MaxPacketMsgPayloadSize,
				SendRate:                      tm.P2P.SendRate,
				RecvRate:                      tm.P2P.RecvRate,
				HandshakeTimeout:              tm.P2P.HandshakeTimeout,
				DialTimeout:                   tm.P2P.DialTimeout,
				DialInterval:                  tm.P2P.DialInterval,
				QueueType:                     tm.P2P.QueueType,
			},
		},

		Consensus: ConsensusConfig{
			WALPath:                           tm.Consensus.WALPath,
			CreateEmptyBlocks:                 tm.Consensus.CreateEmptyBlocks,
			CreateEmptyBlocksInterval:         tm.Consensus.CreateEmptyBlocksInterval,
			GossipTransactionKeyOnly:          tm.Consensus.GossipTransactionKeyOnly,
			PeerGossipSleepDuration:           tm.Consensus.PeerGossipSleepDuration,
			PeerQueryMaj23SleepDuration:       tm.Consensus.PeerQueryMaj23SleepDuration,
			DoubleSignCheckHeight:             tm.Consensus.DoubleSignCheckHeight,
			UnsafeProposeTimeoutOverride:      tm.Consensus.UnsafeProposeTimeoutOverride,
			UnsafeProposeTimeoutDeltaOverride: tm.Consensus.UnsafeProposeTimeoutDeltaOverride,
			UnsafeVoteTimeoutOverride:         tm.Consensus.UnsafeVoteTimeoutOverride,
			UnsafeVoteTimeoutDeltaOverride:    tm.Consensus.UnsafeVoteTimeoutDeltaOverride,
			UnsafeCommitTimeoutOverride:       tm.Consensus.UnsafeCommitTimeoutOverride,
			UnsafeBypassCommitTimeoutOverride: tm.Consensus.UnsafeBypassCommitTimeoutOverride,
		},

		Mempool: MempoolConfig{
			Broadcast:                    tm.Mempool.Broadcast,
			Size:                         tm.Mempool.Size,
			MaxTxsBytes:                  tm.Mempool.MaxTxsBytes,
			CacheSize:                    tm.Mempool.CacheSize,
			DuplicateTxsCacheSize:        tm.Mempool.DuplicateTxsCacheSize,
			KeepInvalidTxsInCache:        tm.Mempool.KeepInvalidTxsInCache,
			MaxTxBytes:                   tm.Mempool.MaxTxBytes,
			MaxBatchBytes:                tm.Mempool.MaxBatchBytes,
			TTLDuration:                  tm.Mempool.TTLDuration,
			TTLNumBlocks:                 tm.Mempool.TTLNumBlocks,
			TxNotifyThreshold:            tm.Mempool.TxNotifyThreshold,
			CheckTxErrorBlacklistEnabled: tm.Mempool.CheckTxErrorBlacklistEnabled,
			CheckTxErrorThreshold:        tm.Mempool.CheckTxErrorThreshold,
			PendingSize:                  tm.Mempool.PendingSize,
			MaxPendingTxsBytes:           tm.Mempool.MaxPendingTxsBytes,
			PendingTTLDuration:           tm.Mempool.PendingTTLDuration,
			PendingTTLNumBlocks:          tm.Mempool.PendingTTLNumBlocks,
			RemoveExpiredTxsFromQueue:    tm.Mempool.RemoveExpiredTxsFromQueue,
			DropPriorityThreshold:        tm.Mempool.DropPriorityThreshold,
			DropUtilisationThreshold:     tm.Mempool.DropUtilisationThreshold,
			DropPriorityReservoirSize:    tm.Mempool.DropPriorityReservoirSize,
		},

		StateSync: StateSyncConfig{
			Enable:                    tm.StateSync.Enable,
			UseP2P:                    tm.StateSync.UseP2P,
			RPCServers:                tm.StateSync.RPCServers,
			TrustHeight:               tm.StateSync.TrustHeight,
			TrustHash:                 tm.StateSync.TrustHash,
			TrustPeriod:               tm.StateSync.TrustPeriod,
			BackfillBlocks:            tm.StateSync.BackfillBlocks,
			BackfillDuration:          tm.StateSync.BackfillDuration,
			DiscoveryTime:             tm.StateSync.DiscoveryTime,
			TempDir:                   tm.StateSync.TempDir,
			ChunkRequestTimeout:       tm.StateSync.ChunkRequestTimeout,
			Fetchers:                  tm.StateSync.Fetchers,
			VerifyLightBlockTimeout:   tm.StateSync.VerifyLightBlockTimeout,
			BlacklistTTL:              tm.StateSync.BlacklistTTL,
			UseLocalSnapshot:          tm.StateSync.UseLocalSnapshot,
			LightBlockResponseTimeout: tm.StateSync.LightBlockResponseTimeout,
		},

		Storage: StorageConfig{
			DBBackend:           tm.DBBackend,
			DBPath:              tm.DBPath,
			PruningStrategy:     app.Pruning,
			PruningKeepRecent:   app.PruningKeepRecent,
			PruningKeepEvery:    app.PruningKeepEvery,
			PruningInterval:     app.PruningInterval,
			SnapshotInterval:    app.StateSync.SnapshotInterval,
			SnapshotKeepRecent:  app.StateSync.SnapshotKeepRecent,
			SnapshotDirectory:   app.StateSync.SnapshotDirectory,
			CompactionInterval:  app.CompactionInterval,
			IAVLDisableFastNode: app.IAVLDisableFastNode,
			StateCommit: StateCommitConfig{
				Enable:            app.StateCommit.Enable,
				Directory:         app.StateCommit.Directory,
				AsyncCommitBuffer: app.StateCommit.AsyncCommitBuffer,
				WriteMode:         WriteMode(app.StateCommit.WriteMode),
				ReadMode:          ReadMode(app.StateCommit.ReadMode),
				MemIAVL: MemIAVLConfig{
					SnapshotKeepRecent:        app.StateCommit.KeepRecent,
					SnapshotInterval:          app.StateCommit.SnapshotInterval,
					SnapshotMinTimeInterval:   app.StateCommit.SnapshotMinTimeInterval,
					SnapshotWriterLimit:       app.StateCommit.SnapshotWriterLimit,
					SnapshotPrefetchThreshold: app.StateCommit.SnapshotPrefetchThreshold,
				},
			},
			StateStore: StateStoreConfig{
				Enable:               app.StateStore.Enable,
				DBDirectory:          app.StateStore.DBDirectory,
				Backend:              app.StateStore.Backend,
				AsyncWriteBuffer:     app.StateStore.AsyncWriteBuffer,
				KeepRecent:           app.StateStore.KeepRecent,
				PruneIntervalSeconds: app.StateStore.PruneIntervalSeconds,
				ImportNumWorkers:     app.StateStore.ImportNumWorkers,
				KeepLastVersion:      app.StateStore.KeepLastVersion,
				UseDefaultComparer:   app.StateStore.UseDefaultComparer,
				WriteMode:            WriteMode(app.StateStore.WriteMode),
				ReadMode:             ReadMode(app.StateStore.ReadMode),
				EVMDBDirectory:       app.StateStore.EVMDBDirectory,
			},
		},

		TxIndex: TxIndexConfig{
			Indexer:  tm.TxIndex.Indexer,
			PsqlConn: tm.TxIndex.PsqlConn,
		},

		EVM: EVMConfig{
			HTTPEnabled:                  app.EVM.HTTPEnabled,
			HTTPPort:                     app.EVM.HTTPPort,
			WSEnabled:                    app.EVM.WSEnabled,
			WSPort:                       app.EVM.WSPort,
			ReadTimeout:                  app.EVM.ReadTimeout,
			ReadHeaderTimeout:            app.EVM.ReadHeaderTimeout,
			WriteTimeout:                 app.EVM.WriteTimeout,
			IdleTimeout:                  app.EVM.IdleTimeout,
			SimulationGasLimit:           app.EVM.SimulationGasLimit,
			SimulationEVMTimeout:         app.EVM.SimulationEVMTimeout,
			CORSOrigins:                  app.EVM.CORSOrigins,
			WSOrigins:                    app.EVM.WSOrigins,
			FilterTimeout:                app.EVM.FilterTimeout,
			CheckTxTimeout:               app.EVM.CheckTxTimeout,
			MaxTxPoolTxs:                 app.EVM.MaxTxPoolTxs,
			Slow:                         app.EVM.Slow,
			DenyList:                     app.EVM.DenyList,
			MaxLogNoBlock:                app.EVM.MaxLogNoBlock,
			MaxBlocksForLog:              app.EVM.MaxBlocksForLog,
			MaxSubscriptionsNewHead:      app.EVM.MaxSubscriptionsNewHead,
			EnableTestAPI:                app.EVM.EnableTestAPI,
			MaxConcurrentTraceCalls:      app.EVM.MaxConcurrentTraceCalls,
			MaxConcurrentSimulationCalls: app.EVM.MaxConcurrentSimulationCalls,
			MaxTraceLookbackBlocks:       app.EVM.MaxTraceLookbackBlocks,
			TraceTimeout:                 app.EVM.TraceTimeout,
			RPCStatsInterval:             app.EVM.RPCStatsInterval,
			WorkerPoolSize:               app.EVM.WorkerPoolSize,
			WorkerQueueSize:              app.EVM.WorkerQueueSize,
			Query: EVMQueryConfig{
				GasLimit: app.EVMQuery.GasLimit,
			},
			Replay: EVMReplayConfig{
				Enabled:             app.ETHReplay.Enabled,
				EthRPC:              app.ETHReplay.EthRPC,
				EthDataDir:          app.ETHReplay.EthDataDir,
				ContractStateChecks: app.ETHReplay.ContractStateChecks,
			},
			BlockTest: EVMBlockTestConfig{
				Enabled:      app.ETHBlockTest.Enabled,
				TestDataPath: app.ETHBlockTest.TestDataPath,
			},
		},

		API: APIConfig{
			REST: RESTAPIConfig{
				Enable:             app.API.Enable,
				Swagger:            app.API.Swagger,
				Address:            app.API.Address,
				MaxOpenConnections: app.API.MaxOpenConnections,
				RPCReadTimeout:     app.API.RPCReadTimeout,
				RPCWriteTimeout:    app.API.RPCWriteTimeout,
				RPCMaxBodyBytes:    app.API.RPCMaxBodyBytes,
				EnableUnsafeCORS:   app.API.EnableUnsafeCORS,
			},
			GRPC: GRPCConfig{
				Enable:  app.GRPC.Enable,
				Address: app.GRPC.Address,
			},
			GRPCWeb: GRPCWebConfig{
				Enable:           app.GRPCWeb.Enable,
				Address:          app.GRPCWeb.Address,
				EnableUnsafeCORS: app.GRPCWeb.EnableUnsafeCORS,
			},
		},

		Metrics: MetricsConfig{
			Enabled:                 app.Telemetry.Enabled,
			PrometheusListenAddr:    tm.Instrumentation.PrometheusListenAddr,
			MaxOpenConnections:      tm.Instrumentation.MaxOpenConnections,
			Namespace:               tm.Instrumentation.Namespace,
			ServiceName:             app.Telemetry.ServiceName,
			EnableHostname:          app.Telemetry.EnableHostname,
			EnableHostnameLabel:     app.Telemetry.EnableHostnameLabel,
			EnableServiceLabel:      app.Telemetry.EnableServiceLabel,
			PrometheusRetentionTime: app.Telemetry.PrometheusRetentionTime,
			GlobalLabels:            app.Telemetry.GlobalLabels,
		},

		Logging: LogConfig{
			Level:  tm.LogLevel,
			Format: tm.LogFormat,
		},

		WASM: WASMConfig{
			QueryGasLimit: app.WASM.QueryGasLimit,
			LruSize:       app.WASM.LruSize,
		},

		GigaExecutor: GigaExecutorConfig{
			Enabled:    app.GigaExecutor.Enabled,
			OccEnabled: app.GigaExecutor.OccEnabled,
		},

		LightInvariance: LightInvarianceConfig{
			SupplyEnabled: app.LightInvariance.SupplyEnabled,
		},

		PrivValidator: PrivValidatorConfig{
			KeyFile:           tm.PrivValidator.KeyFile,
			StateFile:         tm.PrivValidator.StateFile,
			ListenAddr:        tm.PrivValidator.ListenAddr,
			ClientCertificate: tm.PrivValidator.ClientCertificate,
			ClientKey:         tm.PrivValidator.ClientKey,
			RootCA:            tm.PrivValidator.RootCA,
		},

		SelfRemediation: SelfRemediationConfig{
			P2PNoPeersRestartWindowSeconds:       tm.SelfRemediation.P2PNoPeersRestartWindowSeconds,
			StatesyncNoPeersRestartWindowSeconds: tm.SelfRemediation.StatesyncNoPeersRestartWindowSeconds,
			BlocksBehindThreshold:                tm.SelfRemediation.BlocksBehindThreshold,
			BlocksBehindCheckIntervalSeconds:     tm.SelfRemediation.BlocksBehindCheckIntervalSeconds,
			RestartCooldownSeconds:               tm.SelfRemediation.RestartCooldownSeconds,
		},

		Genesis: GenesisConfig{
			StreamImport:      app.Genesis.StreamImport,
			GenesisStreamFile: app.Genesis.GenesisStreamFile,
		},
	}
}
