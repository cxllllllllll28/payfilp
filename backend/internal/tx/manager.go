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
	nm := GetGlobalNonceManager(client, crypto.PubkeyToAddress(privateKey.PublicKey), chainID)
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

// BuildBatchTx 构建多步交易 — 把 targets/values/datas 打包进一笔 via MultiSend/util contract
func (m *TxManager) BuildBatchTx(ctx context.Context, targets []common.Address, values []*big.Int, datas [][]byte) (*types.Transaction, error) {
	// 单步直接复用 BuildTx
	if len(targets) == 1 {
		return m.BuildTx(ctx, &targets[0], values[0], datas[0], 0, nil, nil)
	}
	// 多步：调用简单批处理合约
	batchData, err := encodeBatch(targets, values, datas)
	if err != nil {
		return nil, fmt.Errorf("encode batch: %w", err)
	}
	to := common.HexToAddress("0x0000000000000000000000000000000000000000") // 占位，后续换真正的 MultiSend 地址
	return m.BuildTx(ctx, &to, big.NewInt(0), batchData, 0, nil, nil)
}

// encodeBatch 编码批处理 calldata（纯函数）
func encodeBatch(targets []common.Address, values []*big.Int, datas [][]byte) ([]byte, error) {
	// 简易逻辑：把多个 calldata 串联起来，用 delegatecall 逐条执行
	// 正式上线前换 MultiSend 或 Permit2 的 batch 方法
	var out []byte
	for i := range targets {
		// 每步写入 dest + value + length + data
		out = append(out, common.LeftPadBytes(targets[i].Bytes(), 32)...)
		out = append(out, common.LeftPadBytes(values[i].Bytes(), 32)...)
		out = append(out, common.LeftPadBytes(big.NewInt(int64(len(datas[i]))).Bytes(), 32)...)
		out = append(out, datas[i]...)
	}
	return out, nil
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
