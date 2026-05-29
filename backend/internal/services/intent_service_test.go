package services

import (
	"os"
	"testing"
	"strings"

	"github.com/joho/godotenv"

	"github.com/yourusername/hacker-mantle-backend/config"
)

func init() { _ = godotenv.Load("../../.env") }

func TestBuildPlan_Swap(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" { t.Skip("DEEPSEEK_API_KEY not set") }
	registry, _ := config.ParseProtocolRegistry([]byte(`{"protocols":[]}`))
	plan, err := NewIntentService(nil, registry).BuildPlan("用 1 USDT 换 MNT")
	if err != nil { t.Fatalf("BuildPlan: %v", err) }
	if len(plan.Steps) < 1 { t.Errorf("expected >=1 steps, got %d", len(plan.Steps)) }
	// Deepseek 可能会自动补 approve，只要包含 swap 就算成功
	hasSwap := false
	for _, s := range plan.Steps {
		if s.Action == "swap" { hasSwap = true; break }
	}
	if !hasSwap { t.Errorf("expected at least one swap step") }
}

func TestBuildPlan_SwapAndStake(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" { t.Skip("DEEPSEEK_API_KEY not set") }
	registry, _ := config.ParseProtocolRegistry([]byte(`{"protocols":[]}`))
	plan, err := NewIntentService(nil, registry).BuildPlan("用 200 USDT 换成 MNT 并质押")
	if err != nil { t.Fatalf("BuildPlan: %v", err) }
	if len(plan.Steps) < 2 { t.Errorf("expected >=2 steps, got %d", len(plan.Steps)) }
	for _, s := range plan.Steps {
		if !strings.Contains("approve swap stake unstake", s.Action) {
			t.Errorf("unexpected action: %s", s.Action)
		}
	}
}
