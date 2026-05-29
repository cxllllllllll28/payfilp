package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	ctx := context.Background()
	// 1. 拉全链收益数据
	yields, err := c.yield.FetchMantleYields(1_000_000) // TVL >= $1M
	if err != nil {
		return nil, "", fmt.Errorf("fetch yields: %w", err)
	}

	// 2. 通过 RPC 查用户在几个关键协议的余额
	positions := c.discoverPositions(ctx, addr, yields)

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

// balanceOfSelector = keccak256("balanceOf(address)")[:4]
var balanceOfSelector = crypto.Keccak256([]byte("balanceOf(address)"))[:4]

// positionChecklist 定义已知的 protocol -> token 映射
var positionChecklist = []struct {
	Protocol   string
	Symbol     string
	TokenAddr  common.Address
	Decimals   int64
	CurrentAPY float64
}{
	{Protocol: "Aave V3", Symbol: "USDT", TokenAddr: common.HexToAddress("0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE"), Decimals: 6, CurrentAPY: 0},
	{Protocol: "Ondo USDY", Symbol: "USDY", TokenAddr: common.HexToAddress("0x8c82B0bD9613b0C6CdED0aE8C1a06191E87F0aF4"), Decimals: 18, CurrentAPY: 0},
}

func (c *YieldChecker) queryBalance(ctx context.Context, tokenAddr common.Address, userAddr common.Address, decimals int64) (float64, error) {
	data := append(balanceOfSelector, common.LeftPadBytes(userAddr.Bytes(), 32)...)
	result, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("call balanceOf: %w", err)
	}
	balance := new(big.Int).SetBytes(result)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(decimals), nil)
	f := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(divisor))
	human, _ := f.Float64()
	return human, nil
}

func (c *YieldChecker) discoverPositions(ctx context.Context, addr common.Address, yields []YieldInfo) []WalletPosition {
	var positions []WalletPosition
	for _, check := range positionChecklist {
		apy := check.CurrentAPY
		for _, y := range yields {
			if strings.EqualFold(y.Protocol, check.Protocol) && strings.EqualFold(y.Symbol, check.Symbol) {
				apy = y.APY
				break
			}
		}
		balance, err := c.queryBalance(ctx, check.TokenAddr, addr, check.Decimals)
		if err != nil {
			log.Printf("查询 %s 余额失败: %v", check.Protocol, err)
			continue
		}
		positions = append(positions, WalletPosition{
			Protocol:     check.Protocol,
			Symbol:       check.Symbol,
			TokenAddress: check.TokenAddr.Hex(),
			BalanceHuman: balance,
			CurrentAPY:   apy,
		})
	}
	if len(positions) == 0 {
		return []WalletPosition{
			{Protocol: "Aave V3", Symbol: "USDT", TokenAddress: "0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE", BalanceHuman: 0, CurrentAPY: 0},
		}
	}
	return positions
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
