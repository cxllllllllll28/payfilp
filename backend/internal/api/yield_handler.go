package api

import (
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
	c.JSON(http.StatusOK, gin.H{"total": len(yields), "data": yields})
}

// TriggerRebalance POST /api/yield/rebalance
func (h *YieldHandler) TriggerRebalance(c *gin.Context) {
	decisions := h.scheduler.ManualTrigger()
	c.JSON(http.StatusOK, gin.H{"decisions": decisions})
}
