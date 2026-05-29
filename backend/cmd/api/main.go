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

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yourusername/hacker-mantle-backend/config"
	"github.com/yourusername/hacker-mantle-backend/internal/api"
	"github.com/yourusername/hacker-mantle-backend/internal/scheduler"
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

	// 3. 加载协议注册表
	registry, err := config.LoadProtocolRegistry("config/protocols.json")
	if err != nil {
		log.Printf("⚠️ 协议注册表加载失败（将继续用硬编码模式）: %v", err)
		registry, _ = config.ParseProtocolRegistry([]byte(`{"protocols":[]}`))
	} else {
		log.Printf("📋 已加载 %d 个协议适配器: %s", len(registry.All()), registry.ProtocolNames())
	}

	// 4. 初始化 Handler
	builder := tx.NewBuilder(nil)
	intentSvc := services.NewIntentService(builder, registry)

	yieldSvc := services.NewYieldService()
	schedCb := func(d scheduler.RebalanceDecision) {
		fmt.Printf("[调度] 调仓建议: %+v\n", d)
	}
	sched := scheduler.NewScheduler(30*time.Minute, schedCb)

	// 设置自动换仓执行器 — 从私钥恢复钱包，发送 swap+stake 交易
	sched.SetExecutor(func(wallet scheduler.ManagedWallet, d scheduler.RebalanceDecision) error {
		privKey, err := crypto.HexToECDSA(wallet.PrivateKey)
		if err != nil {
			return fmt.Errorf("恢复私钥: %w", err)
		}
		rpcURL := os.Getenv("MANTLE_TESTNET_RPC")
		if rpcURL == "" { rpcURL = "https://rpc.sepolia.mantle.xyz" }
		cli, err := ethclient.Dial(rpcURL)
		if err != nil {
			return fmt.Errorf("连接 RPC: %w", err)
		}
		defer cli.Close()
		chainID, _ := cli.ChainID(context.Background())
		txmgr, err := tx.NewTxManager(cli, privKey, chainID)
		if err != nil {
			return fmt.Errorf("创建 TxManager: %w", err)
		}
		defer txmgr.Stop()
		executor := services.NewIntentExecutor(txmgr, rpcURL, chainID.Int64(), tx.NewBuilder(txmgr))
		// 构造换仓步骤：取出旧仓 → 换成目标代币 → 存入新池
		// 简化版：执行 intent "质押到最佳池"
		svc := services.NewIntentService(tx.NewBuilder(txmgr), registry)
		plan, err := svc.BuildPlan(fmt.Sprintf("把所有资金都存到 %s 收益池", d.ToProtocol))
		if err != nil {
			return fmt.Errorf("解析调仓意图: %w", err)
		}
		targets, values, datas := svc.BuildCalldata(plan.Steps)
		if len(targets) == 0 {
			return fmt.Errorf("调仓无步骤")
		}
		_, err = executor.ExecuteCalldata(context.Background(), targets, values, datas)
		return err
	})

	intentHandler := api.NewIntentHandler(intentSvc, sched)
	yieldHandler := api.NewYieldHandler(yieldSvc, sched)

	// 启动收益调度器（30分钟检查一次最佳APY）
	sched.Start()
	fmt.Println("⏰ 收益调度器已启动（周期: 30分钟）")

	// 4. 初始化路由（带收益接口）
	router := api.SetupRouterWithYield(intentHandler, yieldHandler)

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