package tx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// SwapRoute DEX 兑换路由信息
type SwapRoute struct {
	Router     common.Address
	Path       []common.Address
	AmountOut  *big.Int
}

// Builder calldata 构建器
type Builder struct {
	mgr *TxManager
}

// NewBuilder 创建 calldata 构建器
func NewBuilder(mgr *TxManager) *Builder {
	return &Builder{mgr: mgr}
}

// BuildSwapCalldata 构造兑换 calldata（占位 — 后续填入 DEX ABI）
func (b *Builder) BuildSwapCalldata(ctx context.Context, fromToken, toToken common.Address, amountIn *big.Int) ([]byte, *SwapRoute, error) {
	// TODO: 查询 DEX 路由（Merchant Moe / Agni）并构造 swapExactTokensForTokens calldata
	return nil, nil, fmt.Errorf("not implemented: DEX router integration pending")
}
