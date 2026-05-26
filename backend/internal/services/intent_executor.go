package services

import (
	"context"
	"fmt"
	"math/big"

	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

// IntentExecutor 意图执行器
type IntentExecutor struct {
	txmgr   *tx.TxManager
	builder *tx.Builder
	sender  *tx.Sender
}

func NewIntentExecutor(txmgr *tx.TxManager, rpcURL string, chainID int64) *IntentExecutor {
	return &IntentExecutor{
		txmgr:   txmgr,
		builder: tx.NewBuilder(txmgr),
		sender:  tx.NewSender(txmgr, rpcURL, chainID),
	}
}

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
	fromAddr := tx.TokenAddr(intent.FromToken)
	toAddr := tx.TokenAddr(intent.ToToken)
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
