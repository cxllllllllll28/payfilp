package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/hacker-mantle-backend/internal/services"
)

// ManagedWallet 托管钱包配置
type ManagedWallet struct {
	Address      string `json:"address"`
	TelegramID   string `json:"telegramId,omitempty"`
	DiscordID    string `json:"discordId,omitempty"`
	AutoMode     bool   `json:"autoMode"`
	CurrentYield string `json:"currentYield,omitempty"`
}

// RebalanceDecision 一次调仓决策
type RebalanceDecision struct {
	WalletAddress string  `json:"walletAddress"`
	FromProtocol  string  `json:"fromProtocol"`
	ToProtocol    string  `json:"toProtocol"`
	OldAPY        float64 `json:"oldApy"`
	NewAPY        float64 `json:"newApy"`
	EstimatedGain string  `json:"estimatedGain"`
}

// Scheduler 托管收益调度器
type Scheduler struct {
	interval     time.Duration
	yield            *services.YieldService
	wallets      []ManagedWallet
	mu           sync.Mutex
	stopCh       chan struct{}
	onNotify     func(decision RebalanceDecision)
}

// NewScheduler 创建调度器
func NewScheduler(interval time.Duration, onNotify func(RebalanceDecision)) *Scheduler {
	return &Scheduler{
		interval: interval,
		yield:    services.NewYieldService(),
		stopCh:   make(chan struct{}),
		onNotify: onNotify,
	}
}

// RegisterWallet 注册托管钱包
func (s *Scheduler) RegisterWallet(w ManagedWallet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wallets = append(s.wallets, w)
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runCycle()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// ManualTrigger 手动触发调仓检查
func (s *Scheduler) ManualTrigger() []RebalanceDecision {
	return s.runCycle()
}

func (s *Scheduler) runCycle() []RebalanceDecision {
	s.mu.Lock()
	wallets := s.wallets
	s.mu.Unlock()

	yields, err := s.yield.FetchMantleYields(1_000_000)
	if err != nil {
		return nil
	}
	best := services.BestRecommendation(yields)

	var decisions []RebalanceDecision
	for _, w := range wallets {
		decision := RebalanceDecision{
			WalletAddress: w.Address,
			FromProtocol:  w.CurrentYield,
			OldAPY:        0,
			NewAPY:        best.APY,
			ToProtocol:    fmt.Sprintf("%s %s (%.2f%%)", best.Protocol, best.Symbol, best.APY),
			EstimatedGain: "see console",
		}
		decisions = append(decisions, decision)
		if s.onNotify != nil {
			s.onNotify(decision)
		}
	}
	return decisions
}
