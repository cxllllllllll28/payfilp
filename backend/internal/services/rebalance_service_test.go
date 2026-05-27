package services

import (
	"testing"
)

func TestBestRecommendation(t *testing.T) {
	yields := []YieldInfo{
		{Protocol: "aave-v3", Symbol: "USDT0", APY: 5.64, TVLUsd: 23_500_000},
		{Protocol: "ondo", Symbol: "USDY", APY: 3.55, TVLUsd: 29_400_000},
		{Protocol: "aave-v3", Symbol: "WETH", APY: 2.69, TVLUsd: 7_300_000},
		{Protocol: "fluxion", Symbol: "OPG-USDT0", APY: 42.59, TVLUsd: 450_000},
	}
	best := BestRecommendation(yields)
	if best == nil {
		t.Fatal("expected best, got nil")
	}
	if best.APY != 42.59 {
		t.Errorf("expected 42.59, got %.2f", best.APY)
	}
}

func TestBestRecommendation_Empty(t *testing.T) {
	best := BestRecommendation(nil)
	if best != nil {
		t.Errorf("expected nil, got %v", best)
	}
}
