package seiconfig

import (
	"encoding/json"
	"testing"
)

func TestGenesisForChain_KnownChains(t *testing.T) {
	for _, chainID := range KnownChainIDs() {
		data, err := GenesisForChain(chainID)
		if err != nil {
			t.Fatalf("GenesisForChain(%q): %v", chainID, err)
		}
		if len(data) == 0 {
			t.Fatalf("GenesisForChain(%q) returned empty data", chainID)
		}

		var doc struct {
			ChainID     string `json:"chain_id"`
			GenesisTime string `json:"genesis_time"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("GenesisForChain(%q): invalid JSON: %v", chainID, err)
		}
		if doc.ChainID != chainID {
			t.Errorf("GenesisForChain(%q): chain_id mismatch: got %q", chainID, doc.ChainID)
		}

		info := KnownChain(chainID)
		if info == nil {
			t.Fatalf("KnownChain(%q) returned nil", chainID)
		}
		if doc.GenesisTime != info.GenesisTime {
			t.Errorf("GenesisForChain(%q): genesis_time=%q, ChainInfo says %q",
				chainID, doc.GenesisTime, info.GenesisTime)
		}
	}
}

func TestGenesisForChain_UnknownChain(t *testing.T) {
	_, err := GenesisForChain("nonexistent-99")
	if err == nil {
		t.Fatal("expected error for unknown chain")
	}
}

func TestKnownChain_Nil(t *testing.T) {
	if info := KnownChain("nonexistent-99"); info != nil {
		t.Fatalf("expected nil, got %+v", info)
	}
}

func TestKnownChainIDs(t *testing.T) {
	ids := KnownChainIDs()
	if len(ids) == 0 {
		t.Fatal("expected at least one known chain")
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate chain ID: %s", id)
		}
		seen[id] = true
	}
}

func TestKnownChain_HasRPC(t *testing.T) {
	for _, chainID := range KnownChainIDs() {
		info := KnownChain(chainID)
		if info.RPC == "" {
			t.Errorf("KnownChain(%q): RPC is empty", chainID)
		}
	}
}
