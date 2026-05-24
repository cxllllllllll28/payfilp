/*
Package config 从环境变量加载 Mantle 链配置。

=== 测试命令 ===

	go test ./config/ -v                       # 跑 config 包所有测试
	go test ./config/ -v -run TestMantle       # 跑指定测试
	go test ./... -v                           # 跑项目所有测试

.env 文件（backend/.env）：
	MANTLE_RPC_URL=https://rpc.mantle.xyz      # 主网 RPC
	MANTLE_CHAIN_ID=5000                       # 5000=主网, 5001=测试网, 5003=Sepolia

=== 知识点 ===

Q1: 为什么测试代码要放在 config_test.go 而不是 config.go？
	Go 的约定：测试文件用 _test.go 后缀。
	go test 只会执行 _test.go 文件里的测试函数。
	好处：测试代码不会编译进生产二进制 → 包体积更小。

Q2: 为什么 config_test.go 不需要重新定义 MantleConfig 结构体？
	同一个 package 内的所有文件自动共享类型、函数、变量。
	config_test.go 和 config.go 都在 package config 下,
	所以测试文件可以直接用 NewMantleConfig()，不需要 import。

Q3: os.LookupEnv 和 os.Getenv 有什么区别？为什么用 LookupEnv？
	os.LookupEnv(key) → 返回 (value string, exists bool)
	os.Getenv(key)    → 只返回 value string

	LookupEnv 能区分"环境变量不存在"和"环境变量存在但值为空字符串"。
	getEnv() 的逻辑是：key 不存在 → 用默认值；key 存在即使为空 → 用空值。

Q4: godotenv.Load() 和 t.Setenv() 同时用会有什么问题？
	godotenv.Load() 在 NewMantleConfig() 内部执行，读取 .env 文件。
	t.Setenv() 在测试函数开头执行。

	执行顺序：t.Setenv() 先跑 → godotenv.Load() 后跑（覆盖了 Setenv 的值）
	所以测试"默认值"时，不能用 Setenv 清空环境变量再断言具体值。
	正确做法：验证结构完整性（值不为空、范围合理），而不是验精确匹配。

=== Mantle 合约地址（任务 2） ===
	EntryPoint v0.8 (Pimlico) → 0x0000000071727De22E5E9d8BAf0edAc6f37da032
	USDT (Mantle 主网)         → 0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE
	验证命令: cast code 0x... --rpc-url https://rpc.mantle.xyz

Q5: MantleEntryPointV08 和 PimlicoBundlerURL 如何搭配使用？
	EntryPoint 是链上合约地址，Pimlico 是帮你把交易发给这个合约的中转站。

	你（Go 后端）
	    │ 1. 构造并签名 UserOperation
	    │ 2. POST JSON 到 PimlicoBundlerURL()
	    ▼
	Pimlico Bundler（链下服务）
	    │ 3. 验证 UserOp 合法性
	    │ 4. 调用 EntryPoint 合约的 handleOps()
	    ▼
	MantleEntryPointV08（链上 ERC-4337 合约）
	    │ 5. 验证用户签名（通过 Simple7702Account.validateUserOp）
	    │ 6. 执行 calldata（调用 Simple7702Account.execute）
	    │ 7. 调用 Paymaster 收 Gas 费（现场从用户钱包 transferFrom USDT）
	    ▼
	目标合约（兑换 DEX / 质押协议）

	关键：用户不需要持有 MNT，不需要提前存钱。
	ERC20Paymaster 在交易发生时现场从用户钱包扣 USDT 作为 Gas 费。
	用户的 UserOp 里包含了 USDT approve 授权，Paymaster 才能扣款。
*/
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// MantleConfig holds Mantle chain configuration.
type MantleConfig struct {
	RPCURL  string // Mantle RPC endpoint (env: MANTLE_RPC_URL)
	ChainID int64  // Mantle chain ID (env: MANTLE_CHAIN_ID, default: 5000)
}

// === Mantle 合约地址常量 ===

// MantleEntryPointV08 Pimlico ERC-4337 EntryPoint v0.8（公共，无需自行部署）
const MantleEntryPointV08 = "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

// 验证: cast code 0x0000000071727De22E5E9d8BAf0edAc6f37da032 --rpc-url https://rpc.mantle.xyz

// MantleUSDT USDT on Mantle 主网
const MantleUSDT = "0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE"

// 验证: cast code 0x201EBa5CC46D216Ce6DC03F6a759e8E766e956aE --rpc-url https://rpc.mantle.xyz

// PimlicoBundlerURL 返回 Mantle 链的 Pimlico Bundler RPC 地址
func PimlicoBundlerURL() string {
	return fmt.Sprintf("https://api.pimlico.io/v2/5000/rpc?apikey=%s", getEnv("PIMLICO_API_KEY", ""))
}

func NewMantleConfig() *MantleConfig {
	// Load .env file — ok if missing (production uses system env vars)
	_ = godotenv.Load()

	return &MantleConfig{
		RPCURL:  getEnv("MANTLE_RPC_URL", "https://rpc.mantle.xyz"),
		ChainID: getEnvInt("MANTLE_CHAIN_ID", 5000),
	}
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt returns an int64 environment variable or a default value.
func getEnvInt(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		var result int64
		_, err := fmt.Sscanf(value, "%d", &result)
		if err == nil {
			return result
		}
	}
	return defaultValue
}