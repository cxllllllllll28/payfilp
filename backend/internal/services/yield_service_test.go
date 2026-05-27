package services

import (
	"testing"
)

func TestFetchMantleYields(t *testing.T) {
	svc := NewYieldService()
	yields, err := svc.FetchMantleYields(1_000_000) // TVL >= $1M
	if err != nil {
		t.Fatalf("FetchMantleYields: %v", err)
	}
	if len(yields) == 0 {
		t.Fatal("expected at least 1 pool, got 0")
	}

	// 打印 TVL 前 5
	for i, p := range yields {
		if i >= 5 {
			break
		}
		t.Logf("[%d] %-20s %-10s APY=%5.2f%% TVL=$%.2fM", i+1, p.Protocol, p.Symbol, p.APY, p.TVLUsd/1e6)
	}
}
