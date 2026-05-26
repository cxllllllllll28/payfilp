package services

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	_ = godotenv.Load("../../.env")
}

func TestParseIntent_BuyMNT(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" { t.Skip("DEEPSEEK_API_KEY not set") }
	svc := NewIntentService(nil)
	result, err := svc.Parse("帮我把 100 USDT 换成 MNT")
	assert.NoError(t, err)
	assert.Equal(t, "swap", result.Action)
	assert.Equal(t, "USDT", result.FromToken)
	assert.Equal(t, "MNT", result.ToToken)
	assert.Equal(t, "100", result.Amount)
}

func TestParseIntent_BuyMNTAndStake(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" { t.Skip("DEEPSEEK_API_KEY not set") }
	svc := NewIntentService(nil)
	result, err := svc.Parse("用 200 USDT 换成 MNT 并质押生息")
	assert.NoError(t, err)
	assert.Equal(t, "swap_and_stake", result.Action)
}

func TestIntentService_BuildPlan(t *testing.T) {
	if os.Getenv("DEEPSEEK_API_KEY") == "" { t.Skip("DEEPSEEK_API_KEY not set") }
	svc := NewIntentService(nil)
	plan, err := svc.BuildPlan("帮我把 100 USDT 换成 MNT 并质押")
	assert.NoError(t, err)
	if len(plan.Steps) < 2 {
		t.Errorf("expected at least 2 steps, got %d", len(plan.Steps))
	}
}
