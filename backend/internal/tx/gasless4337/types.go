// Package gasless4337 实现 EIP-4337 Account Abstraction 的 Gas 代付（Paymaster）。
//
// # 我们只用 4337 的哪一部分？
//
// 完整的 AA 特性很多（合约账户、批量调用、社交恢复……），
// 这里只关心「让 EOA 通过 UserOperation 让 Paymaster 代付 Gas」。
//
// # 流程
//
//  1. 用私钥生成 EOA 地址（与普通钱包相同）
//  2. 为该 EOA 创建/查询链上 SimpleAccount（由 EntryPoint + Factory 确定）
//  3. 构造 UserOperation（callData = swap/approve 的 ABI 编码）
//  4. 向 Paymaster 请求签名（paymasterAndData）
//  5. 把完整 UserOperation 发给 Bundler（eth_sendUserOperation）
//
// # 文件职责划分
//
//   - types.go      — 数据结构（UserOperation、SmartWalletInfo）
//   - paymaster.go  — Paymaster/Bundler 运营商接口与注册表
//   - userop.go     — UserOperation 构造与发送总入口
package gasless4337

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ── EIP-7702 Authorization ────────────────────────────────────────────────────

// Eip7702Auth 是 EIP-7702 授权元组，打包进 UserOperation 让 EntryPoint v0.8
// 在执行前将 sender EOA 临时委托给 delegationContract（Simple7702Account）。
type Eip7702Auth struct {
	ChainId *big.Int
	Address common.Address // delegationContract
	Nonce   uint64
	YParity uint8
	R, S    *big.Int
}

// eip7702AuthHex 是 JSON-RPC 发送时的十六进制格式（内部用）。
type eip7702AuthHex struct {
	ChainId string `json:"chainId"`
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
	YParity string `json:"yParity"`
	R       string `json:"r"`
	S       string `json:"s"`
}

// ── UserOperation ─────────────────────────────────────────────────────────────

// UserOperation EIP-4337 结构体，兼容 v0.6（无 7702）和 v0.8（含 7702 Auth）。
type UserOperation struct {
	Sender               common.Address `json:"sender"`
	Nonce                *big.Int       `json:"nonce"`
	InitCode             []byte         `json:"initCode"`             // 首次部署 SmartAccount 时非空
	CallData             []byte         `json:"callData"`             // 直接调用目标的 calldata（7702 模式不需要 execute 包装）
	CallGasLimit         *big.Int       `json:"callGasLimit"`
	VerificationGasLimit *big.Int       `json:"verificationGasLimit"`
	PreVerificationGas   *big.Int       `json:"preVerificationGas"`
	MaxFeePerGas         *big.Int       `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *big.Int       `json:"maxPriorityFeePerGas"`
	PaymasterAndData     []byte         `json:"paymasterAndData"` // Paymaster 合约地址 + 签名
	Signature            []byte         `json:"signature"`        // EOA 对整个 UserOp hash 的签名
	Auth7702             *Eip7702Auth   `json:"-"`                // EIP-7702 授权，仅 EntryPoint v0.8 时填
}

// UserOpHex 是发送给 Bundler JSON-RPC 时的十六进制字符串表示。
type UserOpHex struct {
	Sender               string          `json:"sender"`
	Nonce                string          `json:"nonce"`
	InitCode             string          `json:"initCode,omitempty"`
	CallData             string          `json:"callData"`
	CallGasLimit         string          `json:"callGasLimit"`
	VerificationGasLimit string          `json:"verificationGasLimit"`
	PreVerificationGas   string          `json:"preVerificationGas"`
	MaxFeePerGas         string          `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string          `json:"maxPriorityFeePerGas"`
	PaymasterAndData     string          `json:"paymasterAndData,omitempty"`
	Signature            string          `json:"signature"`
	Eip7702Auth          *eip7702AuthHex `json:"eip7702Auth,omitempty"`
}

// ToHex 把 UserOperation 转为 Bundler 可接受的十六进制格式。
// initCode 和 paymasterAndData 为空时不输出（Pimlico pm_sponsorUserOperation 会拒绝这些 key）。
func (u *UserOperation) ToHex() UserOpHex {
	h := UserOpHex{
		Sender:               u.Sender.Hex(),
		Nonce:                "0x" + u.Nonce.Text(16),
		CallData:             "0x" + hexEncode(u.CallData),
		CallGasLimit:         "0x" + u.CallGasLimit.Text(16),
		VerificationGasLimit: "0x" + u.VerificationGasLimit.Text(16),
		PreVerificationGas:   "0x" + u.PreVerificationGas.Text(16),
		MaxFeePerGas:         "0x" + u.MaxFeePerGas.Text(16),
		MaxPriorityFeePerGas: "0x" + u.MaxPriorityFeePerGas.Text(16),
		Signature:            "0x" + hexEncode(u.Signature),
	}
	if len(u.InitCode) > 0 {
		h.InitCode = "0x" + hexEncode(u.InitCode)
	}
	if len(u.PaymasterAndData) > 0 {
		h.PaymasterAndData = "0x" + hexEncode(u.PaymasterAndData)
	}
	if a := u.Auth7702; a != nil {
		h.Eip7702Auth = &eip7702AuthHex{
			ChainId: "0x" + a.ChainId.Text(16),
			Address: a.Address.Hex(),
			Nonce:   "0x" + new(big.Int).SetUint64(a.Nonce).Text(16),
			YParity: "0x" + new(big.Int).SetUint64(uint64(a.YParity)).Text(16),
			R:       "0x" + a.R.Text(16),
			S:       "0x" + a.S.Text(16),
		}
	}
	return h
}

func hexEncode(b []byte) string {
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(b)*2)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

// ── SmartWalletInfo ───────────────────────────────────────────────────────────

// SmartWalletInfo 保存一个 EOA 对应的 4337 SmartAccount 信息。
// SmartAccount 地址由 EntryPoint + Factory + EOA owner + salt 决定，可离线计算。
type SmartWalletInfo struct {
	EOAAddress          common.Address `json:"eoa_address"`          // 签名用的普通 EOA
	SmartAccountAddress common.Address `json:"smart_account_address"` // 链上 SmartAccount（代理合约）
	IsDeployed          bool           `json:"is_deployed"`           // SmartAccount 是否已部署
	ChainID             int64          `json:"chain_id"`
}

// ── EntryPoint & Factory ──────────────────────────────────────────────────────

// 常用合约地址（BSC 主网，与 Ethereum 主网相同）。
const (
	// EntryPoint v0.6 合约地址。
	EntryPointV06 = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"

	// EntryPoint v0.9 合约地址（最新），原生支持 EIP-7702 eip7702Auth 字段。
	// BSC 上已验证存在字节码。
	EntryPointV09 = "0x433709009B8330FDa32311DF1C2AFA402eD8D009"

	// Simple7702Account 实现的合约地址（EP9 版本，CREATE2 部署，所有链相同）。
	Simple7702AccountEP9 = "0xa46cc63eBF4Bd77888AA327837d20b23A63a56B5"

	// SimpleAccountFactory v0.6。
	SimpleAccountFactoryV06 = "0x9406Cc6185a346906296840746125a0E44976454"
)
