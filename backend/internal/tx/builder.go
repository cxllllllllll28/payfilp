package tx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BuildApproveCalldata 构造 ERC-20 approve(spender, amount) calldata
func BuildApproveCalldata(spender common.Address, amount *big.Int) []byte {
	sel := crypto.Keccak256([]byte("approve(address,uint256)"))[:4]
	data := append(sel, common.LeftPadBytes(spender.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	return data
}

// BuildStakeMETHCalldata 构造 mETH.deposit() calldata
func BuildStakeMETHCalldata() []byte {
	return crypto.Keccak256([]byte("deposit()"))[:4]
}

// BuildUnwrapMETHCalldata 构造 mETH.withdraw(uint256) calldata
func BuildUnwrapMETHCalldata(amount *big.Int) []byte {
	sel := crypto.Keccak256([]byte("withdraw(uint256)"))[:4]
	data := append(sel, common.LeftPadBytes(amount.Bytes(), 32)...)
	return data
}

// DEX 路由表
var dexRouters = map[string]common.Address{
	"MerchantMoe": common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"),
}

// SwapRoute DEX 兑换路由信息
type SwapRoute struct {
	Router     common.Address
	Path       []common.Address
	AmountOut  *big.Int
}

// Builder calldata 构建器
type Builder struct {
	mgr          *TxManager
	ownerAddress common.Address // 独立存储，不依赖 mgr.Address()
}

// NewBuilder 创建 calldata 构建器
// mgr 可以为 nil（仅返回 ownerAddress=zero），主流程调用时传入有效 mgr
func NewBuilder(mgr *TxManager) *Builder {
	b := &Builder{mgr: mgr}
	if mgr != nil && mgr.privateKey != nil {
		b.ownerAddress = crypto.PubkeyToAddress(mgr.privateKey.PublicKey)
	}
	return b
}

// BuildSwapCalldata 构造 Uniswap V2 swapExactTokensForTokens 的 calldata
//
// 调用 ABI: swapExactTokensForTokens(uint256 amountIn, uint256 amountOutMin,
//                                    address[] path, address to, uint256 deadline)
// recipient: 换出的代币接收地址（传入用户 EOA 地址，不能传零地址）
func (b *Builder) BuildSwapCalldata(ctx context.Context, fromToken, toToken common.Address, amountIn *big.Int, recipient common.Address) ([]byte, *SwapRoute, error) {
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
		big.NewInt(1),         // amountOutMin: 最少 1 wei（测试用）
		[]common.Address{fromToken, toToken},
		recipient,              // 换出的 token 发到用户 EOA
		big.NewInt(9999999999), // deadline: 远的将来
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

	// address[] path — 先写偏移量（0xa0），再写长度，再写地址
	// 前面有 5 个静态参数: amountIn(32) + amountOutMin(32) + pathOffset(32) + to(32) + deadline(32) = 160 bytes = 0xa0
	pathOffset := common.LeftPadBytes(big.NewInt(160).Bytes(), 32) // 偏移 160 bytes = 5×32
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
