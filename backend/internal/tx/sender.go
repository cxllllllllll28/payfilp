package tx

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/yourusername/hacker-mantle-backend/internal/tx/gasless7702"
	"github.com/yourusername/hacker-mantle-backend/internal/tx/gasless4337"
)

// GlobalGaslessRegistry gasless 注册表（bootstrap 时设置）
var GlobalGaslessRegistry *gasless4337.PaymasterRegistry

// GlobalDelegationContract EIP-7702 委托合约（bootstrap 时设置）
var GlobalDelegationContract common.Address

// Sender 交易发送器 — 自动选择 normal / gasless 路径
type Sender struct {
	mgr      *TxManager
	rpcURL   string
	chainID  int64
}

// NewSender 创建发送器
func NewSender(mgr *TxManager, rpcURL string, chainID int64) *Sender {
	return &Sender{mgr: mgr, rpcURL: rpcURL, chainID: chainID}
}

// Send 发送交易 — 自动分路：有余额走 normal，没余额走 gasless 7702+4337
func (s *Sender) Send(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	// 有 gas 直接发
	hasGas, _ := gasless7702.HasEnoughGas(ctx, s.rpcURL, s.mgr.Address())
	if hasGas {
		return s.sendNormal(ctx, tx)
	}

	// 没 gas → 7702+4337
	// 前提：GlobalGaslessRegistry 和 GlobalDelegationContract 已由 bootstrap 注入
	if GlobalGaslessRegistry == nil || GlobalDelegationContract == (common.Address{}) {
		return common.Hash{}, fmt.Errorf("gasless not configured: PaymasterRegistry or DelegationContract missing")
	}

	nonce := tx.Nonce()
	chainID := tx.ChainId()
	if chainID == nil {
		chainID = big.NewInt(s.chainID)
	}

	// 1. 签 EIP-7702 授权
	privKeyBytes, err := s.loadPrivKeyBytes()
	if err != nil {
		return common.Hash{}, fmt.Errorf("load private key: %w", err)
	}
	auth7702, err := gasless7702.BuildAuthorization(chainID, GlobalDelegationContract, nonce, privKeyBytes)
	if err != nil {
		return common.Hash{}, fmt.Errorf("build 7702 auth: %w", err)
	}

	// 2. 查 EntryPoint nonce
	epNonce, epErr := gasless4337.GetEntryPointNonce(ctx, s.rpcURL, s.mgr.Address(), gasless4337.EntryPointV09)
	if epErr != nil {
		log.Printf("[Gasless] EntryPoint nonce query failed, fallback to normal: %v", epErr)
		return s.sendNormal(ctx, tx)
	}

	// 3. 转换 7702 auth → 4337 结构
	a4337 := &gasless4337.Eip7702Auth{
		ChainId: auth7702.ChainID,
		Address: auth7702.DelegationContract,
		Nonce:   auth7702.Nonce,
		YParity: uint8(auth7702.V.Uint64()),
		R:       auth7702.R,
		S:       auth7702.S,
	}

	tip := tx.GasTipCap()
	fee := tx.GasFeeCap()
	if tip == nil { tip = tx.GasPrice() }
	if fee == nil { fee = tx.GasPrice() }

	// 4. TrySponsoredOp
	hash, err := gasless4337.TrySponsoredOp(
		ctx, GlobalGaslessRegistry, s.chainID,
		s.mgr.Address(), epNonce, *tx.To(), tx.Value(), tx.Data(),
		fee, tip, a4337, privKeyBytes,
	)
	if err != nil {
		log.Printf("[Gasless] 7702+4337 failed, fallback to normal: %v", err)
		return s.sendNormal(ctx, tx)
	}

	return hash, nil
}

func (s *Sender) sendNormal(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	signedTx, err := s.mgr.SignTx(ctx, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("sign: %w", err)
	}
	err = s.mgr.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("send: %w", err)
	}
	return signedTx.Hash(), nil
}

func (s *Sender) loadPrivKeyBytes() ([]byte, error) {
	hexKey := os.Getenv("PRIVATE_KEY")
	if hexKey == "" {
		hexKey = os.Getenv("TEST_PRIVATE_KEY")
	}
	if hexKey == "" {
		return nil, fmt.Errorf("PRIVATE_KEY not set in .env")
	}
	return common.FromHex(hexKey), nil
}
