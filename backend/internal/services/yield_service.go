package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// YieldInfo 一个收益池的数据
type YieldInfo struct {
	Chain      string  `json:"chain"`
	Protocol   string  `json:"project"`
	Symbol     string  `json:"symbol"`
	APY        float64 `json:"apy"`
	TVLUsd     float64 `json:"tvlUsd"`
	StableCoin bool    `json:"stablecoin"`
	Exposure   string  `json:"exposure"`
}

// YieldService 全链收益数据抓取器
type YieldService struct {
	client *http.Client
}

// NewYieldService 创建收益服务
func NewYieldService() *YieldService {
	return &YieldService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchMantleYields 获取 Mantle 上 TVL >= minTVL 的活跃收益池
func (s *YieldService) FetchMantleYields(minTVL float64) ([]YieldInfo, error) {
	resp, err := s.client.Get("https://yields.llama.fi/pools")
	if err != nil {
		return nil, fmt.Errorf("defillama: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var raw struct {
		Data []YieldInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("defillama parse: %w", err)
	}

	filtered := make([]YieldInfo, 0, len(raw.Data))
	for _, p := range raw.Data {
		if p.Chain == "Mantle" && p.TVLUsd >= minTVL {
			filtered = append(filtered, p)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TVLUsd > filtered[j].TVLUsd
	})
	return filtered, nil
}
