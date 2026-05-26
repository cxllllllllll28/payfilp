package tx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NonceManager 单个地址的 nonce 分配（线程安全）
type NonceManager struct {
	mu        sync.Mutex
	baseNonce uint64
	reserved  uint64
	client    *ethclient.Client
	addr      common.Address
	lastSync  time.Time
}

// NewNonceManager 创建 nonce 管理器
func NewNonceManager(client *ethclient.Client, addr common.Address) *NonceManager {
	return &NonceManager{
		client: client,
		addr:   addr,
	}
}

// Next 返回下一个可用 nonce
func (n *NonceManager) Next(ctx context.Context) (uint64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 如果超过 30 秒未同步，刷新链上 nonce
	if time.Since(n.lastSync) > 30*time.Second {
		onChain, err := n.client.PendingNonceAt(ctx, n.addr)
		if err != nil {
			return 0, fmt.Errorf("pending nonce: %w", err)
		}
		n.baseNonce = onChain
		n.reserved = 0
		n.lastSync = time.Now()
	}

	nonce := n.baseNonce + n.reserved
	n.reserved++
	return nonce, nil
}
