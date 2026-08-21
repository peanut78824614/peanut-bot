package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func withTempHistoryFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pool_earn_fee_history.json")
	earnFeeHistoryMu.Lock()
	old := earnFeeHistoryFilePath
	earnFeeHistoryFilePath = path
	earnFeeHistoryMu.Unlock()
	t.Cleanup(func() {
		earnFeeHistoryMu.Lock()
		earnFeeHistoryFilePath = old
		earnFeeHistoryMu.Unlock()
	})
	return path
}

func TestParseEarnFeeHistory_TrailingQuoteAfterTopLevel(t *testing.T) {
	ts := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	valid, err := json.Marshal(map[string]EarnFeeHistory{
		"0xabc": {Value: 23.07, Timestamp: ts},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := append(append([]byte{}, valid...), []byte(`"timestamp": "2026-08-16T12:00:00Z"`)...)

	var dumped map[string]EarnFeeHistory
	err = json.Unmarshal(raw, &dumped)
	if err == nil {
		t.Fatal("expected json.Unmarshal to fail on trailing quote")
	}
	if !strings.Contains(err.Error(), `invalid character '"' after top-level value`) {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	history, trailing, err := parseEarnFeeHistoryContent(raw)
	if err != nil {
		t.Fatalf("parse should recover first JSON object, got %v", err)
	}
	if !trailing {
		t.Fatal("expected trailing=true")
	}
	got, ok := history["0xabc"]
	if !ok {
		t.Fatalf("missing recovered pool, history=%v", history)
	}
	if got.Value != 23.07 {
		t.Fatalf("value=%v", got.Value)
	}
	if !got.Timestamp.Equal(ts) {
		t.Fatalf("timestamp=%v", got.Timestamp)
	}
}

func TestParseEarnFeeHistory_OldFormatAndConcatenatedObjects(t *testing.T) {
	raw := []byte(`{"0xold": 12.5}{"0xnew": 9}`)
	history, trailing, err := parseEarnFeeHistoryContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !trailing {
		t.Fatal("expected trailing")
	}
	if history["0xold"].Value != 12.5 {
		t.Fatalf("got %+v", history)
	}
}

func TestGetPoolEarnFeeHistoryWithTime_RecoversCorruptFile(t *testing.T) {
	path := withTempHistoryFile(t)
	ts := time.Date(2026, 8, 16, 12, 1, 14, 0, time.UTC)
	valid, _ := json.MarshalIndent(map[string]EarnFeeHistory{
		"pool-1": {Value: 100, Timestamp: ts},
	}, "", "  ")
	if err := os.WriteFile(path, append(valid, []byte(`"timestamp": "leftover"`)...), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	history, err := KyberSwap().GetPoolEarnFeeHistoryWithTime(ctx)
	if err != nil {
		t.Fatalf("must not return parse error to job: %v", err)
	}
	if history["pool-1"].Value != 100 {
		t.Fatalf("history=%v", history)
	}

	clean, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var check map[string]EarnFeeHistory
	if err := json.Unmarshal(clean, &check); err != nil {
		t.Fatalf("rewritten file still invalid: %v\n%s", err, clean)
	}
}

func TestUpdatePoolEarnFeeHistories_BatchAndConcurrent(t *testing.T) {
	withTempHistoryFile(t)
	ctx := context.Background()
	s := KyberSwap()

	if err := s.UpdatePoolEarnFeeHistories(ctx, map[string]float64{
		"a": 1,
		"b": 2,
	}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.UpdatePoolEarnFeeHistory(ctx, fmt.Sprintf("p%d", i%8), float64(i)); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	history, err := s.GetPoolEarnFeeHistoryWithTime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(earnFeeHistoryFilePath)
	if err != nil {
		t.Fatal(err)
	}
	var check map[string]EarnFeeHistory
	if err := json.Unmarshal(content, &check); err != nil {
		t.Fatalf("concurrent writes corrupted file: %v\n%s", err, content)
	}
	if len(history) < 8 {
		t.Fatalf("expected at least 8 keys, got %d", len(history))
	}
}
