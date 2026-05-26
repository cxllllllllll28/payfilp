package gasless4337

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ── Paymaster/Bundler 接口 ────────────────────────────────────────────────────

// Paymaster 代付 Gas 的运营商接口。
//   - SponsorUserOp：向 Paymaster 请求签名，填充 paymasterAndData 字段
//   - SendUserOp：把已签名的 UserOperation 发给 Bundler，返回 userOpHash
type Paymaster interface {
	Name() string
	// SponsorUserOp 向 Paymaster 申请代付，返回填好 paymasterAndData 的 UserOperation。
	SponsorUserOp(ctx context.Context, op *UserOperation, entryPoint string) (*UserOperation, error)
	// SendUserOp 把 UserOperation 发给 Bundler。
	SendUserOp(ctx context.Context, op *UserOperation, entryPoint string) (userOpHash common.Hash, err error)
}

// PaymasterRegistry chainID → Paymaster。
type PaymasterRegistry struct {
	m map[int64]Paymaster
}

func NewPaymasterRegistry() *PaymasterRegistry {
	return &PaymasterRegistry{m: make(map[int64]Paymaster)}
}

func (r *PaymasterRegistry) Register(chainID int64, p Paymaster) { r.m[chainID] = p }
func (r *PaymasterRegistry) Get(chainID int64) (Paymaster, bool) {
	p, ok := r.m[chainID]
	return p, ok
}

// ── Pimlico 实现 ──────────────────────────────────────────────────────────────

// NewPimlico 创建 Pimlico Paymaster+Bundler。
// bundlerURL/paymasterURL 示例: https://api.pimlico.io/v2/56/rpc?apikey=<KEY>
// gasToken: ERC-20 代付 Gas 时扣除的 token 地址（如 BSC USDT），传空字符串则走 sponsored 模式。
func NewPimlico(bundlerURL, paymasterURL, gasToken string) Paymaster {
	return &pimlicoPaymaster{bundlerURL: bundlerURL, paymasterURL: paymasterURL, gasToken: gasToken}
}

type pimlicoPaymaster struct {
	bundlerURL   string
	paymasterURL string
	gasToken     string // ERC-20 token 地址，空则走 sponsored
}

func (p *pimlicoPaymaster) Name() string { return "pimlico" }

// SponsorUserOp 调用 pm_sponsorUserOperation。
// gasToken 非空时传入 tokenPaymaster 参数，让 Pimlico 事后从用户 token 余额扣 Gas 费。
func (p *pimlicoPaymaster) SponsorUserOp(ctx context.Context, op *UserOperation, entryPoint string) (*UserOperation, error) {
	var params []interface{}
	if p.gasToken != "" {
		params = []interface{}{
			op.ToHex(),
			entryPoint,
			map[string]interface{}{
				"tokenPaymaster": map[string]string{
					"erc20Token":         p.gasToken,
					"priceMarkupPercent": "10",
				},
			},
		}
	} else {
		params = []interface{}{op.ToHex(), entryPoint}
	}

	result, err := sendRPC(ctx, p.paymasterURL, "pm_sponsorUserOperation", params)
	if err != nil {
		return nil, fmt.Errorf("pm_sponsorUserOperation: %w", err)
	}

	var sponsorResp struct {
		PaymasterAndData     string `json:"paymasterAndData"`
		PreVerificationGas   string `json:"preVerificationGas"`
		VerificationGasLimit string `json:"verificationGasLimit"`
		CallGasLimit         string `json:"callGasLimit"`
	}
	if err := json.Unmarshal(result, &sponsorResp); err != nil {
		return nil, fmt.Errorf("parse sponsor response: %w", err)
	}

	op.PaymasterAndData = hexDecode(sponsorResp.PaymasterAndData)
	if v := parseBigHex(sponsorResp.PreVerificationGas); v != nil {
		op.PreVerificationGas = v
	}
	if v := parseBigHex(sponsorResp.VerificationGasLimit); v != nil {
		op.VerificationGasLimit = v
	}
	if v := parseBigHex(sponsorResp.CallGasLimit); v != nil {
		op.CallGasLimit = v
	}
	return op, nil
}

// SendUserOp 调用 eth_sendUserOperation。
func (p *pimlicoPaymaster) SendUserOp(ctx context.Context, op *UserOperation, entryPoint string) (common.Hash, error) {
	result, err := sendRPC(ctx, p.bundlerURL, "eth_sendUserOperation", []interface{}{
		op.ToHex(), entryPoint,
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("eth_sendUserOperation: %w", err)
	}
	var hashStr string
	if err := json.Unmarshal(result, &hashStr); err != nil {
		return common.Hash{}, fmt.Errorf("parse userOpHash: %w", err)
	}
	return common.HexToHash(hashStr), nil
}

// ── Biconomy 实现 ─────────────────────────────────────────────────────────────

// NewBiconomy 创建 Biconomy Paymaster+Bundler。
// bundlerURL 示例: https://bundler.biconomy.io/api/v2/56/<KEY>
// paymasterURL 示例: https://paymaster.biconomy.io/api/v1/56/<KEY>
func NewBiconomy(bundlerURL, paymasterURL string) Paymaster {
	return &biconomyPaymaster{bundlerURL: bundlerURL, paymasterURL: paymasterURL}
}

type biconomyPaymaster struct {
	bundlerURL   string
	paymasterURL string
}

func (b *biconomyPaymaster) Name() string { return "biconomy" }

// SponsorUserOp 调用 pm_sponsorUserOperation（Biconomy 格式相同）。
func (b *biconomyPaymaster) SponsorUserOp(ctx context.Context, op *UserOperation, entryPoint string) (*UserOperation, error) {
	result, err := sendRPC(ctx, b.paymasterURL, "pm_sponsorUserOperation", []interface{}{
		op.ToHex(),
		map[string]string{"mode": "SPONSORED"},
	})
	if err != nil {
		return nil, fmt.Errorf("biconomy pm_sponsorUserOperation: %w", err)
	}
	var sponsorResp struct {
		PaymasterAndData     string `json:"paymasterAndData"`
		PreVerificationGas   string `json:"preVerificationGas"`
		VerificationGasLimit string `json:"verificationGasLimit"`
		CallGasLimit         string `json:"callGasLimit"`
	}
	if err := json.Unmarshal(result, &sponsorResp); err != nil {
		return nil, fmt.Errorf("parse biconomy sponsor response: %w", err)
	}
	op.PaymasterAndData = hexDecode(sponsorResp.PaymasterAndData)
	if v := parseBigHex(sponsorResp.PreVerificationGas); v != nil {
		op.PreVerificationGas = v
	}
	if v := parseBigHex(sponsorResp.VerificationGasLimit); v != nil {
		op.VerificationGasLimit = v
	}
	if v := parseBigHex(sponsorResp.CallGasLimit); v != nil {
		op.CallGasLimit = v
	}
	return op, nil
}

func (b *biconomyPaymaster) SendUserOp(ctx context.Context, op *UserOperation, entryPoint string) (common.Hash, error) {
	result, err := sendRPC(ctx, b.bundlerURL, "eth_sendUserOperation", []interface{}{
		op.ToHex(), entryPoint,
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("biconomy eth_sendUserOperation: %w", err)
	}
	var hashStr string
	if err := json.Unmarshal(result, &hashStr); err != nil {
		return common.Hash{}, fmt.Errorf("parse biconomy userOpHash: %w", err)
	}
	return common.HexToHash(hashStr), nil
}

// ── 通用 JSON-RPC ─────────────────────────────────────────────────────────────

var httpCli = &http.Client{Timeout: 15 * time.Second}

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
	resp, err := httpCli.Do(req)
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
