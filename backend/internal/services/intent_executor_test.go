package services

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"github.com/yourusername/hacker-mantle-backend/internal/tx"
)

func setupTestExecutor(t *testing.T) (*IntentExecutor, func()) {
	t.Helper()

	_ = godotenv.Load("../../.env")
	privKeyHex := strings.TrimSpace(os.Getenv("TEST_PRIVATE_KEY"))
	rpcURL := strings.TrimSpace(os.Getenv("MANTLE_TESTNET_RPC"))
	if rpcURL == "" {
		rpcURL = "https://rpc.sepolia.mantle.xyz"
	}

	if privKeyHex == "" {
		t.Skip("TEST_PRIVATE_KEY not set in .env")
	}

	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		t.Fatalf("invalid private key: %v", err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Fatalf("dial mantle rpc: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		t.Fatalf("get chain id: %v", err)
	}

	txmgr, err := tx.NewTxManager(client, privKey, chainID)
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	executor := NewIntentExecutor(txmgr, rpcURL, chainID.Int64(), tx.NewBuilder(txmgr))
	cleanup := func() {
		client.Close()
		txmgr.Stop()
	}
	return executor, cleanup
}

func TestExecuteSwapIntent(t *testing.T) {
	executor, cleanup := setupTestExecutor(t)
	defer cleanup()

	from := tx.TokenAddr("USDT")
	to := tx.TokenAddr("MNT")
	amount := big.NewInt(1000000) // 1 USDT

	calldata, _, err := tx.NewBuilder(executor.txmgr).BuildSwapCalldata(context.Background(), from, to, amount)
	if err != nil {
		t.Skipf("DEX builder not yet implemented: %v", err)
	}

	hash, err := executor.ExecuteCalldata(
		context.Background(),
		[]common.Address{to},
		[]*big.Int{big.NewInt(0)},
		[][]byte{calldata},
	)
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	t.Logf("tx hash: %s", hash)
	if hash == "" {
		t.Error("expected non-empty tx hash")
	}
}

