package services

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// WalletPosition 某个协议中的用户持仓（按收益机会排序）
type WalletPosition struct {
	Protocol     string  `json:"protocol"`
	Symbol       string  `json:"symbol"`
	TokenAddress string  `json:"tokenAddress"`
	BalanceWei   string  `json:"balanceWei"`
	BalanceHuman float64 `json:"balanceHuman"`
	CurrentAPY   float64 `json:"currentApy"`
	BetterAPY    float64 `json:"betterApy,omitempty"`
	BetterPool   string  `json:"betterPool,omitempty"`
}

// CheckResult 收益检查结果
type CheckResult struct {
	Address   string            `json:"address"`
	Positions []WalletPosition  `json:"positions"`
	Summary   string            `json:"summary"`
}

// YieldChecker 收益检查器 — 给定钱包地址 + 全链数据，给出调仓建议
type YieldChecker struct {
	client *ethclient.Client
	yield  *YieldService
}

// NewYieldChecker 创建收益检查器
func NewYieldChecker(client *ethclient.Client) *YieldChecker {
	return &YieldChecker{client: client, yield: NewYieldService()}
}

// CheckAddress 检查一个 Mantle 钱包地址的收益状态
func (c *YieldChecker) CheckAddress(addr common.Address) (*CheckResult, string, error) {
	// 1. 拉全链收益数据
	yields, err := c.yield.FetchMantleYields(1_000_000) // TVL >= $1M
	if err != nil {
		return nil, "", fmt.Errorf("fetch yields: %w", err)
	}

	// 2. 查用户在几个关键协议的余额（通过 RPC 查合约余额）
	// 固定检查 Aave USDT、Ondo USDY
	positions := c.discoverPositions(addr)

	// 3. 找出更好的收益池
	bestYield := 0.0
	bestPool := ""
	for _, y := range yields {
		if y.APY > bestYield {
			bestYield = y.APY
			bestPool = fmt.Sprintf("%s %s", y.Protocol, y.Symbol)
		}
	}

	for i, pos := range positions {
		for _, y := range yields {
			if y.APY > pos.CurrentAPY*1.2 && y.APY > positions[i].BetterAPY {
				positions[i].BetterAPY = y.APY
				positions[i].BetterPool = fmt.Sprintf("%s %s (%.2f%%)", y.Protocol, y.Symbol, y.APY)
			}
		}
	}

	// 构建总结
	summary := "✅ 当前收益不错"
	hasBetter := false
	for _, p := range positions {
		if p.BetterPool != "" {
			hasBetter = true
		}
	}
	if hasBetter {
		summary = fmt.Sprintf("💡 发现更好机会！当前全链最佳收益为 %.2f%%（%s）", bestYield, bestPool)
	}

	return &CheckResult{
		Address:   addr.Hex(),
		Positions: positions,
		Summary:   summary,
	}, bestPool, nil
}

func (c *YieldChecker) discoverPositions(addr common.Address) []WalletPosition {
	// 模拟数据 — 后续通过 RPC 查真实余额
	return []WalletPosition{
		{Protocol: "Aave V3", Symbol: "USDT", TokenAddress: "0x...", BalanceHuman: 10000, CurrentAPY: 5.94},
		{Protocol: "Ondo USDY", Symbol: "USDY", TokenAddress: "0x...", BalanceHuman: 0, CurrentAPY: 3.55},
	}
}

// BestRecommendation 从全链数据中找出最佳收益池
func BestRecommendation(yields []YieldInfo) *YieldInfo {
	if len(yields) == 0 { return nil }
	best := yields[0]
	for _, y := range yields {
		if y.APY > best.APY { best = y }
	}
	return &best
}
