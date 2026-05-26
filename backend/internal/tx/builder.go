package tx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TokenAddr 代币符号 → Mantle 主网地址
func TokenAddr(symbol string) common.Address {
	m := map[string]common.Address{
		"USDT": common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		"USDC": common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		"MNT":  common.HexToAddress("0x3c3a81e81dc49A522A592e7622A7E711c06bf354"),
	}
	return m[symbol]
}

// DEX 路由表
var dexRouters = map[string]common.Address{
	"MerchantMoe": common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"),
	"Agni":        common.HexToAddress(""), // TBD
}

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

// BuildSwapCalldata 构造 Uniswap V2 swapExactTokensForTokens 的 calldata
//
// 调用 ABI: swapExactTokensForTokens(uint256 amountIn, uint256 amountOutMin,
//                                    address[] path, address to, uint256 deadline)
func (b *Builder) BuildSwapCalldata(ctx context.Context, fromToken, toToken common.Address, amountIn *big.Int) ([]byte, *SwapRoute, error) {
	router := dexRouters["MerchantMoe"]
	if router == (common.Address{}) {
		return nil, nil, fmt.Errorf("no DEX router configured for this chain")
	}

	route := &SwapRoute{
		Router:    router,
		Path:      []common.Address{fromToken, toToken},
		AmountOut: big.NewInt(0), // 先不设滑点下限，测试走通再说
	}

	// swapExactTokensForTokens(address,uint256,address[],address,uint256)
	calldata, err := packSwapExactTokensForTokens(
		amountIn,
		big.NewInt(1),                 // amountOutMin: 最少 1 wei（测试用）
		[]common.Address{fromToken, toToken},
		b.mgr.Address(),               // 换完的 token 发给自己
		big.NewInt(9999999999),        // deadline: 远的将来
	)
	if err != nil {
		return nil, nil, fmt.Errorf("pack swap calldata: %w", err)
	}

	return calldata, route, nil
}

// packSwapExactTokensForTokens 用 go-ethereum 的 ABI 编码拼 swapExactTokensForTokens 的 calldata
func packSwapExactTokensForTokens(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) ([]byte, error) {
	// swapExactTokensForTokens(address,uint256,address[],address,uint256)
	// keccak256("swapExactTokensForTokens(uint256,uint256,address[],address,uint256)") = 38ed1739
	selector := crypto.Keccak256([]byte("swapExactTokensForTokens(uint256,uint256,address[],address,uint256)"))[:4]

	// ABI encode 参数
	data := append(selector, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amountOutMin.Bytes(), 32)...)

	// address[] path — 先写偏移量（0x80），再写长度，再写地址
	pathOffset := common.LeftPadBytes(big.NewInt(128).Bytes(), 32) // 偏移 128 bytes
	data = append(data, pathOffset...)

	toBytes := common.LeftPadBytes(to.Bytes(), 32)
	data = append(data, toBytes...)

	// deadline
	data = append(data, common.LeftPadBytes(deadline.Bytes(), 32)...)

	// 动态数组：长度 + 每个地址左填充 32 字节
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(path))).Bytes(), 32)...)
	for _, p := range path {
		data = append(data, common.LeftPadBytes(p.Bytes(), 32)...)
	}

	return data, nil
}
