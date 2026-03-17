package seiconfig

import (
	"embed"
	"fmt"
	"path"
)

//go:embed chains/*/genesis.json
var chainFS embed.FS

// ChainInfo describes a well-known Sei network whose genesis is embedded in
// this binary. Callers that need a chain not listed here must supply their own
// genesis configuration.
type ChainInfo struct {
	ChainID     string
	RPC         string
	GenesisTime string
}

// knownChains is the authoritative set of chains with embedded genesis data.
var knownChains = map[string]ChainInfo{
	"pacific-1": {
		ChainID:     "pacific-1",
		RPC:         "https://rpc.sei-apis.com",
		GenesisTime: "2023-05-22T15:00:00Z",
	},
	"atlantic-2": {
		ChainID:     "atlantic-2",
		RPC:         "https://rpc-testnet.sei-apis.com",
		GenesisTime: "2023-02-24T01:00:00Z",
	},
	"arctic-1": {
		ChainID:     "arctic-1",
		RPC:         "https://rpc-arctic-1.sei-apis.com",
		GenesisTime: "2024-01-25T20:18:30.242526108Z",
	},
}

// KnownChain returns metadata for a well-known chain, or nil if the chain ID
// is not recognised.
func KnownChain(chainID string) *ChainInfo {
	info, ok := knownChains[chainID]
	if !ok {
		return nil
	}
	return &info
}

// KnownChainIDs returns the set of chain IDs that have embedded genesis data.
func KnownChainIDs() []string {
	ids := make([]string, 0, len(knownChains))
	for id := range knownChains {
		ids = append(ids, id)
	}
	return ids
}

// GenesisForChain returns the embedded genesis.json bytes for a well-known
// chain. Returns an error if the chain ID is not recognised — the caller must
// provide their own genesis source for unknown chains.
func GenesisForChain(chainID string) ([]byte, error) {
	if _, ok := knownChains[chainID]; !ok {
		return nil, fmt.Errorf("unknown chain %q: provide a custom genesis source", chainID)
	}
	return chainFS.ReadFile(path.Join("chains", chainID, "genesis.json"))
}
