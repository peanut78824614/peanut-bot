package service

import (
	"testing"
)

func TestIsStableSymbol(t *testing.T) {
	cases := map[string]bool{
		"USDT":   true,
		"usdc":   true,
		"USDG":   true,
		" usdg ": true,
		"WETH":   false,
		"CA":     false,
	}
	for sym, want := range cases {
		if got := isStableSymbol(sym); got != want {
			t.Fatalf("isStableSymbol(%q)=%v want %v", sym, got, want)
		}
	}
}

func TestParsePoolFromInterface_RobinhoodKeepsAll(t *testing.T) {
	s := &kyberSwapImpl{}
	raw := map[string]interface{}{
		"address":  "0xpoolusdg",
		"apr":      12.3,
		"tvl":      1000.0,
		"volume":   200.0,
		"earnFee":  30.0,
		"feeTier":  0.3,
		"exchange": "uniswap-v4",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xusdg", "symbol": "USDG"},
			map[string]interface{}{"address": "0xca", "symbol": "CA"},
		},
		"chain": map[string]interface{}{"id": 4663.0, "name": "robinhood"},
	}
	pool := s.parsePoolFromInterface(raw, false)
	if pool == nil {
		t.Fatal("USDG pool should be kept")
	}
	if pool.ContractAddress != "0xca" {
		t.Fatalf("contract=%s", pool.ContractAddress)
	}

	wethPool := map[string]interface{}{
		"address": "0xpoolweth",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xweth", "symbol": "WETH"},
			map[string]interface{}{"address": "0xusdg", "symbol": "USDG"},
		},
		"chain": map[string]interface{}{"id": 4663.0, "name": "robinhood"},
	}
	got := s.parsePoolFromInterface(wethPool, false)
	if got == nil {
		t.Fatal("Robinhood WETH pool should be pushed")
	}
	if got.ContractAddress != "0xweth" {
		t.Fatalf("contract=%s", got.ContractAddress)
	}

	noStable := map[string]interface{}{
		"address": "0xpoolnone",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xa", "symbol": "PUNKA"},
			map[string]interface{}{"address": "0xb", "symbol": "HIMS"},
		},
		"chain": map[string]interface{}{"id": 4663.0, "name": "robinhood"},
	}
	if s.parsePoolFromInterface(noStable, false) == nil {
		t.Fatal("Robinhood pool without stablecoin should be pushed")
	}
}

func TestParsePoolFromInterface_BaseBSCStillFiltered(t *testing.T) {
	s := &kyberSwapImpl{}
	wethPool := map[string]interface{}{
		"address": "0xpoolweth",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xweth", "symbol": "WETH"},
			map[string]interface{}{"address": "0xusdc", "symbol": "USDC"},
		},
		"chain": map[string]interface{}{"id": 8453.0, "name": "base"},
	}
	if s.parsePoolFromInterface(wethPool, false) != nil {
		t.Fatal("Base WETH pool should still be filtered")
	}

	noStable := map[string]interface{}{
		"address": "0xpoolnone",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xa", "symbol": "PUNKA"},
			map[string]interface{}{"address": "0xb", "symbol": "HIMS"},
		},
		"chain": map[string]interface{}{"id": 56.0, "name": "bsc"},
	}
	if s.parsePoolFromInterface(noStable, false) != nil {
		t.Fatal("BSC pool without USDT/USDC/USDG should still be filtered")
	}
}

func TestParsePoolFromInterface_KeepAllNoFilter(t *testing.T) {
	s := &kyberSwapImpl{}
	wethPool := map[string]interface{}{
		"address": "0xpoolweth",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xweth", "symbol": "WETH"},
			map[string]interface{}{"address": "0xusdc", "symbol": "USDC"},
		},
		"chain": map[string]interface{}{"id": 1.0, "name": "ethereum"},
	}
	got := s.parsePoolFromInterface(wethPool, true)
	if got == nil {
		t.Fatal("ETH WETH pool should be kept when keepAll=true")
	}
	if got.ChainID != 1 {
		t.Fatalf("chainID=%d", got.ChainID)
	}

	noStable := map[string]interface{}{
		"address": "0xpoolnone",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xa", "symbol": "PUNKA"},
			map[string]interface{}{"address": "0xb", "symbol": "HIMS"},
		},
		"chain": map[string]interface{}{"id": 8453.0, "name": "base"},
	}
	if s.parsePoolFromInterface(noStable, true) == nil {
		t.Fatal("Base pool without stablecoin should be kept when keepAll=true")
	}

	bscWeth := map[string]interface{}{
		"address": "0xpoolbsc",
		"tokens": []interface{}{
			map[string]interface{}{"address": "0xweth", "symbol": "WETH"},
			map[string]interface{}{"address": "0xbnb", "symbol": "WBNB"},
		},
		"chain": map[string]interface{}{"id": 56.0, "name": "bsc"},
	}
	if s.parsePoolFromInterface(bscWeth, true) == nil {
		t.Fatal("BSC WETH pool should be kept when keepAll=true")
	}
}
