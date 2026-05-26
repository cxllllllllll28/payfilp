package tx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxManager 交易管理器 — 组装 builder/signer/sender 的入口
type TxManager struct {
	client       *ethclient.Client
	privateKey   *ecdsa.PrivateKey
	chainID      *big.Int
	nm           *NonceManager
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewTxManager 创建交易管理器
func NewTxManager(client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int) (*TxManager, error) {
	if client == nil || privateKey == nil || chainID == nil {
		return nil, fmt.Errorf("missing components for TxManager")
	}
	ctx, cancel := context.WithCancel(context.Background())
	nm := NewNonceManager(client, crypto.PubkeyToAddress(privateKey.PublicKey))
	return &TxManager{
		client:     client,
		privateKey: privateKey,
		chainID:    chainID,
		nm:         nm,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Address 返回 EOA 地址
func (m *TxManager) Address() common.Address {
	return crypto.PubkeyToAddress(m.privateKey.PublicKey)
}

// BuildTx 构建交易 — calldata 由 builder 提供
func (m *TxManager) BuildTx(ctx context.Context, to *common.Address, value *big.Int, data []byte, gasLimit uint64, feeCap, tipCap *big.Int) (*types.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nonce, err := m.nm.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}

	if gasLimit == 0 {
		msg := ethereum.CallMsg{From: m.Address(), To: to, Value: value, Data: data}
		estimated, err := m.client.EstimateGas(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("estimate gas: %w", err)
		}
		gasLimit = estimated
	}

	if feeCap == nil || tipCap == nil {
		fc, tc, err := m.suggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("suggest gas price: %w", err)
		}
		feeCap, tipCap = fc, tc
	}

	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   m.chainID,
		Nonce:     nonce,
		To:        to,
		Value:     value,
		Gas:       gasLimit,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		Data:      data,
	}), nil
}

// SignTx 签名交易
func (m *TxManager) SignTx(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return types.SignTx(tx, types.LatestSignerForChainID(m.chainID), m.privateKey)
}

// SendAndWait 发送交易并等待确认
func (m *TxManager) SendAndWait(ctx context.Context, tx *types.Transaction, timeout int64) (*types.Receipt, error) {
	signedTx, err := m.SignTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	err = m.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	if timeout <= 0 {
		timeout = 60
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := m.client.TransactionReceipt(ctx, signedTx.Hash())
		if err == nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("tx %s not confirmed within %ds", signedTx.Hash().Hex(), timeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// Stop 停止管理器
func (m *TxManager) Stop() {
	m.cancel()
}

func (m *TxManager) suggestGasPrice(ctx context.Context) (*big.Int, *big.Int, error) {
	tip, err := m.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	head, err := m.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	baseFee := head.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}
	feeCap := new(big.Int).Add(baseFee, tip)
	feeCap.Mul(feeCap, big.NewInt(2))
	return feeCap, tip, nil
}
