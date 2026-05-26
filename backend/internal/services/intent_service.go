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

	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

// ── 意图解析结果 ────────────────────────────────────────────────────────────

// IntentResult 单步意图结果（兼容旧接口）
type IntentResult struct {
	Action          string `json:"action"`
	FromToken       string `json:"fromToken"`
	ToToken         string `json:"toToken"`
	Amount          string `json:"amount"`
	YieldPreference string `json:"yieldPreference,omitempty"`
}

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

// ── 意图解析服务 ────────────────────────────────────────────────────────────

// IntentService 意图解析 + calldata 编排
type IntentService struct {
	apiKey     string
	baseURL    string
	txBuilder  *tx.Builder
}

// NewIntentService 创建意图解析服务
func NewIntentService(txBuilder *tx.Builder) *IntentService {
	return &IntentService{
		apiKey:    os.Getenv("DEEPSEEK_API_KEY"),
		baseURL:   "https://api.deepseek.com/v1",
		txBuilder: txBuilder,
	}
}

// Parse 解析自然语言 →（单步）
func (s *IntentService) Parse(input string) (*IntentResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	prompt := fmt.Sprintf(`你是一个 DeFi 意图解析器。将用户的自然语言输入解析为 JSON。

可用动作：swap（兑换）, swap_and_stake（兑换并质押）, stake（质押）, unstake（解质押）

示例：
- "把 100 USDT 换成 MNT" → {"action":"swap","fromToken":"USDT","toToken":"MNT","amount":"100"}
- "用 200 USDT 换成 MNT 并质押生息" → {"action":"swap_and_stake","fromToken":"USDT","toToken":"MNT","amount":"200"}

现在解析以下输入，只返回 JSON，不要其他文字：
用户输入: %s`, input)
	content, err := s.callDeepSeek(prompt)
	if err != nil {
		return nil, err
	}
	var r IntentResult
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		return nil, fmt.Errorf("parse intent json: %w, raw: %s", err, content)
	}
	return &r, nil
}

// BuildPlan 解析自然语言 → 多步计划
func (s *IntentService) BuildPlan(input string) (*StepPlan, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	prompt := fmt.Sprintf(`你是 DeFi 意图解析器。将用户输入转为 JSON 步骤数组。可用 action: approve, swap, stake, unstake。

用户: %s
只返回 JSON，格式: {"steps":[{"action":"approve","token":"USDT","spender":"0x...","amount":"100"},...]}`, input)
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

// ── calldata 编排 ─────────────────────────────────────────

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
	resp, err := (&http.Client{}).Do(httpReq)
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
	amount := amountToBig(step.Amount)
	calldata := tx.BuildApproveCalldata(spender, amount)
	return append(targets, tokenAddr(step.Token)), append(datas, calldata)
}

func (s *IntentService) packSwap(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	from := tokenAddr(step.From)
	to := tokenAddr(step.To)
	amountIn := amountToBig(step.Amount)
	calldata, _, _ := s.txBuilder.BuildSwapCalldata(nil, from, to, amountIn)
	rtr := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	return append(targets, rtr), append(datas, calldata)
}

func (s *IntentService) packStake(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	proto := protocolAddr(step.Protocol)
	return append(targets, proto), append(datas, tx.BuildStakeMETHCalldata())
}

func (s *IntentService) packUnstake(targets []common.Address, datas [][]byte, step Step) ([]common.Address, [][]byte) {
	proto := protocolAddr(step.Protocol)
	amount := amountToBig(step.Amount)
	return append(targets, proto), append(datas, tx.BuildUnwrapMETHCalldata(amount))
}

func tokenAddr(symbol string) common.Address {
	m := map[string]common.Address{
		"USDT": common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
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
