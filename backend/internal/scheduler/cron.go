package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/payflip-backend/internal/services"
)

// ManagedWallet 托管钱包配置
type ManagedWallet struct {
	Address      string `json:"address"`
	PrivateKey   string `json:"-"` // 用于自动执行换仓（不出现�?JSON 序列化中�?

	AutoMode     bool   `json:"autoMode"`
	CurrentYield string `json:"currentYield,omitempty"`
}

// RebalanceDecision 一次调仓决�?
type RebalanceDecision struct {
	WalletAddress string  `json:"walletAddress"`
	FromProtocol  string  `json:"fromProtocol"`
	ToProtocol    string  `json:"toProtocol"`
	OldAPY        float64 `json:"oldApy"`
	NewAPY        float64 `json:"newApy"`
	EstimatedGain string  `json:"estimatedGain"`
}

// RebalanceExecutor 执行一次调仓的函数签名
// wallet: 托管钱包信息（含私钥�?
// decision: 调仓决策（目标协�?代币/APY�?
type RebalanceExecutor func(wallet ManagedWallet, decision RebalanceDecision) error

// Scheduler 托管收益调度�?
type Scheduler struct {
	interval     time.Duration
	yield        *services.YieldService
	wallets      []ManagedWallet
	mu           sync.Mutex
	stopCh       chan struct{}
	onNotify     func(decision RebalanceDecision)
	onExecute    RebalanceExecutor // 自动执行换仓
}

// NewScheduler 创建调度�?
func NewScheduler(interval time.Duration, onNotify func(RebalanceDecision)) *Scheduler {
	return &Scheduler{
		interval: interval,
		yield:    services.NewYieldService(),
		stopCh:   make(chan struct{}),
		onNotify: onNotify,
	}
}

// SetExecutor 设置自动换仓执行�?
func (s *Scheduler) SetExecutor(exec RebalanceExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onExecute = exec
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

// Stop 停止调度�?
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// ManualTrigger 手动触发调仓检�?
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

		// 通知回调（日�?WebSocket 等）
		if s.onNotify != nil {
			s.onNotify(decision)
		}

		// AutoMode + 有私�?�?自动执行换仓
		s.mu.Lock()
		exec := s.onExecute
		s.mu.Unlock()
		if w.AutoMode && w.PrivateKey != "" && exec != nil {
			go func(wallet ManagedWallet, d RebalanceDecision) {
				if err := exec(wallet, d); err != nil {
					fmt.Printf("[调度] 换仓执行失败 %s: %v\n", wallet.Address, err)
				} else {
					fmt.Printf("[调度] 换仓执行成功 %s �?%s\n", wallet.Address, d.ToProtocol)
				}
			}(w, decision)
		}
	}
	return decisions
}
