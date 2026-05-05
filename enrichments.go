package seiconfig

// DefaultEnrichments returns the curated field metadata for every field
// where description, unit, hot-reload, or deprecation information is known.
// Call registry.EnrichAll(DefaultEnrichments()) after BuildRegistry().
func DefaultEnrichments() map[string][]FieldOption {
	return map[string][]FieldOption{
		// ---------------------------------------------------------------
		// Top-level
		// ---------------------------------------------------------------
		"version": {
			WithDescription("Config schema version. Set by tooling, not operators."),
		},
		"mode": {
			WithDescription("Node operating mode: validator, full, seed, archive."),
		},

		// ---------------------------------------------------------------
		// Chain
		// ---------------------------------------------------------------
		"chain.chain_id": {
			WithDescription("The network chain ID. Sourced from genesis.json at init time."),
		},
		"chain.moniker": {
			WithDescription("A human-readable name for this node, used in P2P and logging."),
		},
		"chain.min_gas_prices": {
			WithDescription("Minimum gas prices a validator will accept for processing a transaction."),
		},
		"chain.halt_height": {
			WithDescription("Block height at which the node will gracefully halt. 0 disables."),
			WithUnit("blocks"),
		},
		"chain.halt_time": {
			WithDescription("Minimum block time (Unix seconds) at which the node will halt. 0 disables."),
			WithUnit("seconds"),
		},
		"chain.min_retain_blocks": {
			WithDescription("Minimum block height offset from current for Tendermint block pruning. 0 keeps all."),
			WithUnit("blocks"),
		},
		"chain.inter_block_cache": {
			WithDescription("Enable inter-block caching for improved read performance."),
		},
		"chain.concurrency_workers": {
			WithDescription("Number of workers for concurrent transaction execution. -1 for unlimited."),
		},
		"chain.occ_enabled": {
			WithDescription("Enable optimistic concurrency control for transaction processing."),
		},

		// ---------------------------------------------------------------
		// Network — RPC
		// ---------------------------------------------------------------
		"network.rpc.listen_address": {
			WithDescription("TCP or UNIX socket address for the Tendermint RPC server."),
			WithHotReload(),
		},
		"network.rpc.cors_allowed_origins": {
			WithDescription("Origins allowed for cross-domain requests. '*' allows all."),
			WithHotReload(),
		},
		"network.rpc.unsafe": {
			WithDescription("Enable unsafe RPC commands like /dial-persistent-peers."),
		},
		"network.rpc.max_open_connections": {
			WithDescription("Maximum simultaneous connections (including WebSocket). 0 for unlimited."),
			WithUnit("connections"),
			WithHotReload(),
		},
		"network.rpc.timeout_broadcast_tx_commit": {
			WithDescription("How long to wait for a tx to be committed during /broadcast_tx_commit."),
		},
		"network.rpc.max_body_bytes": {
			WithDescription("Maximum size of request body."),
			WithUnit("bytes"),
		},
		"network.rpc.lag_threshold": {
			WithDescription("Block lag threshold for the /lag_status health endpoint."),
			WithUnit("blocks"),
			WithHotReload(),
		},

		// ---------------------------------------------------------------
		// Network — P2P
		// ---------------------------------------------------------------
		"network.p2p.listen_address": {
			WithDescription("Address to listen for incoming P2P connections."),
		},
		"network.p2p.external_address": {
			WithDescription("Address to advertise to peers for them to dial."),
		},
		"network.p2p.persistent_peers": {
			WithDescription("Comma-separated list of node IDs to maintain persistent connections to."),
			WithHotReload(),
		},
		"network.p2p.max_connections": {
			WithDescription("Maximum connected peers (inbound + outbound)."),
			WithUnit("connections"),
		},
		"network.p2p.pex": {
			WithDescription("Enable the peer exchange reactor for peer discovery."),
		},
		"network.p2p.allow_duplicate_ip": {
			WithDescription("Allow multiple peers from the same IP address."),
		},
		"network.p2p.send_rate": {
			WithDescription("Rate at which packets can be sent per connection."),
			WithUnit("bytes/sec"),
		},
		"network.p2p.recv_rate": {
			WithDescription("Rate at which packets can be received per connection."),
			WithUnit("bytes/sec"),
		},

		// ---------------------------------------------------------------
		// Consensus
		// ---------------------------------------------------------------
		"consensus.create_empty_blocks": {
			WithDescription("Whether to create blocks even when there are no transactions."),
		},
		"consensus.gossip_transaction_key_only": {
			WithDescription("Gossip only transaction hashes instead of full transactions."),
		},
		"consensus.peer_gossip_sleep_duration": {
			WithDescription("Sleep duration between rounds of peer gossip."),
		},
		"consensus.double_sign_check_height": {
			WithDescription("Number of past blocks to check for double-signing. 0 disables."),
			WithUnit("blocks"),
		},
		"consensus.unsafe_commit_timeout_override": {
			WithDescription("Unsafe override for the Commit timeout consensus parameter."),
		},

		// ---------------------------------------------------------------
		// Mempool
		// ---------------------------------------------------------------
		"mempool.size": {
			WithDescription("Maximum number of transactions in the mempool."),
			WithUnit("transactions"),
			WithHotReload(),
		},
		"mempool.max_txs_bytes": {
			WithDescription("Total size limit for all transactions in the mempool."),
			WithUnit("bytes"),
		},
		"mempool.max_tx_bytes": {
			WithDescription("Maximum size of a single transaction."),
			WithUnit("bytes"),
		},
		"mempool.ttl_duration": {
			WithDescription("Maximum time a transaction can remain in the mempool."),
		},
		"mempool.ttl_num_blocks": {
			WithDescription("Maximum number of blocks a transaction can remain in the mempool."),
			WithUnit("blocks"),
		},

		// ---------------------------------------------------------------
		// State Sync
		// ---------------------------------------------------------------
		"state_sync.enable": {
			WithDescription("Enable state sync to bootstrap from snapshots instead of replaying blocks."),
		},
		"state_sync.trust_height": {
			WithDescription("Height of a trusted block for light client verification."),
			WithUnit("blocks"),
		},
		"state_sync.trust_hash": {
			WithDescription("Hash of the trusted block (hex-encoded)."),
		},
		"state_sync.trust_period": {
			WithDescription("Trust period for light client verification. Should be < unbonding period."),
		},
		"state_sync.use_local_snapshot": {
			WithDescription("Use a local snapshot for state sync instead of discovering from peers."),
		},

		// ---------------------------------------------------------------
		// Storage
		// ---------------------------------------------------------------
		"storage.db_backend": {
			WithDescription("Database backend: goleveldb, cleveldb, boltdb, rocksdb."),
		},
		"storage.db_path": {
			WithDescription("Path to the database directory relative to the home directory."),
		},
		"storage.pruning": {
			WithDescription("Pruning strategy: default, nothing, everything, custom."),
		},
		"storage.pruning_keep_recent": {
			WithDescription("Number of recent states to keep when pruning strategy is 'custom'."),
		},
		"storage.pruning_interval": {
			WithDescription("Block interval between pruning operations when strategy is 'custom'."),
			WithUnit("blocks"),
		},
		"storage.snapshot_interval": {
			WithDescription("Block interval for state sync snapshot creation. 0 disables."),
			WithUnit("blocks"),
		},
		"storage.snapshot_keep_recent": {
			WithDescription("Number of recent snapshots to keep. 0 keeps all."),
		},
		"storage.compaction_interval": {
			WithDescription("Interval between forced LevelDB compaction. 0 disables."),
			WithUnit("seconds"),
		},

		// Storage — State Commit
		"storage.state_commit.enable": {
			WithDescription("Enable SeiDB state-commit layer (replaces IAVL with MemIAVL)."),
		},
		"storage.state_commit.write_mode": {
			WithDescription("EVM write routing: cosmos_only, dual_write, split_write, evm_only."),
		},
		"storage.state_commit.read_mode": {
			WithDescription("EVM read routing: cosmos_only, evm_first, split_read."),
		},

		// Storage — State Store
		"storage.state_store.enable": {
			WithDescription("Enable SeiDB state-store for historical queries."),
		},
		"storage.state_store.backend": {
			WithDescription("State store backend: pebbledb, rocksdb."),
		},
		"storage.state_store.keep_recent": {
			WithDescription("Number of recent versions to keep. 0 keeps all."),
			WithUnit("versions"),
		},
		"storage.state_store.prune_interval_seconds": {
			WithDescription("Interval between state store pruning operations."),
			WithUnit("seconds"),
		},

		// Storage — Receipt Store
		"storage.receipt_store.backend": {
			WithDescription("Receipt store backend: pebbledb, parquet."),
		},
		"storage.receipt_store.db_directory": {
			WithDescription("Receipt store data directory. Empty means use the application home."),
		},
		"storage.receipt_store.async_write_buffer": {
			WithDescription("Async write queue length for the pebbledb receipt store. Set <=0 for synchronous writes."),
		},
		"storage.receipt_store.keep_recent": {
			WithDescription("Receipt versions to retain. 0 keeps all."),
			WithUnit("versions"),
		},
		"storage.receipt_store.prune_interval_seconds": {
			WithDescription("Interval between receipt-store pruning passes."),
			WithUnit("seconds"),
		},
		"storage.receipt_store.tx_index_backend": {
			WithDescription("Tx-hash index backend for the parquet receipt store. Empty disables the index."),
		},

		// ---------------------------------------------------------------
		// Tx Index
		// ---------------------------------------------------------------
		"tx_index.indexer": {
			WithDescription("Transaction indexer backends. Options: null, kv, psql."),
		},

		// ---------------------------------------------------------------
		// EVM
		// ---------------------------------------------------------------
		"evm.http_enabled": {
			WithDescription("Enable the EVM JSON-RPC HTTP server."),
		},
		"evm.http_port": {
			WithDescription("Port for the EVM JSON-RPC HTTP server."),
		},
		"evm.ws_enabled": {
			WithDescription("Enable the EVM JSON-RPC WebSocket server."),
		},
		"evm.ws_port": {
			WithDescription("Port for the EVM JSON-RPC WebSocket server."),
		},
		"evm.simulation_gas_limit": {
			WithDescription("Maximum gas limit for eth_call and eth_estimateGas simulations."),
			WithUnit("gas"),
		},
		"evm.cors_origins": {
			WithDescription("CORS allowed origins for EVM RPC, comma-separated. '*' allows all."),
			WithHotReload(),
		},
		"evm.max_tx_pool_txs": {
			WithDescription("Maximum transactions to pull from the mempool for EVM."),
			WithUnit("transactions"),
		},
		"evm.deny_list": {
			WithDescription("RPC methods that should immediately fail (e.g. debug_traceBlockByNumber)."),
			WithHotReload(),
		},
		"evm.max_log_no_block": {
			WithDescription("Maximum logs returned when block range is open-ended."),
		},
		"evm.max_blocks_for_log": {
			WithDescription("Maximum block range for eth_getLogs queries."),
			WithUnit("blocks"),
		},
		"evm.max_concurrent_trace_calls": {
			WithDescription("Maximum concurrent debug_trace calls. 0 for unlimited."),
		},
		"evm.max_trace_lookback_blocks": {
			WithDescription("Maximum blocks to look back for tracing. -1 for unlimited (archive)."),
			WithUnit("blocks"),
		},
		"evm.trace_timeout": {
			WithDescription("Timeout for each trace call."),
		},
		"evm.worker_pool_size": {
			WithDescription("Number of workers in the EVM RPC worker pool. 0 for default."),
		},

		// ---------------------------------------------------------------
		// API
		// ---------------------------------------------------------------
		"api.rest.enable": {
			WithDescription("Enable the Cosmos REST API server."),
		},
		"api.rest.address": {
			WithDescription("Address for the REST API server."),
		},
		"api.rest.swagger": {
			WithDescription("Enable Swagger documentation endpoint."),
		},
		"api.grpc.enable": {
			WithDescription("Enable the gRPC server."),
		},
		"api.grpc.address": {
			WithDescription("Address for the gRPC server."),
		},
		"api.grpc_web.enable": {
			WithDescription("Enable the gRPC-Web proxy server."),
		},
		"api.grpc_web.address": {
			WithDescription("Address for the gRPC-Web server."),
		},

		// ---------------------------------------------------------------
		// Metrics
		// ---------------------------------------------------------------
		"metrics.enabled": {
			WithDescription("Enable Prometheus metrics collection and serving."),
		},
		"metrics.prometheus_listen_addr": {
			WithDescription("Address for the Prometheus metrics endpoint."),
		},
		"metrics.namespace": {
			WithDescription("Metrics namespace prefix for all emitted metrics."),
		},
		"metrics.prometheus_retention_time": {
			WithDescription("How long to retain Prometheus metrics in memory."),
			WithUnit("seconds"),
		},

		// ---------------------------------------------------------------
		// Logging
		// ---------------------------------------------------------------
		"logging.level": {
			WithDescription("Log level: debug, info, warn, error."),
			WithHotReload(),
		},
		"logging.format": {
			WithDescription("Log format: plain, text, json."),
		},

		// ---------------------------------------------------------------
		// WASM
		// ---------------------------------------------------------------
		"wasm.query_gas_limit": {
			WithDescription("Maximum gas for CosmWasm smart query execution."),
			WithUnit("gas"),
		},
		"wasm.lru_size": {
			WithDescription("Size of the CosmWasm compiled module LRU cache."),
		},

		// ---------------------------------------------------------------
		// Giga Executor
		// ---------------------------------------------------------------
		"giga_executor.enabled": {
			WithDescription("Enable the Giga parallel execution engine."),
		},
		"giga_executor.occ_enabled": {
			WithDescription("Enable OCC within the Giga executor."),
		},

		// ---------------------------------------------------------------
		// Self-Remediation
		// ---------------------------------------------------------------
		"self_remediation.p2p_no_peers_restart_window_seconds": {
			WithDescription("Restart if no P2P peers available for this duration. 0 disables."),
			WithUnit("seconds"),
		},
		"self_remediation.blocks_behind_threshold": {
			WithDescription("Restart if node falls this many blocks behind. 0 disables."),
			WithUnit("blocks"),
		},
		"self_remediation.restart_cooldown_seconds": {
			WithDescription("Minimum time between self-remediation restarts."),
			WithUnit("seconds"),
		},
	}
}
