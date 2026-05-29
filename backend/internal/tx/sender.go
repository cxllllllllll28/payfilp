package tx

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sender 交易发送器
type Sender struct {
	mgr     *TxManager
	rpcURL  string
	chainID int64
}

func NewSender(mgr *TxManager, rpcURL string, chainID int64) *Sender {
	return &Sender{mgr: mgr, rpcURL: rpcURL, chainID: chainID}
}

// Send 发送交易（当前仅 normal 路径：签名 → eth_sendRawTransaction）
func (s *Sender) Send(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	return s.sendNormal(ctx, tx)
}

func (s *Sender) sendNormal(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
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
