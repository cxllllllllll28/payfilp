package services

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/yourusername/payflip-backend/internal/tx"
)

// IntentExecutor 意图执行器
type IntentExecutor struct {
	txmgr   *tx.TxManager
	builder *tx.Builder
	sender  *tx.Sender
}

func NewIntentExecutor(txmgr *tx.TxManager, rpcURL string, chainID int64, txBuilder *tx.Builder) *IntentExecutor {
	return &IntentExecutor{
		txmgr:   txmgr,
		builder: txBuilder,
		sender:  tx.NewSender(txmgr, rpcURL, chainID),
	}
}

// ExecuteCalldata 统一执行入口，接受 targets/datas/values，打包一个多步交易发送
func (e *IntentExecutor) ExecuteCalldata(ctx context.Context, targets []common.Address, values []*big.Int, datas [][]byte) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("empty calldata")
	}
	// 把多个 calldata 塞进一个多投交易
	tx, err := e.txmgr.BuildBatchTx(ctx, targets, values, datas)
	if err != nil {
		return "", fmt.Errorf("build batch tx: %w", err)
	}
	hash, err := e.sender.Send(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("send tx: %w", err)
	}
	return hash.Hex(), nil
}
