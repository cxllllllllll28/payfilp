// Package gasless7702 实现 EIP-7702 无 gas 代付。
//
// adapter_evm.go 使用方式（一行接入）：
//
//	if hash, ok, err := gasless7702.TryGasless(ctx, exec, from, privKey, nonce, chainID, rpcURL, normalTx); ok {
//	    return hash, err
//	}
//	// 原有正常路径...
package gasless7702

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"

	"log"
)

// ── Gas 余额检查 ──────────────────────────────────────────────────────────────

// GasThreshold 低于此值（默认 0.001 ETH）时触发 gasless 路径。
var GasThreshold = new(big.Int).Mul(big.NewInt(1e15), big.NewInt(1))

// HasEnoughGas 检查 addr 在 rpcURL 对应链上的余额是否 >= GasThreshold。
func HasEnoughGas(ctx context.Context, rpcURL string, addr common.Address) (bool, error) {
	result, err := sendRPC(ctx, rpcURL, "eth_getBalance", []interface{}{addr.Hex(), "latest"})
	if err != nil {
		return false, err
	}
	var hexBalance string
	if err := json.Unmarshal(result, &hexBalance); err != nil {
		return false, err
	}
	balance, ok := new(big.Int).SetString(hexBalance[2:], 16) // strip "0x"
	if !ok {
		return false, fmt.Errorf("invalid balance: %s", hexBalance)
	}
	return balance.Cmp(GasThreshold) >= 0, nil
}

// ── EIP-7702 Authorization ────────────────────────────────────────────────────

// Authorization EIP-7702 授权元组：允许 EOA 临时委托给 DelegationContract。
type Authorization struct {
	ChainID            *big.Int
	DelegationContract common.Address
	Nonce              uint64
	V, R, S            *big.Int
}

// BuildAuthorization 构造并签署 EIP-7702 Authorization。
//
// 流程（EIP-7702 规范）：
//  1. RLP 编码 [chainID, delegationContract, nonce]
//  2. magic = 0x05 || rlp_bytes（0x05 是 EIP-7702 规定的域分隔前缀）
//  3. hash = keccak256(magic)
//  4. ECDSA 签名 hash，提取 V/R/S
func BuildAuthorization(chainID *big.Int, contract common.Address, nonce uint64, privKey []byte) (Authorization, error) {
	// Step 1: RLP 编码三元组 [chainID, address, nonce]
	encoded, err := rlp.EncodeToBytes([]interface{}{chainID, contract, nonce})
	if err != nil {
		return Authorization{}, fmt.Errorf("rlp encode authorization: %w", err)
	}

	// Step 2 & 3: 0x05 前缀 + keccak256
	msg := append([]byte{0x05}, encoded...)
	hash := crypto.Keccak256(msg)

	// Step 4: 用私钥签名（go-ethereum 的 crypto.Sign 返回 [R(32)|S(32)|V(1)]）
	key, err := crypto.ToECDSA(privKey)
	if err != nil {
		return Authorization{}, fmt.Errorf("parse private key: %w", err)
	}
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		return Authorization{}, fmt.Errorf("sign authorization: %w", err)
	}

	return Authorization{
		ChainID:            chainID,
		DelegationContract: contract,
		Nonce:              nonce,
		R:                  new(big.Int).SetBytes(sig[:32]),
		S:                  new(big.Int).SetBytes(sig[32:64]),
		V:                  new(big.Int).SetUint64(uint64(sig[64])), // 0 或 1
	}, nil
}

// BuildSetCodeTx 把 Authorization + 原交易 calldata 组装成 EIP-7702 type-4 交易。
//
// 逻辑：把 normalTx 的 To/Value/Data/Gas 原封不动搬进 SetCodeTx，
// 再附上 AuthList，运营商拿到这笔交易后代付 gas 广播。
func BuildSetCodeTx(from common.Address, nonce uint64, normalTx *types.Transaction, auth Authorization) (*types.Transaction, error) {
	if normalTx.To() == nil {
		return nil, fmt.Errorf("normalTx.To is nil, contract creation not supported in gasless path")
	}

	// 把 Authorization 转成 go-ethereum 内部的 SetCodeAuthorization 类型
	authorization := types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(auth.ChainID),
		Address: auth.DelegationContract,
		Nonce:   auth.Nonce,
		V:       uint8(auth.V.Uint64()),
		R:       *uint256.MustFromBig(auth.R),
		S:       *uint256.MustFromBig(auth.S),
	}

	inner := &types.SetCodeTx{
		ChainID:   uint256.MustFromBig(normalTx.ChainId()),
		Nonce:     nonce,
		GasTipCap: uint256.MustFromBig(normalTx.GasTipCap()),
		GasFeeCap: uint256.MustFromBig(normalTx.GasFeeCap()),
		Gas:       normalTx.Gas(),
		To:        *normalTx.To(),
		Value:     uint256.MustFromBig(normalTx.Value()),
		Data:      normalTx.Data(),
		AuthList:  []types.SetCodeAuthorization{authorization},
	}

	return types.NewTx(inner), nil
}

// ── GasSponsor 接口 & 注册表 ──────────────────────────────────────────────────

// GasSponsor 运营商统一接口，只需实现一个方法。
// 能够根据不同链（Ethereum、BSC、Polygon 等）自动选择合适的“付费方”来替用户支付 Gas 费。
type GasSponsor interface {
	SponsorTx(ctx context.Context, tx *types.Transaction) (common.Hash, error)
	Name() string
}

// SponsorRegistry chainID → 运营商。
type SponsorRegistry struct {
	m map[int64]GasSponsor
}

func NewSponsorRegistry() *SponsorRegistry { return &SponsorRegistry{m: make(map[int64]GasSponsor)} }

func (r *SponsorRegistry) Register(chainID int64, s GasSponsor) { r.m[chainID] = s }
func (r *SponsorRegistry) Get(chainID int64) (GasSponsor, bool) {
	s, ok := r.m[chainID]
	return s, ok
}

// ── 总入口：TryGasless ────────────────────────────────────────────────────────

// TryGasless adapter 一行调用的总入口。
//   - gas 充足 → 返回 false，走原有路径
//   - gas 不足 → 构造 type-4 交易，运营商广播，返回 txHash + true
func TryGasless(
	ctx context.Context,
	registry *SponsorRegistry,
	delegationContract common.Address,
	from common.Address,
	privKey []byte,
	nonce uint64,
	chainID *big.Int,
	rpcURL string,
	normalTx *types.Transaction,
) (common.Hash, bool, error) {
	enough, err := HasEnoughGas(ctx, rpcURL, from)
	if err != nil {
		log.Printf("[Gasless] gas 查询失败，降级正常路径: %v", err)
		return common.Hash{}, false, nil
	}
	if enough {
		return common.Hash{}, false, nil
	}

	//运营商注册
	sponsor, ok := registry.Get(chainID.Int64())
	if !ok {
		log.Printf("[Gasless] chainID=%d 无运营商，降级正常路径", chainID.Int64())
		return common.Hash{}, false, nil
	}

	auth, err := BuildAuthorization(chainID, delegationContract, nonce, privKey)
	if err != nil {
		return common.Hash{}, false, fmt.Errorf("build authorization: %w", err)
	}

	tx, err := BuildSetCodeTx(from, nonce, normalTx, auth)
	if err != nil {
		return common.Hash{}, false, fmt.Errorf("build set-code tx: %w", err)
	}

	hash, err := sponsor.SponsorTx(ctx, tx)
	if err != nil {
		return common.Hash{}, false, fmt.Errorf("sponsor [%s]: %w", sponsor.Name(), err)
	}

	log.Printf("[Gasless] 成功 sponsor=%s hash=%s", sponsor.Name(), hash.Hex())
	return hash, true, nil
}

// ── 通用 JSON-RPC（内部使用）────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 10 * time.Second}

type rpcReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}
type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func sendRPC(ctx context.Context, url, method string, params []interface{}) (json.RawMessage, error) {
	body, _ := json.Marshal(rpcReq{JSONRPC: "2.0", Method: method, Params: params, ID: 1})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("rpc %s error %d: %s", method, r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}
