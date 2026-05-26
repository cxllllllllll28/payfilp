package services

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

// Step 一个执行步骤
type Step struct {
	Action   string `json:"action"`
	Token    string `json:"token,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Amount   string `json:"amount,omitempty"`
	Spender  string `json:"spender,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// StepPlan DeepSeek 返回的完整多步计划
type StepPlan struct {
	Steps []Step `json:"steps"`
}

// StepBuilder 把 Step 数组转为 executeBatch 所需的 targets、values、datas
type StepBuilder struct {
	txBuilder *tx.Builder
}

// NewStepBuilder 创建步骤构建器，复用 tx.Builder 的 calldata 方法
func NewStepBuilder(txBuilder *tx.Builder) *StepBuilder {
	return &StepBuilder{txBuilder: txBuilder}
}

// Build 返回三个并行数组
func (b *StepBuilder) Build(steps []Step) (targets []common.Address, values []*big.Int, datas [][]byte) {
	for _, s := range steps {
		switch s.Action {
		case "approve":
			targets, datas = b.packApprove(targets, datas, s)
		case "swap":
			targets, datas = b.packSwap(targets, datas, s)
		case "stake":
			targets, datas = b.packStake(targets, datas, s)
		case "unstake":
			targets, datas = b.packUnstake(targets, datas, s)
		}
		values = append(values, big.NewInt(0))
	}
	return
}

func (b *StepBuilder) packApprove(targets []common.Address, datas [][]byte, s Step) ([]common.Address, [][]byte) {
	spender := common.HexToAddress(s.Spender)
	amount := amountToBig(s.Amount)
	calldata := tx.BuildApproveCalldata(spender, amount)
	return append(targets, tokenAddr(s.Token)), append(datas, calldata)
}

func (b *StepBuilder) packSwap(targets []common.Address, datas [][]byte, s Step) ([]common.Address, [][]byte) {
	from := tokenAddr(s.From)
	to := tokenAddr(s.To)
	amountIn := amountToBig(s.Amount)
	calldata, _, _ := b.txBuilder.BuildSwapCalldata(nil, from, to, amountIn)
	rtr := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	return append(targets, rtr), append(datas, calldata)
}

func (b *StepBuilder) packStake(targets []common.Address, datas [][]byte, s Step) ([]common.Address, [][]byte) {
	proto := protocolAddr(s.Protocol)
	amount := amountToBig(s.Amount)
	calldata := tx.BuildStakeMETHCalldata()
	_ = amount
	return append(targets, proto), append(datas, calldata)
}

func (b *StepBuilder) packUnstake(targets []common.Address, datas [][]byte, s Step) ([]common.Address, [][]byte) {
	proto := protocolAddr(s.Protocol)
	amount := amountToBig(s.Amount)
	calldata := tx.BuildUnwrapMETHCalldata(amount)
	return append(targets, proto), append(datas, calldata)
}

// ── 辅助函数 ─────────────────────────────────────────────────────────────────

func tokenAddr(symbol string) common.Address {
	m := map[string]common.Address{
		"USDT": common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		"USDC": common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		"MNT":  common.HexToAddress("0x3c3a81e81dc49A522A592e7622A7E711c06bf354"),
		"mETH": common.HexToAddress("0xcDA86A272531e8640cD7F1a92c01839911B90bb0"),
		"USDY": common.HexToAddress("0x8c82B0bD9613b0C6CdED0aE8C1a06191E87F0aF4"),
	}
	if addr, ok := m[symbol]; ok {
		return addr
	}
	return common.Address{}
}

func protocolAddr(protocol string) common.Address {
	m := map[string]common.Address{
		"mETH": common.HexToAddress("0xcDA86A272531e8640cD7F1a92c01839911B90bb0"),
		"USDY": common.HexToAddress("0x8c82B0bD9613b0C6CdED0aE8C1a06191E87F0aF4"),
	}
	if addr, ok := m[protocol]; ok {
		return addr
	}
	return common.Address{}
}

func amountToBig(amount string) *big.Int {
	a := new(big.Int)
	a.SetString(amount, 10)
	if a.Sign() == 0 {
		return big.NewInt(1)
	}
	return a.Mul(a, big.NewInt(1e6))
}
