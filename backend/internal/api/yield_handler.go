package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourusername/hacker-mantle-backend/internal/scheduler"
	"github.com/yourusername/hacker-mantle-backend/internal/services"
)

// YieldHandler 收益管理 API
type YieldHandler struct {
	yield     *services.YieldService
	scheduler *scheduler.Scheduler
}

func NewYieldHandler(yield *services.YieldService, sched *scheduler.Scheduler) *YieldHandler {
	return &YieldHandler{yield: yield, scheduler: sched}
}

// GetCurrentYields GET /api/yield/current
func (h *YieldHandler) GetCurrentYields(c *gin.Context) {
	yields, err := h.yield.FetchMantleYields(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 转为前端 YieldDashboard 需要的格式
	type poolItem struct {
		Pool    string  `json:"pool"`
		Project string  `json:"project"`
		Symbol  string  `json:"symbol"`
		Apy     float64 `json:"apy"`
		TvlUsd  float64 `json:"tvlUsd"`
	}
	pools := make([]poolItem, 0, len(yields))
	for _, y := range yields {
		pools = append(pools, poolItem{
			Pool:    y.Protocol + ":" + y.Symbol,
			Project: y.Protocol,
			Symbol:  y.Symbol,
			Apy:     y.APY,
			TvlUsd:  y.TVLUsd,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"pools":     pools,
		"updatedAt": c.Request.Context().Value("time"),
	})
}

// TriggerRebalance POST /api/yield/rebalance
func (h *YieldHandler) TriggerRebalance(c *gin.Context) {
	var req struct {
		WalletPk string `json:"walletPk"`
		Strategy string `json:"strategy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 获取全链收益数据，找出最佳协议
	yields, err := h.yield.FetchMantleYields(1_000_000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch yields: " + err.Error()})
		return
	}

	best := services.BestRecommendation(yields)
	if best == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "recommendation": "未找到符合条件的收益池"})
		return
	}

	recommendation := fmt.Sprintf("当前最佳收益: %s %s (APY: %.2f%%, TVL: $%.2fM)",
		best.Protocol, best.Symbol, best.APY, best.TVLUsd/1_000_000)

	// 如果没有私钥，只返回推荐
	if req.WalletPk == "" {
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"recommendation": recommendation,
			"decisions":      h.scheduler.ManualTrigger(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"recommendation": recommendation,
		"decisions":      h.scheduler.ManualTrigger(),
	})
}

// RegisterManaged POST /api/yield/register  — 注册托管钱包
func (h *YieldHandler) RegisterManaged(c *gin.Context) {
	var req struct {
		PrivateKey string `json:"privateKey"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "privateKey is required"})
		return
	}

	addr, err := walletFromPK(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid private key"})
		return
	}

	h.scheduler.RegisterWallet(scheduler.ManagedWallet{
		Address:  addr.Hex(),
		AutoMode: true,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"address": addr.Hex(),
	})
}
