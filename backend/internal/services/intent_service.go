package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/yourusername/hacker-mantle-backend/config"
	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

// ── 意图解析结果 ────────────────────────────────────────────────────────────

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
// Mode: "single"（单次执行）或 "managed"（质押到最高APY，需要托管监控）
type StepPlan struct {
	Steps []Step `json:"steps"`
	Mode  string `json:"mode,omitempty"`
}

// ── 意图解析服务 ────────────────────────────────────────────────────────────

// IntentService 意图解析 + calldata 编排
type IntentService struct {
	apiKey    string
	baseURL   string
	txBuilder *tx.Builder
	registry  *config.ProtocolRegistry
}

// NewIntentService 创建意图解析服务
func NewIntentService(txBuilder *tx.Builder, registry *config.ProtocolRegistry) *IntentService {
	return &IntentService{
		apiKey:    os.Getenv("DEEPSEEK_API_KEY"),
		baseURL:   "https://api.deepseek.com/v1",
		txBuilder: txBuilder,
		registry:  registry,
	}
}

// BuildPlan 解析自然语言 → 多步计划（单步/多步统一入口）
func (s *IntentService) BuildPlan(input string) (*StepPlan, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	prompt := fmt.Sprintf(`你是 DeFi 意图解析器。将用户输入转为 JSON。

可用 action: approve, swap, stake, unstake。
%s

可用代币符号: USDT (0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE), MNT, mETH, USDY。
代币地址对应关系：USDT=0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE, MNT=0x3c3a81e81dc49A522A592e7622A7E711c06bf354, mETH=0xcDA86A272531e8640cD7F1a92c01839911B90bb0, USDY=0x8c82B0bD9613b0C6CdED0aE8C1a06191E87F0aF4。
如果用户指定了协议名称，在 step 中填写 protocol 字段。

金额单位: 用户输入的是人类可读单位（如 1 MNT、100 USDT），你直接输出即可。
连续步骤: 如果上一步是 swap，下一步的 amount 用 "all"（表示用上一步得到的全部）。除非用户明确指定新金额。

mode 判断规则:
- 如果用户明确说 "质押到收益最高的池"、"自动调仓"、"最高 APY"、"最佳收益" 或类似表达 → mode="managed"
- 否则 → mode="single"

用户: %s
只返回 JSON，格式: {"mode":"single","steps":[{"action":"swap","from":"MNT","to":"USDT","amount":"1"}]}`, s.registry.ProtocolPrompt(), input)
	content, err := s.callDeepSeek(prompt)
	if err != nil {
		return nil, err
	}
	var plan StepPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("parse step plan: %w", err)
	}
	return &plan, nil
}

// ── calldata 编排 ────────────────────────────────────────────────────────────

// BuildCalldata 把 steps 转成 targets/values/datas
func (s *IntentService) BuildCalldata(steps []Step) (targets []common.Address, values []*big.Int, datas [][]byte) {
	for _, step := range steps {
		switch step.Action {
		case "approve":
			targets, datas = s.packApprove(targets, datas, step)
		case "swap":
			targets, datas = s.packSwap(targets, datas, step)
		case "stake":
			targets, datas = s.packStake(targets, datas, step)
		case "unstake":
			targets, datas = s.packUnstake(targets, datas, step)
		}
		values = append(values, big.NewInt(0))
	}
	return
}

// ── 私有 ─────────────────────────────────────────────────────────────────────

func (s *IntentService) callDeepSeek(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0, "max_tokens": 300,
	}
	jsonBody, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("deepseek request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deepseek %d: %s", resp.StatusCode, string(body))
	}
	var cr struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("deepseek response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("deepseek empty")
	}
	content := strings.TrimSpace(cr.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	// Deepseek 偶尔在 JSON 前加文本，直接找第一个 { 开始
	if idx := strings.IndexByte(content, '{'); idx > 0 {
		content = content[idx:]
	}
	return content, nil
}

func (s *IntentService) packApprove(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	spender := common.HexToAddress(step.Spender)
	amount := amountToBig(step.Amount, step.Token)
	calldata := tx.BuildApproveCalldata(spender, amount)
	return append(targets, tokenAddr(step.Token)), append(datas, calldata)
}

func (s *IntentService) packSwap(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	from := tokenAddr(step.From)
	to := tokenAddr(step.To)
	amountIn := amountToBig(step.Amount, step.From)
	// recipient 先用零地址占位，调用方会在 ExecuteCalldata/executeOnChain 前替换为实际地址
	calldata, _, _ := s.txBuilder.BuildSwapCalldata(nil, from, to, amountIn, common.Address{})
	rtr := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	return append(targets, rtr), append(datas, calldata)
}

func (s *IntentService) packStake(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	adapter, ok := s.registry.Get(step.Protocol)
	if !ok {
		// fallback — 尝试按名称匹配
		addr := common.HexToAddress(step.Protocol)
		if addr == (common.Address{}) {
			addr = tokenAddr(step.Protocol)
		}
		return append(targets, addr), append(datas, tx.BuildStakeMETHCalldata())
	}

	poolAddr := common.HexToAddress(adapter.PoolAddress)

	var assetAddr common.Address
	tokenSymbol := step.Token
	if step.Token != "" {
		assetAddr = tokenAddr(step.Token)
	} else if len(adapter.Assets) > 0 {
		tokenSymbol = adapter.Assets[0]
		assetAddr = tokenAddr(adapter.Assets[0])
	}

	amount := amountToBig(step.Amount, tokenSymbol)
	onBehalfOf := common.HexToAddress("0x0000000000000000000000000000000000000000") // 调用者自己
	calldata := adapter.BuildDepositCalldata(assetAddr, amount, onBehalfOf)
	return append(targets, poolAddr), append(datas, calldata)
}

func (s *IntentService) packUnstake(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	adapter, ok := s.registry.Get(step.Protocol)
	if !ok {
		addr := common.HexToAddress(step.Protocol)
		if addr == (common.Address{}) {
			addr = tokenAddr(step.Protocol)
		}
		amount := amountToBig(step.Amount, step.Protocol)
		return append(targets, addr), append(datas, tx.BuildUnwrapMETHCalldata(amount))
	}

	poolAddr := common.HexToAddress(adapter.PoolAddress)
	var assetAddr common.Address
	tokenSymbol := step.Token
	if step.Token != "" {
		assetAddr = tokenAddr(step.Token)
	} else if len(adapter.Assets) > 0 {
		tokenSymbol = adapter.Assets[0]
		assetAddr = tokenAddr(adapter.Assets[0])
	}

	amount := amountToBig(step.Amount, tokenSymbol)
	to := common.HexToAddress("0x0000000000000000000000000000000000000000")
	calldata := adapter.BuildWithdrawCalldata(assetAddr, amount, to)
	return append(targets, poolAddr), append(datas, calldata)
}

func tokenAddr(symbol string) common.Address {
	m := map[string]common.Address{
		"USDT": common.HexToAddress("0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE"), // Mantle 主网 USDT
		"MNT":  common.HexToAddress("0x3c3a81e81dc49A522A592e7622A7E711c06bf354"),
		"mETH": common.HexToAddress("0xcDA86A272531e8640cD7F1a92c01839911B90bb0"),
		"USDY": common.HexToAddress("0x8c82B0bD9613b0C6CdED0aE8C1a06191E87F0aF4"),
		"aUSDT": common.HexToAddress("0x5B4cF1f7A8E6F0f0E0B8f0E0c0E0B8f0E0c0E0B8"), // 占位，需替换为实际地址
	}
	if addr, ok := m[symbol]; ok {
		return addr
	}
	return common.Address{}
}

// tokenDecimal 返回代币的小数位数
func tokenDecimal(symbol string) int64 {
	switch symbol {
	case "USDT", "USDC":
		return 6
	case "MNT", "mETH", "USDY", "WETH", "WBTC":
		return 18
	default:
		return 18
	}
}

func amountToBig(amount string, symbol ...string) *big.Int {
	a := new(big.Int)
	a.SetString(amount, 10)
	if a.Sign() == 0 {
		return big.NewInt(1)
	}
	// 默认 1e6 (USDT)，如果传入了 symbol 则使用对应精度
	dec := int64(6)
	if len(symbol) > 0 && symbol[0] != "" {
		dec = tokenDecimal(symbol[0])
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(dec), nil)
	return a.Mul(a, pow)
}
