package service

import (
	"context"
	"os"
	"testing"
)

func TestFarmingSentPoolIDsIsolation(t *testing.T) {
	s := &kyberSwapImpl{}
	ctx := context.Background()
	chainA := 99991
	chainB := 99992
	defer os.Remove(farmingSentFilePath(chainA))
	defer os.Remove(farmingSentFilePath(chainB))

	if err := s.AddSentFarmingPoolIDs(ctx, chainA, []string{"0xaaa"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddSentFarmingPoolIDs(ctx, chainB, []string{"0xbbb"}); err != nil {
		t.Fatal(err)
	}

	a, err := s.GetTodaySentFarmingPoolIDs(ctx, chainA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.GetTodaySentFarmingPoolIDs(ctx, chainB)
	if err != nil {
		t.Fatal(err)
	}
	if !a["0xaaa"] || a["0xbbb"] {
		t.Fatalf("chain A map=%v", a)
	}
	if !b["0xbbb"] || b["0xaaa"] {
		t.Fatalf("chain B map=%v", b)
	}
}

func TestFetchFarmingPoolsByChainLive(t *testing.T) {
	s := &kyberSwapImpl{}
	ctx := context.Background()
	for _, chain := range FarmingChains() {
		pools, err := s.FetchFarmingPoolsByChain(ctx, chain.ID)
		if err != nil {
			t.Skipf("network fetch %s failed: %v", chain.Label, err)
		}
		if len(pools) == 0 {
			t.Fatalf("%s farming_pool returned 0 pools", chain.Label)
		}
		for _, p := range pools {
			if p.ID == "" {
				t.Fatalf("%s pool missing ID", chain.Label)
			}
			if p.ChainID != 0 && p.ChainID != chain.ID {
				t.Fatalf("%s pool chainID=%d", chain.Label, p.ChainID)
			}
		}
	}
}
