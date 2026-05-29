package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"github.com/yourusername/payflip-backend/config"
	"github.com/yourusername/payflip-backend/internal/tx"
)

func TestMultiStep_ApproveSwap(t *testing.T) {
	// ── 环境准备 ──
	_ = godotenv.Load("../../.env")
	privKeyHex := strings.TrimSpace(os.Getenv("TEST_PRIVATE_KEY"))
	if privKeyHex == "" { t.Skip("TEST_PRIVATE_KEY not set") }
	rpcURL := os.Getenv("MANTLE_TESTNET_RPC")
	if rpcURL == "" { rpcURL = "https://rpc.sepolia.mantle.xyz" }

	privKey, _ := crypto.HexToECDSA(privKeyHex)
	client, _ := ethclient.Dial(rpcURL)
	defer client.Close()
	chainID, _ := client.ChainID(context.Background())

	txmgr, _ := tx.NewTxManager(client, privKey, chainID)
	builder := tx.NewBuilder(txmgr)
	executor := NewIntentExecutor(txmgr, rpcURL, chainID.Int64(), builder)
	registry, _ := config.ParseProtocolRegistry([]byte(`{"protocols":[]}`))
	svc := NewIntentService(builder, registry)

	// ── 1. AI 意图解析 ──
	input := "帮我用 1 USDT 换成 MNT"
	plan, err := svc.BuildPlan(input)
	if err != nil { t.Fatalf("BuildPlan: %v", err) }

	fmt.Println("\n========== AI 多步计划 ==========")
	fmt.Printf("用户输入: %s\n", input)
	for i, s := range plan.Steps {
		fmt.Printf("Step %d: action=%s token=%s amount=%s spender=%s protocol=%s\n", i+1, s.Action, s.Token, s.Amount, s.Spender, s.Protocol)
	}

	// ── 2. calldata 编排 ──
	targets, values, datas := svc.BuildCalldata(plan.Steps)
	fmt.Println("\n========== calldata 编排结果 ==========")
	for i := range targets {
		fmt.Printf("[%d] target=%s value=%s calldata_len=%d\n", i, targets[i].Hex(), values[i].String(), len(datas[i]))
	}

	// ── 3. 执行 ──
	fmt.Println("\n========== 执行结果 ==========")
	hash, err := executor.ExecuteCalldata(context.Background(), targets, values, datas)
	if err != nil {
		t.Fatalf("ExecuteCalldata: %v", err)
	}
	fmt.Printf("txHash: %s\n", hash)
	fmt.Printf("explorer: https://explorer.sepolia.mantle.xyz/tx/%s\n", hash)
}


