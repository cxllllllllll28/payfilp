package services

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

// IntentExecutor 意图执行器 — 调 tx/ 子系统执行链上交易
type IntentExecutor struct {
	txmgr   *tx.TxManager
	builder *tx.Builder
	sender  *tx.Sender
}

// NewIntentExecutor 创建意图执行器
func NewIntentExecutor(txmgr *tx.TxManager) *IntentExecutor {
	return &IntentExecutor{
		txmgr:   txmgr,
		builder: tx.NewBuilder(txmgr),
		sender:  tx.NewSender(txmgr),
	}
}

// Execute 执行意图
func (e *IntentExecutor) Execute(ctx context.Context, intent *IntentResult) (string, error) {
	switch intent.Action {
	case "swap":
		return e.executeSwap(ctx, intent)
	case "swap_and_stake":
		return e.executeSwapAndStake(ctx, intent)
	default:
		return "", fmt.Errorf("unknown action: %s", intent.Action)
	}
}

func (e *IntentExecutor) executeSwap(ctx context.Context, intent *IntentResult) (string, error) {
	fromAddr := common.HexToAddress(tokenAddress(intent.FromToken))
	toAddr := common.HexToAddress(tokenAddress(intent.ToToken))
	amount := new(big.Int)
	amount.SetString(intent.Amount, 10)
	amount.Mul(amount, big.NewInt(1e6))

	calldata, _, err := e.builder.BuildSwapCalldata(ctx, fromAddr, toAddr, amount)
	if err != nil {
		return "", fmt.Errorf("build swap calldata: %w", err)
	}

	tx, err := e.txmgr.BuildTx(ctx, &toAddr, big.NewInt(0), calldata, 0, nil, nil)
	if err != nil {
		return "", fmt.Errorf("build tx: %w", err)
	}

	hash, err := e.sender.Send(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("send tx: %w", err)
	}
	return hash.Hex(), nil
}

func (e *IntentExecutor) executeSwapAndStake(ctx context.Context, intent *IntentResult) (string, error) {
	return "", fmt.Errorf("swap_and_stake not yet implemented")
}

func tokenAddress(symbol string) string {
	m := map[string]string{
		"USDT": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		"USDC": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		"MNT":  "0x3c3a81e81dc49A522A592e7622A7E711c06bf354",
	}
	return m[symbol]
}
