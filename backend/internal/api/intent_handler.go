package api

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"

	"github.com/yourusername/hacker-mantle-backend/internal/scheduler"
	"github.com/yourusername/hacker-mantle-backend/internal/services"
	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

type IntentHandler struct {
	svc      *services.IntentService
	scheduler *scheduler.Scheduler
}

func NewIntentHandler(svc *services.IntentService, sched *scheduler.Scheduler) *IntentHandler {
	return &IntentHandler{svc: svc, scheduler: sched}
}

// ExecuteIntent POST /api/intent/execute
func (h *IntentHandler) ExecuteIntent(c *gin.Context) {
	var req struct {
		Input    string `json:"input"`
		WalletPK string `json:"walletPk"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Input == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input is required"})
		return
	}

	plan, err := h.svc.BuildPlan(req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "intent parse: " + err.Error()})
		return
	}

	targets, values, datas := h.svc.BuildCalldata(plan.Steps)

	// 无私钥 → 返回交易参数让前端 MetaMask 签名
	if req.WalletPK == "" {
		var txParams []map[string]interface{}
		for i := range targets {
			txParams = append(txParams, map[string]interface{}{
				"to":    targets[i].Hex(),
				"value": values[i].String(),
				"data":  "0x" + common.Bytes2Hex(datas[i]),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"steps":    plan.Steps,
			"txParams": txParams,
			"preview":  true,
		})
		return
	}

	hash, _, err := h.executeOnChain(c.Request.Context(), strings.TrimSpace(req.WalletPK), targets, values, datas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "execute: " + err.Error()})
		return
	}

	// 构建 explorer URL
	chainID := os.Getenv("MANTLE_CHAIN_ID")
	explorerBase := "https://explorer.sepolia.mantle.xyz"
	if chainID == "5000" {
		explorerBase = "https://mantlescan.io"
	}
	explorerURL := explorerBase + "/tx/" + hash

	// 如果是托管模式 → 自动注册托管监控（存私钥用于自动换仓）
	var managed bool
	if plan.Mode == "managed" {
		managed = true
		wallet, err := walletFromPK(req.WalletPK)
		if err == nil {
			h.scheduler.RegisterWallet(scheduler.ManagedWallet{
				Address:    wallet.Hex(),
				PrivateKey: strings.TrimPrefix(req.WalletPK, "0x"),
				AutoMode:   true,
			})
			fmt.Printf("[托管] 已自动注册钱包 %s 到收益监控调度器（含自动换仓）\n", wallet.Hex())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"txHash":      hash,
		"explorerUrl": explorerURL,
		"steps":       fmt.Sprintf("%+v", plan.Steps),
		"mode":        plan.Mode,
		"managed":     managed,
	})
}

// walletFromPK 从私钥解析出钱包地址
func walletFromPK(pkHex string) (common.Address, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return common.Address{}, err
	}
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	return from, nil
}

func (h *IntentHandler) executeOnChain(ctx context.Context, pkHex string, targets []common.Address, values []*big.Int, datas [][]byte) (string, string, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return "", "", fmt.Errorf("invalid key: %w", err)
	}
	rpcURL := os.Getenv("MANTLE_TESTNET_RPC")
	if rpcURL == "" { rpcURL = "https://rpc.sepolia.mantle.xyz" }
	client, _ := ethclient.Dial(rpcURL)
	defer client.Close()
	chainID, _ := client.ChainID(ctx)
	txmgr, _ := tx.NewTxManager(client, privKey, chainID)
	executor := services.NewIntentExecutor(txmgr, rpcURL, chainID.Int64(), tx.NewBuilder(txmgr))
	hash, err := executor.ExecuteCalldata(ctx, targets, values, datas)
	if err != nil {
		return "", "", err
	}
	return hash, "", nil
}

