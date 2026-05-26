package gasless7702

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// operator.go — 三个主流运营商的实现。
// 核心逻辑完全相同：RLP 序列化交易 → eth_sendRawTransaction → 返回 txHash。
// 区别只是 bundlerURL 和初始化参数不同。

// ── Alchemy ───────────────────────────────────────────────────────────────────

// NewAlchemy 使用 Alchemy bundler RPC 广播 type-4 交易。
// bundlerURL 示例：https://eth-mainnet.g.alchemy.com/v2/<API_KEY>
func NewAlchemy(bundlerURL string) GasSponsor { return &bundlerSponsor{"alchemy", bundlerURL} }

// ── Pimlico ───────────────────────────────────────────────────────────────────

// NewPimlico 使用 Pimlico bundler 广播 type-4 交易。
// bundlerURL 示例：https://api.pimlico.io/v2/1/rpc?apikey=<KEY>
func NewPimlico(bundlerURL string) GasSponsor { return &bundlerSponsor{"pimlico", bundlerURL} }

// ── Biconomy ──────────────────────────────────────────────────────────────────

// NewBiconomy 使用 Biconomy bundler 广播 type-4 交易。
// bundlerURL 示例：https://bundler.biconomy.io/api/v2/1/<KEY>
func NewBiconomy(bundlerURL string) GasSponsor { return &bundlerSponsor{"biconomy", bundlerURL} }

// ── 通用实现（三家 bundler API 兼容 eth_sendRawTransaction）────────────────────

type bundlerSponsor struct {
	name       string
	bundlerURL string
}

func (b *bundlerSponsor) Name() string { return b.name }

// SponsorTx 将 type-4 交易序列化后通过 eth_sendRawTransaction 广播。
//
// TODO: BuildSetCodeTx 实现后这里就能跑通，无需改动。
func (b *bundlerSponsor) SponsorTx(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	raw, err := tx.MarshalBinary()
	if err != nil {
		return common.Hash{}, fmt.Errorf("marshal tx: %w", err)
	}

	result, err := sendRPC(ctx, b.bundlerURL, "eth_sendRawTransaction",
		[]interface{}{"0x" + hex.EncodeToString(raw)})
	if err != nil {
		return common.Hash{}, err
	}

	// 解析返回的 txHash（带引号的 JSON 字符串）
	var hashStr string
	if err := jsonUnquote(result, &hashStr); err != nil {
		return common.Hash{}, fmt.Errorf("parse txHash: %w", err)
	}
	return common.HexToHash(hashStr), nil
}

func jsonUnquote(raw []byte, out *string) error {
	if len(raw) < 2 || raw[0] != '"' {
		return fmt.Errorf("expected JSON string, got: %s", raw)
	}
	*out = string(raw[1 : len(raw)-1])
	return nil
}
