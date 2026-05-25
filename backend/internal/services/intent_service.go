package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// IntentResult 意图解析结果
type IntentResult struct {
	Action          string `json:"action"`     // swap | swap_and_stake | stake | unstake
	FromToken       string `json:"fromToken"`
	ToToken         string `json:"toToken"`
	Amount          string `json:"amount"`
	YieldPreference string `json:"yieldPreference,omitempty"`
}

// IntentService 意图解析服务，调用 DeepSeek API 做自然语言→结构化意图
type IntentService struct {
	apiKey  string
	baseURL string
}

// NewIntentService 创建意图解析服务
func NewIntentService() *IntentService {
	return &IntentService{
		apiKey:  os.Getenv("DEEPSEEK_API_KEY"),
		baseURL: "https://api.deepseek.com/v1",
	}
}

// Parse 将用户的自然语言输入解析为结构化意图
func (s *IntentService) Parse(input string) (*IntentResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY 未设置，请在 .env 中配置")
	}
	prompt := fmt.Sprintf(`你是一个 DeFi 意图解析器。将用户的自然语言输入解析为 JSON。

可用动作：swap（兑换）, swap_and_stake（兑换并质押）, stake（质押）, unstake（解质押）

示例：
- "把 100 USDT 换成 MNT" → {"action":"swap","fromToken":"USDT","toToken":"MNT","amount":"100"}
- "用 200 USDT 换成 MNT 并质押生息" → {"action":"swap_and_stake","fromToken":"USDT","toToken":"MNT","amount":"200"}

现在解析以下输入，只返回 JSON，不要其他文字：
用户输入: %s`, input)

	reqBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
		"max_tokens":  200,
	}

	jsonBody, _ := json.Marshal(reqBody)  // 把 Go map 变成 JSON 字节
	httpReq, err := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 DeepSeek 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DeepSeek API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 解析 DeepSeek 的响应
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("解析 DeepSeek 响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("DeepSeek 返回空结果")
	}

	// 从返回文本中提取 JSON
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// 去掉可能的 markdown 代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result IntentResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("解析意图 JSON 失败: %w, 原文: %s", err, content)
	}

	return &result, nil
}
