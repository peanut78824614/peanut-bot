package service

import (
	"fmt"
	"testing"
)

func TestEarnServicePoolURLs_PerChain(t *testing.T) {
	want := map[int]string{
		4663: "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=4663&page=1&limit=100&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q=",
		8453: "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=8453&page=1&limit=100&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q=",
		56:   "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=56&page=1&limit=100&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q=",
	}
	if len(earnServiceChainIDs) != 3 {
		t.Fatalf("expected 3 chain IDs, got %v", earnServiceChainIDs)
	}
	for _, chainID := range earnServiceChainIDs {
		got := fmt.Sprintf(earnServicePoolsURL, chainID, 1)
		if got != want[chainID] {
			t.Fatalf("chain %d URL mismatch\n got: %s\nwant: %s", chainID, got, want[chainID])
		}
	}
}

func TestFarmingServicePoolURLs_PerChain(t *testing.T) {
	want := map[int]string{
		1:    "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=1&page=1&limit=100&interval=24h&protocol=&tag=farming_pool&sortBy=&orderBy=&q=",
		8453: "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=8453&page=1&limit=100&interval=24h&protocol=&tag=farming_pool&sortBy=&orderBy=&q=",
		56:   "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=56&page=1&limit=100&interval=24h&protocol=&tag=farming_pool&sortBy=&orderBy=&q=",
		4663: "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=4663&page=1&limit=100&interval=24h&protocol=&tag=farming_pool&sortBy=&orderBy=&q=",
	}
	chains := FarmingChains()
	if len(chains) != 4 {
		t.Fatalf("expected 4 farming chains, got %v", chains)
	}
	seen := map[string]bool{}
	for _, chain := range chains {
		got := farmingServiceURL(chain.ID, 1)
		if got != want[chain.ID] {
			t.Fatalf("chain %d URL mismatch\n got: %s\nwant: %s", chain.ID, got, want[chain.ID])
		}
		if chain.Key == "" {
			t.Fatalf("chain %d missing config key", chain.ID)
		}
		if seen[chain.Key] {
			t.Fatalf("duplicate farming key %s", chain.Key)
		}
		seen[chain.Key] = true
		delete(want, chain.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing farming chains: %v", want)
	}
}
