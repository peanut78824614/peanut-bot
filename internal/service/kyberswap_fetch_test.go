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
