package seiconfig

import (
	"os"
	"strconv"
	"time"
)

// Default returns a SeiConfig populated with baseline defaults (mode=full).
// Use DefaultForMode to get defaults tailored to a specific node mode.
func Default() *SeiConfig {
	return DefaultForMode(ModeFull)
}

// DefaultForMode returns a SeiConfig with defaults appropriate for the given mode.
func DefaultForMode(mode NodeMode) *SeiConfig {
	cfg := baseDefaults()
	cfg.Mode = mode
	applyModeOverrides(cfg, mode)
	return cfg
}

// baseDefaults returns the shared baseline configuration before mode overrides.
func baseDefaults() *SeiConfig {
	return &SeiConfig{
		Version: CurrentVersion,
		Mode:    ModeFull,

		Chain: ChainConfig{
			Moniker:            defaultMoniker(),
			ProxyApp:           "tcp://127.0.0.1:26658",
			ABCI:               "socket",
			MinGasPrices:       "0.01usei",
			InterBlockCache:    true,
			ConcurrencyWorkers: defaultConcurrencyWorkers(),
			OccEnabled:         true,
		},

		Network: NetworkConfig{
			RPC: RPCConfig{
				ListenAddress:             "tcp://127.0.0.1:26657",
				CORSOrigins:               []string{},
				CORSMethods:               []string{"HEAD", "GET", "POST"},
				CORSHeaders:               []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "X-Server-Time"},
				Unsafe:                    false,
				MaxOpenConnections:        900,
				MaxSubscriptionClients:    100,
				MaxSubscriptionsPerClient: 5,
				EventLogWindowSize:        Dur(30 * time.Second),
				TimeoutBroadcastTxCommit:  Dur(10 * time.Second),
				MaxBodyBytes:              1_000_000,
				MaxHeaderBytes:            1 << 20,
				LagThreshold:              300,
				TimeoutRead:               Dur(10 * time.Second),
			},
			P2P: P2PConfig{
				ListenAddress:                 "tcp://127.0.0.1:26656",
				MaxConnections:                100,
				MaxIncomingConnectionAttempts: 100,
				FlushThrottleTimeout:          Dur(100 * time.Millisecond),
				MaxPacketMsgPayloadSize:       1_000_000,
				SendRate:                      20_971_520,
				RecvRate:                      20_971_520,
				PexReactor:                    true,
				HandshakeTimeout:              Dur(10 * time.Second),
				DialTimeout:                   Dur(3 * time.Second),
				DialInterval:                  Dur(10 * time.Second),
				QueueType:                     "simple-priority",
			},
		},

		Consensus: ConsensusConfig{
			WALPath:                     "data/cs.wal/wal",
			CreateEmptyBlocks:           true,
			GossipTransactionKeyOnly:    true,
			PeerGossipSleepDuration:     Dur(100 * time.Millisecond),
			PeerQueryMaj23SleepDuration: Dur(2000 * time.Millisecond),
		},

		Mempool: MempoolConfig{
			Broadcast:                    true,
			Size:                         5000,
			MaxTxsBytes:                  1 << 30,
			CacheSize:                    10_000,
			DuplicateTxsCacheSize:        100_000,
			MaxTxBytes:                   1 << 20,
			TTLDuration:                  Dur(5 * time.Second),
			TTLNumBlocks:                 10,
			CheckTxErrorBlacklistEnabled: true,
			CheckTxErrorThreshold:        50,
			PendingSize:                  5000,
			MaxPendingTxsBytes:           1 << 30,
			RemoveExpiredTxsFromQueue:    true,
			DropPriorityThreshold:        0.1,
			DropUtilisationThreshold:     1.0,
			DropPriorityReservoirSize:    10_240,
		},

		StateSync: StateSyncConfig{
			TrustPeriod:               Dur(168 * time.Hour),
			DiscoveryTime:             Dur(15 * time.Second),
			ChunkRequestTimeout:       Dur(15 * time.Second),
			Fetchers:                  2,
			VerifyLightBlockTimeout:   Dur(60 * time.Second),
			BlacklistTTL:              Dur(5 * time.Minute),
			LightBlockResponseTimeout: Dur(10 * time.Second),
		},

		Storage: StorageConfig{
			DBBackend:           "goleveldb",
			DBPath:              "data",
			PruningStrategy:     PruningNothing,
			PruningKeepRecent:   "0",
			PruningKeepEvery:    "0",
			PruningInterval:     "0",
			SnapshotKeepRecent:  2,
			IAVLDisableFastNode: true,
			StateCommit: StateCommitConfig{
				Enable:    true,
				WriteMode: WriteModeCosmosOnly,
				ReadMode:  ReadModeCosmosOnly,
			},
			StateStore: StateStoreConfig{
				Enable:               true,
				Backend:              BackendPebbleDB,
				AsyncWriteBuffer:     100,
				KeepRecent:           100_000,
				PruneIntervalSeconds: 600,
				ImportNumWorkers:     1,
				KeepLastVersion:      true,
				WriteMode:            WriteModeCosmosOnly,
				ReadMode:             ReadModeCosmosOnly,
			},
			ReceiptStore: ReceiptStoreConfig{
				Backend:              BackendPebbleDB,
				AsyncWriteBuffer:     100,
				KeepRecent:           100_000,
				PruneIntervalSeconds: 600,
				TxIndexBackend:       BackendPebbleDB,
			},
		},

		TxIndex: TxIndexConfig{
			Indexer: []string{"kv"},
		},

		EVM: EVMConfig{
			HTTPEnabled:                  true,
			HTTPPort:                     int(PortEVMHTTP),
			WSEnabled:                    true,
			WSPort:                       int(PortEVMWS),
			ReadTimeout:                  Dur(30 * time.Second),
			ReadHeaderTimeout:            Dur(30 * time.Second),
			WriteTimeout:                 Dur(30 * time.Second),
			IdleTimeout:                  Dur(120 * time.Second),
			SimulationGasLimit:           10_000_000,
			SimulationEVMTimeout:         Dur(60 * time.Second),
			CORSOrigins:                  "*",
			WSOrigins:                    "*",
			FilterTimeout:                Dur(120 * time.Second),
			CheckTxTimeout:               Dur(5 * time.Second),
			MaxTxPoolTxs:                 1000,
			DenyList:                     []string{},
			EnabledLegacySeiApis:         []string{},
			MaxLogNoBlock:                10_000,
			MaxBlocksForLog:              2000,
			MaxSubscriptionsNewHead:      10_000,
			MaxConcurrentTraceCalls:      10,
			MaxConcurrentSimulationCalls: defaultEVMWorkerPoolSize(),
			MaxTraceLookbackBlocks:       10_000,
			TraceTimeout:                 Dur(30 * time.Second),
			RPCStatsInterval:             Dur(10 * time.Second),
			WorkerPoolSize:               defaultEVMWorkerPoolSize(),
			WorkerQueueSize:              1000,
			Query: EVMQueryConfig{
				GasLimit: 300_000,
			},
			Replay: EVMReplayConfig{
				EthRPC:     "http://44.234.105.54:18545",
				EthDataDir: "/root/.ethereum/chaindata",
			},
			BlockTest: EVMBlockTestConfig{
				TestDataPath: "~/testdata/",
			},
		},

		API: APIConfig{
			REST: RESTAPIConfig{
				Enable:             false,
				Swagger:            true,
				Address:            "tcp://0.0.0.0:1317",
				MaxOpenConnections: 1000,
				RPCReadTimeout:     10,
				RPCMaxBodyBytes:    1_000_000,
			},
			GRPC: GRPCConfig{
				Enable:  true,
				Address: "0.0.0.0:9090",
			},
			GRPCWeb: GRPCWebConfig{
				Enable:  true,
				Address: "0.0.0.0:9091",
			},
		},

		Metrics: MetricsConfig{
			Enabled:                 true,
			PrometheusListenAddr:    ":26660",
			MaxOpenConnections:      3,
			Namespace:               "tendermint",
			PrometheusRetentionTime: 7200,
			GlobalLabels:            [][]string{},
		},

		Logging: LogConfig{
			Level:  "info",
			Format: "plain",
		},

		WASM: WASMConfig{
			QueryGasLimit: 300_000,
			LruSize:       1,
		},

		LightInvariance: LightInvarianceConfig{
			SupplyEnabled: true,
		},

		PrivValidator: PrivValidatorConfig{
			KeyFile:   "config/priv_validator_key.json",
			StateFile: "data/priv_validator_state.json",
		},

		SelfRemediation: SelfRemediationConfig{
			BlocksBehindCheckIntervalSeconds: 60,
			RestartCooldownSeconds:           600,
		},
	}
}

// applyModeOverrides mutates cfg in-place with mode-specific settings.
func applyModeOverrides(cfg *SeiConfig, mode NodeMode) {
	switch mode {
	case ModeValidator:
		applyValidatorOverrides(cfg)
	case ModeSeed:
		applySeedOverrides(cfg)
	case ModeFull:
		applyFullOverrides(cfg)
	case ModeArchive:
		applyArchiveOverrides(cfg)
	}
}

func applyValidatorOverrides(cfg *SeiConfig) {
	cfg.TxIndex.Indexer = []string{"null"}
	cfg.Network.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	cfg.Network.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	cfg.Network.P2P.AllowDuplicateIP = false

	cfg.API.REST.Enable = false
	cfg.API.GRPC.Enable = false
	cfg.API.GRPCWeb.Enable = false
	cfg.Storage.StateStore.Enable = false

	cfg.EVM.HTTPEnabled = false
	cfg.EVM.WSEnabled = false
}

func applySeedOverrides(cfg *SeiConfig) {
	cfg.TxIndex.Indexer = []string{"null"}
	cfg.Network.P2P.MaxConnections = 1000
	cfg.Network.P2P.AllowDuplicateIP = true

	cfg.API.REST.Enable = false
	cfg.API.GRPC.Enable = false
	cfg.API.GRPCWeb.Enable = false
	cfg.Storage.StateStore.Enable = false
	cfg.Storage.PruningStrategy = PruningEverything

	cfg.EVM.HTTPEnabled = false
	cfg.EVM.WSEnabled = false
}

func applyFullOverrides(cfg *SeiConfig) {
	cfg.TxIndex.Indexer = []string{"kv"}
	cfg.Network.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	cfg.Network.P2P.ListenAddress = "tcp://0.0.0.0:26656"

	cfg.Chain.ConcurrencyWorkers = 500
	cfg.Storage.PruningStrategy = PruningCustom
	cfg.Storage.PruningKeepRecent = "86400"
	cfg.Storage.PruningKeepEvery = "500"
	cfg.Storage.PruningInterval = "10"

	cfg.API.REST.Enable = true
	cfg.API.GRPC.Enable = true
	cfg.API.GRPCWeb.Enable = true
	cfg.Storage.StateStore.Enable = true
	cfg.Storage.StateStore.KeepRecent = 100_000
	cfg.Chain.MinRetainBlocks = 100_000

	cfg.EVM.HTTPEnabled = true
	cfg.EVM.WSEnabled = true
}

func applyArchiveOverrides(cfg *SeiConfig) {
	applyFullOverrides(cfg)

	cfg.Storage.PruningStrategy = PruningNothing
	cfg.Storage.StateStore.KeepRecent = 0
	// Only MinRetainBlocks disables receipt pruning at runtime; the next two
	// are emitted to document intent (see ReceiptStoreConfig).
	cfg.Chain.MinRetainBlocks = 0
	cfg.Storage.ReceiptStore.KeepRecent = 0
	cfg.Storage.ReceiptStore.PruneIntervalSeconds = 0
	cfg.EVM.MaxTraceLookbackBlocks = -1

	cfg.Storage.StateCommit.AsyncCommitBuffer = 100
	cfg.Storage.StateCommit.MemIAVL.SnapshotKeepRecent = 1
	cfg.Storage.StateCommit.MemIAVL.SnapshotMinTimeInterval = 20000
}

// SnapshotGenerationOverrides returns overrides for full-mode nodes that
// generate CometBFT state-sync snapshots. Includes snapshot creation,
// tighter pruning than the full-node default, and elevated P2P capacity
// for serving snapshot chunks to peers.
func SnapshotGenerationOverrides(keepRecent int32) map[string]string {
	return map[string]string{
		KeySnapshotInterval:   strconv.FormatInt(DefaultSnapshotInterval, 10),
		KeySnapshotKeepRecent: strconv.FormatInt(int64(keepRecent), 10),
		KeyPruningKeepRecent:  "50000",
		KeyPruningKeepEvery:   "0",
		KeyMinRetainBlocks:    "50000",
		KeyP2PMaxConnections:  "500",
		KeyP2PSendRate:        "20971520",
		KeyP2PRecvRate:        "20971520",
	}
}

func defaultMoniker() string {
	name, err := os.Hostname()
	if err != nil {
		return "anonymous"
	}
	return name
}
