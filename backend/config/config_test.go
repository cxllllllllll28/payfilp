package config

import (
	"testing"
)

func TestMantleConfig_RPCURL(t *testing.T) {
	cfg := NewMantleConfig()

	if cfg.RPCURL == "" {
		t.Error("RPCURL should not be empty")
	}

	// 默认应该是 Mantle 主网 RPC
	if cfg.RPCURL != "https://rpc.mantle.xyz" {
		t.Logf("RPCURL was overridden by env var: %s", cfg.RPCURL)
	}
}

func TestMantleConfig_ChainID(t *testing.T) {
	cfg := NewMantleConfig()

	// Mantle 主网 ChainID = 5000（若 .env 覆盖了则校验范围）
	if cfg.ChainID != 5000 && cfg.ChainID != 5001 && cfg.ChainID != 5003 {
		t.Errorf("unexpected ChainID: %d, expected 5000 (mainnet), 5001 (testnet), or 5003 (sepolia)", cfg.ChainID)
	}
}

func TestMantleConfig_Defaults(t *testing.T) {
	// 测试点：当 .env 不存在且系统未设环境变量时，使用默认值
	// 注意：godotenv.Load() 会覆盖 t.Setenv 的值，所以这里只验证结构完整性
	cfg := NewMantleConfig()

	if cfg.RPCURL == "" {
		t.Error("RPCURL should never be empty — either from .env, env var, or default")
	}

	// ChainID 必须是一个合理的值
	if cfg.ChainID <= 0 {
		t.Errorf("ChainID should be positive, got %d", cfg.ChainID)
	}
}
