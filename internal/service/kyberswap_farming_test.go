package service

import (
	"context"
	"data/internal/model"
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
			t.Fatalf("%s high_apr returned 0 pools", chain.Label)
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

func TestDetectEarnFeeSurges(t *testing.T) {
	history := map[string]EarnFeeHistory{
		"same":   {Value: 100},
		"surge":  {Value: 100},
		"small":  {Value: 100},
		"lowfee": {Value: 10},
		"zero":   {Value: 0},
	}
	pools := []model.Pool{
		{ID: "same", Fees24h: 104},
		{ID: "surge", Fees24h: 106},
		{ID: "small", Fees24h: 101},
		{ID: "lowfee", Fees24h: 15},
		{ID: "zero", Fees24h: 50},
		{ID: "new", Fees24h: 80},
	}
	got, notifyHistory, updates := DetectEarnFeeSurges(pools, history)
	if len(got) != 1 || got[0].ID != "surge" {
		t.Fatalf("toNotify=%v", got)
	}
	if notifyHistory["surge"].Value != 100 {
		t.Fatalf("history=%v", notifyHistory)
	}
	if updates["surge"] != 106 || updates["new"] != 80 {
		t.Fatalf("updates=%v", updates)
	}
}

func TestFarmingEarnFeeHistoryIsolation(t *testing.T) {
	dir := t.TempDir()
	farmingEarnFeeHistoryMu.Lock()
	old := farmingEarnFeeHistoryDir
	farmingEarnFeeHistoryDir = dir
	farmingEarnFeeHistoryMu.Unlock()
	t.Cleanup(func() {
		farmingEarnFeeHistoryMu.Lock()
		farmingEarnFeeHistoryDir = old
		farmingEarnFeeHistoryMu.Unlock()
	})

	s := &kyberSwapImpl{}
	ctx := context.Background()
	if err := s.UpdateFarmingEarnFeeHistories(ctx, 1, map[string]float64{"eth-pool": 10}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateFarmingEarnFeeHistories(ctx, 56, map[string]float64{"bsc-pool": 20}); err != nil {
		t.Fatal(err)
	}
	eth, err := s.GetFarmingEarnFeeHistoryWithTime(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	bsc, err := s.GetFarmingEarnFeeHistoryWithTime(ctx, 56)
	if err != nil {
		t.Fatal(err)
	}
	if eth["eth-pool"].Value != 10 || eth["bsc-pool"].Value != 0 {
		t.Fatalf("eth=%v", eth)
	}
	if bsc["bsc-pool"].Value != 20 || bsc["eth-pool"].Value != 0 {
		t.Fatalf("bsc=%v", bsc)
	}
}
