/*
Package api 路由配置 — AI Gasless Yield Agent

=== 知识点 ===

Q1: router.Use 和 router.Group 有什么区别？
	router.Use → 注册中间件（对所有路由生效，如 CORS、日志）
	router.Group → 按路径分组路由（/api/intent 下的接口归一组）

Q2: func(c *gin.Context) 里的 c 是什么？
	c 是当前请求的上下文（Context），包含：
	- c.GetHeader("xxx") → 读取请求头
	- c.ShouldBindJSON(&req) → 解析 JSON Body
	- c.JSON(200, data) → 返回 JSON 响应
	- c.Set("key", value) → 在中间件之间传递数据
*/
package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化所有路由
func SetupRouter(intentHandler *IntentHandler) *gin.Engine {
	router := gin.Default()

	// CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "chain": "Mantle"})
	})

	// === 意图执行接口（任务 7）===
	intentGroup := router.Group("/api/intent")
	intentGroup.POST("/execute", intentHandler.ExecuteIntent)

	return router
}

// SetupRouterWithYield 带收益路由的全量初始化
func SetupRouterWithYield(intentHandler *IntentHandler, yieldHandler *YieldHandler) *gin.Engine {
	router := SetupRouter(intentHandler)

	// === 收益管理接口（任务 11）===
	yieldGroup := router.Group("/api/yield")
	yieldGroup.GET("/current", yieldHandler.GetCurrentYields)
	yieldGroup.POST("/rebalance", yieldHandler.TriggerRebalance)

	return router
}
