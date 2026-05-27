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

	"github.com/yourusername/hacker-mantle-backend/internal/services"
	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

type IntentHandler struct {
	svc *services.IntentService
}

func NewIntentHandler(svc *services.IntentService) *IntentHandler {
	return &IntentHandler{svc: svc}
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

	if req.WalletPK == "" {
		c.JSON(http.StatusOK, gin.H{"steps": plan.Steps, "targets": len(targets)})
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

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"txHash":      hash,
		"explorerUrl": explorerURL,
		"steps":       fmt.Sprintf("%+v", plan.Steps),
	})
}

func (h *IntentHandler) executeOnChain(ctx context.Context, pkHex string, targets []common.Address, values []*big.Int, datas [][]byte) (string, string, error) {
	privKey, err := crypto.HexToECDSA(pkHex)
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

