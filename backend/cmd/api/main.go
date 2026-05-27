/*
Hacker-Mantle AI Gasless Yield Agent — 后端入口

=== 知识点 ===

Q1: package main 和 package config 有什么区别？
	package main → 可执行程序的入口（必须有 func main()）
	package config → 普通库包，被其他代码 import 使用
	一个 Go 项目只有一个 package main，其余全是库包。

Q2: defer client.Close() 为什么写在这里而不是 config 包里？
	config 包只负责"加载配置"，不负责"管理连接"。
	ethclient 的连接生命周期由 main 函数管理：
	打开连接 → 传给路由 → 服务退出时关闭。
	这遵循 Go 的"谁创建，谁关闭"原则。
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yourusername/hacker-mantle-backend/config"
	"github.com/yourusername/hacker-mantle-backend/internal/api"
	"github.com/yourusername/hacker-mantle-backend/internal/services"
	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

func main() {
	// 1. 加载 Mantle 链配置
	cfg := config.NewMantleConfig()

	// 2. 连接 Mantle RPC
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		log.Fatalf("无法连接 Mantle RPC (%s): %v", cfg.RPCURL, err)
	}
	defer client.Close()

	fmt.Printf("✅ 已连接 Mantle 主网 (ChainID: %d)\n", cfg.ChainID)

	// 3. 初始化 Intent Handler
	builder := tx.NewBuilder(nil) // 后面接 txmgr
	intentSvc := services.NewIntentService(builder)
	intentHandler := api.NewIntentHandler(intentSvc)

	// 4. 初始化路由
	router := api.SetupRouter(intentHandler)

	// 4. 启动 HTTP 服务（支持优雅关闭）
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		fmt.Println("🚀 服务已启动: http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 5. 等待退出信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	fmt.Println("✅ 服务已安全关闭")
}