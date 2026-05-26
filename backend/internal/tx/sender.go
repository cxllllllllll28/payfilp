package tx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sender 交易发送器 — 自动选择 normal / gasless 路径
type Sender struct {
	mgr *TxManager
}

// NewSender 创建发送器
func NewSender(mgr *TxManager) *Sender {
	return &Sender{mgr: mgr}
}

// Send 发送交易
func (s *Sender) Send(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	signedTx, err := s.mgr.SignTx(ctx, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("sign: %w", err)
	}
	err = s.mgr.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("send: %w", err)
	}
	return signedTx.Hash(), nil
}

// SendGasless 发送 Gasless 交易（后续集成 eth/ 包）
func (s *Sender) SendGasless(ctx context.Context, to common.Address, value *big.Int, data []byte) (common.Hash, error) {
	return common.Hash{}, fmt.Errorf("gasless path not yet integrated")
}
