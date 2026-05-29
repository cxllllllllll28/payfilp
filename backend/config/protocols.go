// Package config — 协议注册表
// 集中管理所有支持的 DeFi 协议信息 (Pool 地址 / ABI 签名 / 资产映射)
// 新增协议只需在 protocols.json 中添加条目 + 在 token 映射中补充地址
package config

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ── 协议适配器 ──────────────────────────────────────────────────────────────

// ProtocolAdapter 描述一个质押协议的全部信息
type ProtocolAdapter struct {
	Name         string   `json:"name"`         // 协议标识，如 "Aave V3"
	PoolAddress  string   `json:"poolAddress"`  // 协议核心合约地址
	DepositSig   string   `json:"depositSig"`   // 存款函数签名，如 "supply(address,uint256,address,uint16)"
	WithdrawSig  string   `json:"withdrawSig"`  // 取款函数签名，如 "withdraw(address,uint256,address)"
	Assets       []string `json:"assets"`       // 支持的资产符号列表
	ReceiptToken string   `json:"receiptToken"` // 存款后获得的凭证代币符号，如 "aUSDT"
}

// BuildDepositCalldata 根据协议适配器动态编码存款 calldata
// asset: 要存入的资产地址  amount: 存入数量  onBehalfOf: 代存地址
func (a *ProtocolAdapter) BuildDepositCalldata(asset common.Address, amount *big.Int, onBehalfOf common.Address) []byte {
	// 按签名类型分发 —— 覆盖常见模式，无需硬编码每种协议
	sig := strings.TrimSpace(a.DepositSig)

	switch {
	case strings.HasPrefix(sig, "supply(address,uint256,address,uint16)"):
		// Aave V3: supply(address asset, uint256 amount, address onBehalfOf, uint16 referralCode)
		return encodeWithSelector(sig, asset, amount, onBehalfOf, uint16(0))

	case strings.HasPrefix(sig, "deposit()"):
		// mETH: deposit() — 无需参数（直接存 ETH）
		return crypto.Keccak256([]byte("deposit()"))[:4]

	case strings.HasPrefix(sig, "deposit(uint256)"):
		// 单参数存款: deposit(uint256 amount)
		return encodeWithSelector(sig, amount)

	case sig == "":
		// 无签名 —— 当做纯 payable 转账
		return nil

	default:
		// 其他自定义签名 —— 通用 ABI 编码，传 asset+amount+user
		// 契约：当签名未知时，按 (address,uint256,address) 尝试
		return encodeWithSelector(sig, asset, amount, onBehalfOf)
	}
}

// BuildWithdrawCalldata 根据协议适配器动态编码取款 calldata
func (a *ProtocolAdapter) BuildWithdrawCalldata(asset common.Address, amount *big.Int, to common.Address) []byte {
	sig := strings.TrimSpace(a.WithdrawSig)

	switch {
	case strings.HasPrefix(sig, "withdraw(address,uint256,address)"):
		// Aave V3: withdraw(address asset, uint256 amount, address to)
		return encodeWithSelector(sig, asset, amount, to)

	case strings.HasPrefix(sig, "withdraw(uint256)"):
		// mETH: withdraw(uint256 amount)
		return encodeWithSelector(sig, amount)

	case sig == "":
		return nil

	default:
		return encodeWithSelector(sig, asset, amount, to)
	}
}

// encodeWithSelector 基于函数签名 + 参数动态编码 calldata
func encodeWithSelector(sig string, args ...interface{}) []byte {
	selector := crypto.Keccak256([]byte(sig))[:4]
	data := selector
	for _, arg := range args {
		data = append(data, abiEncode(arg)...)
	}
	return data
}

// abiEncode 将 Go 值编码为 ABI 32 字节对齐
func abiEncode(v interface{}) []byte {
	switch val := v.(type) {
	case common.Address:
		return common.LeftPadBytes(val.Bytes(), 32)
	case *big.Int:
		return common.LeftPadBytes(val.Bytes(), 32)
	case uint16:
		return common.LeftPadBytes(big.NewInt(int64(val)).Bytes(), 32)
	case string:
		// string/bytes 类型暂不处理动态类型
		return common.LeftPadBytes([]byte(val), 32)
	default:
		return make([]byte, 32)
	}
}

// ── 协议注册表 ──────────────────────────────────────────────────────────────

// ProtocolRegistry 协议注册表 — 按协议名索引
type ProtocolRegistry struct {
	Protocols []ProtocolAdapter `json:"protocols"`
	byName    map[string]*ProtocolAdapter // name → adapter 快速查找
	byToken   map[string]*ProtocolAdapter // receiptToken → adapter
}

// LoadProtocolRegistry 从 JSON 文件加载协议注册表
func LoadProtocolRegistry(path string) (*ProtocolRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read protocols file: %w", err)
	}
	return ParseProtocolRegistry(data)
}

// ParseProtocolRegistry 从 JSON bytes 解析协议注册表
func ParseProtocolRegistry(data []byte) (*ProtocolRegistry, error) {
	var r ProtocolRegistry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse protocols: %w", err)
	}
	r.byName = make(map[string]*ProtocolAdapter, len(r.Protocols))
	r.byToken = make(map[string]*ProtocolAdapter, len(r.Protocols))
	for i := range r.Protocols {
		adapter := &r.Protocols[i]
		key := strings.ToLower(adapter.Name)
		r.byName[key] = adapter
		if adapter.ReceiptToken != "" {
			r.byToken[strings.ToUpper(adapter.ReceiptToken)] = adapter
		}
	}
	return &r, nil
}

// Get 按协议名查找适配器（不区分大小写）
func (r *ProtocolRegistry) Get(name string) (*ProtocolAdapter, bool) {
	adapter, ok := r.byName[strings.ToLower(name)]
	return adapter, ok
}

// GetByReceiptToken 按凭证代币符号查找适配器
func (r *ProtocolRegistry) GetByReceiptToken(symbol string) (*ProtocolAdapter, bool) {
	adapter, ok := r.byToken[strings.ToUpper(symbol)]
	return adapter, ok
}

// All 返回所有注册的协议
func (r *ProtocolRegistry) All() []ProtocolAdapter {
	return r.Protocols
}

// ProtocolNames 返回所有支持的协议名列表（用于注入 Prompt）
func (r *ProtocolRegistry) ProtocolNames() string {
	names := make([]string, len(r.Protocols))
	for i, p := range r.Protocols {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

// ProtocolPrompt 生成供 DeepSeek Prompt 使用的协议描述
func (r *ProtocolRegistry) ProtocolPrompt() string {
	var b strings.Builder
	b.WriteString("支持的收益协议：\n")
	for _, p := range r.Protocols {
		b.WriteString(fmt.Sprintf("  - %s (合约: %s, 存入操作: %s, 取出操作: %s)\n",
			p.Name, p.PoolAddress, p.DepositSig, p.WithdrawSig))
		b.WriteString(fmt.Sprintf("    支持资产: %s, 凭证代币: %s\n",
			strings.Join(p.Assets, ", "), p.ReceiptToken))
	}
	return b.String()
}
