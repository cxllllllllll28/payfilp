# PayFlip — 实施计划

> **给执行代理的说明：** 必须使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 来逐任务实施本计划。步骤使用复选框 (`- [ ]`) 语法进行追踪。

**目标：** 在 Mantle 上构建一个 AI DeFi 副驾驶——PayFlip。用户用自然语言描述意图，DeepSeek 解析为 steps，后端动态构建 calldata、一笔交易多步原子执行。后续接入收益监控引擎实现全链 100+ 收益池自动调仓。

**架构：** Go 后端 + React 前端（Vite + TailwindCSS）单体仓库。AI 意图解析 + 动态 calldata 构建 + 多步原子交易执行 + Cron 收益调度。

**技术栈：** Go 1.23, DeepSeek V4 Pro API, React 19 + Vite + TailwindCSS + ethers.js v6, Foundry (Solidity 合约可选), Mantle 主网

> ⚠️ **关键前置信息（来自黑客松官方）：**
>
> - **ERC-8004 Agent 身份 NFT 由 Mantle 官方提供**，无需自行部署。申请后获得 Agent ID，在交易中带上即可。
> - **LLM 选型：DeepSeek V4 Pro**（API Key: `sk-1ccd1f8e4586403da617d8bed2c9aa72`，Base URL: `https://api.deepseek.com/v1`），用于意图解析（任务 4）。
> - **Demo 优化**：Cron 调度器（30分钟间隔）太慢，Demo 时使用手动触发接口 `/api/yield/rebalance`。

---

## 文件结构

```
mantleVault-hacker/
├── backend/                    # Go backend
│   ├── cmd/api/main.go
│   ├── config/config.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go           # Gin routes
│   │   │   ├── intent_handler.go   # POST /api/intent/execute
│   │   │   ├── yield_handler.go    # GET /api/yield/monitor, POST /api/yield/rebalance
│   │   │   └── agent_handler.go    # GET /api/agent/status
│   │   ├── tx/                     # 交易执行子层 (builder → signer → sender)
│   │   │   ├── builder.go          # calldata 构造 + DEX 路由查询 + Gas 估算
│   │   │   ├── sender.go           # 发送交易 (正常路径; Gasless 预留接入点)
│   │   │   ├── manager.go          # Nonce 管理 + 重试 + 等待 Confirmation
│   │   │   └── nonce.go
│   │   ├── services/
│   │   │   ├── intent_service.go   # NLP intent → steps (DeepSeek API)
│   │   │   ├── intent_executor.go  # 调度 → 调 tx/ 子系统
│   │   │   ├── yield_service.go    # 收益监控 + 调仓
│   │   │   └── agent_service.go    # Agent 状态追踪
│   │   └── scheduler/
│   │       └── cron.go             # 托管模式定时任务
│   └── go.mod
├── frontend/                   # React app
│   ├── src/
│   │   ├── App.tsx
│   │   ├── components/
│   │   │   ├── IntentInput.tsx     # 自然语言输入框
│   │   │   ├── ModeSwitch.tsx      # 即时/托管模式切换
│   │   │   ├── YieldDashboard.tsx  # 收益监控面板
│   │   │   ├── TxReceipt.tsx       # 交易回执卡片
│   │   │   └── WalletConnect.tsx   # 钱包连接
│   │   ├── hooks/
│   │   │   ├── useIntent.ts
│   │   │   └── useYield.ts
│   │   └── lib/
│   │       └── api.ts
│   ├── package.json
│   └── vite.config.ts
└── docs/
    └── superpowers/
```

---

## 🔴 第一阶段：Mantle 迁移（第1-3天）

### 任务 1：Mantle RPC 与链配置

**涉及文件：**

- 新建: `backend/config/config.go`
- 新建: `backend/config/mantle.go`

- [ ] **步骤 1：编写 Mantle 链配置测试**

```go
func TestMantleConfig_RPCURL(t *testing.T) {
    cfg := config.NewMantleConfig()
    assert.NotEmpty(t, cfg.RPCURL)
    assert.Contains(t, cfg.RPCURL, "mantle")
}

func TestMantleConfig_ChainID(t *testing.T) {
    cfg := config.NewMantleConfig()
    assert.Equal(t, int64(5000), cfg.ChainID)
}
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./config/ -v -run TestMantle`
预期：FAIL — 找不到 config 包

- [ ] **步骤 3：编写 Mantle 配置**

```go
// config/mantle.go
type MantleConfig struct {
    RPCURL  string
    ChainID int64
}

func NewMantleConfig() *MantleConfig {
    return &MantleConfig{
        RPCURL:  getEnv("MANTLE_RPC_URL", "https://rpc.mantle.xyz"),
        ChainID: 5000,
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./config/ -v -run TestMantle`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: add Mantle chain configuration"
```

---

### 任务 2：Mantle 合约地址配置

**涉及文件：**

- 编辑: `backend/config/mantle.go`（追加内容）

- [ ] **步骤 1：添加 Mantle 合约地址**

```go
// Mantle 核心合约地址
const MantleUSDT = "0x..."
const MantleMETH = "0x..."
const MantleUSDY = "0x..."
```

- [ ] **步骤 2：验证核心合约已部署**

运行：`cast code 0x0000000071727De22E5E9d8BAf0edAc6f37da032 --rpc-url https://rpc.mantle.xyz`
预期：返回非空字节码

- [ ] **步骤 3：提交代码**

```bash
git add -A && git commit -m "feat: 添加 Mantle EntryPoint 和 Paymaster 合约地址"
```

---

### 任务 3：Go 后端初始化（Mantle 链）

**涉及文件：**

- 新建: `backend/cmd/api/main.go`
- 新建: `backend/internal/api/router.go`

- [ ] **步骤 1：创建 main.go，初始化 Mantle RPC 客户端**

```go
func main() {
    cfg := config.Load()
    client, _ := ethclient.Dial(cfg.MantleRPC)
    // 初始化 Gin 路由
    r := gin.Default()
    r.POST("/api/intent/execute", intentHandler.ExecuteIntent)
    r.GET("/api/yield/current", yieldHandler.GetCurrentYields)
    r.POST("/api/yield/rebalance", yieldHandler.TriggerRebalance)
    r.GET("/api/agent/status", agentHandler.GetStatus)
    r.Run(":8080")
}
```

- [ ] **步骤 2：创建 router.go，注册所有路由**

- [ ] **步骤 3：启动后端，验证能连接 Mantle**

运行：`go run ./cmd/api/`
预期：`Server running on :8080, connected to Mantle ChainID 5000`

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: Go 后端初始化，连接 Mantle 主网"
```

---

## 🟡 第二阶段：意图引擎（第4-7天）

### 任务 4：AI 意图解析器

**涉及文件：**

- 新建: `backend/internal/services/intent_service.go`
- 新建: `backend/internal/services/intent_service_test.go`

- [ ] **步骤 1：编写失败测试**

```go
func TestParseIntent_BuyMNT(t *testing.T) {
    svc := NewIntentService()
    result, err := svc.Parse("帮我把 100 USDT 换成 MNT")
    assert.NoError(t, err)
    assert.Equal(t, "swap", result.Action)
    assert.Equal(t, "USDT", result.FromToken)
    assert.Equal(t, "MNT", result.ToToken)
    assert.Equal(t, "100", result.Amount)
}

func TestParseIntent_BuyMNTAndStake(t *testing.T) {
    svc := NewIntentService()
    result, err := svc.Parse("用 200 USDT 换成 MNT 并质押生息")
    assert.NoError(t, err)
    assert.Equal(t, "swap_and_stake", result.Action)
}
```

- [ ] **步骤 2：运行测试确认失败**

运行：`go test ./internal/services/ -v -run TestParseIntent`
预期：FAIL

- [ ] **步骤 3：实现意图解析器（调用 LLM API）**

```go
type IntentResult struct {
    Action    string `json:"action"`    // swap | swap_and_stake | stake | unstake
    FromToken string `json:"fromToken"`
    ToToken   string `json:"toToken"`
    Amount    string `json:"amount"`
    YieldPreference string `json:"yieldPreference,omitempty"`
}

func (s *IntentService) Parse(input string) (*IntentResult, error) {
    // 调用 DeepSeek V4 Pro API 解析自然语言意图
    // API Key: sk-2c8554e3ee8c4f0d9c53310772f4556a
    // Base URL: https://api.deepseek.com/v1
    prompt := fmt.Sprintf(`你是一个 DeFi 意图解析器。将用户的自然语言输入解析为 JSON：
{
  "action": "swap|swap_and_stake|stake|unstake",
  "fromToken": "...",
  "toToken": "...",
  "amount": "..."
}
用户输入: %s`, input)
    response := s.callLLM(prompt)
    return parseResponse(response)
}
```

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/services/ -v -run TestParseIntent`
预期：PASS

- [ ] **步骤 5：提交代码**

```bash
git add -A && git commit -m "feat: AI 意图解析器，支持自然语言输入"
```

---

### 任务 5：意图 → calldata 构建器

**涉及文件：**

- 新建: `backend/internal/services/intent_executor.go` — 意图调度，调 tx/ 子系统
- 新建: `backend/internal/tx/builder.go` — calldata 构造 + DEX 路由 + Gas 估算
- 新建: `backend/internal/tx/signer.go` — EOA 签名
- 新建: `backend/internal/tx/sender.go` — 发送（normal / 7702+4337 gasless）
- 新建: `backend/internal/tx/manager.go` — Nonce 管理 + 重试 + 确认
- 测试: `backend/internal/tx/builder_test.go`

- [ ] **步骤 1：编写兑换执行的失败测试**

```go
func TestExecuteSwapIntent(t *testing.T) {
    // CI 环境跳过 — 需要 Mantle 测试网
    if testing.Short() { t.Skip() }

    intent := &IntentResult{
        Action:    "swap",
        FromToken: "USDT",
        ToToken:   "MNT",
        Amount:    "10",
    }
    hash, err := executor.Execute(context.Background(), intent)
    assert.NoError(t, err)
    assert.NotEmpty(t, hash)
}
```

- [ ] **步骤 2：实现兑换执行器**

```go
func (e *IntentExecutor) Execute(ctx context.Context, intent *IntentResult) (string, error) {
    switch intent.Action {
    case "swap":
        return e.executeSwap(ctx, intent)
    case "swap_and_stake":
        return e.executeSwapAndStake(ctx, intent)
    default:
        return "", fmt.Errorf("未知动作: %s", intent.Action)
    }
}
```

- [ ] **步骤 3：连接 tx/ 子系统**

调用 `builder.BuildSwapCalldata` → `manager.BuildTx` → `sender.Send` 完成一笔 swap。

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: intent-to-calldata execution with DEX swap"
```

---

### 任务 6：多步原子执行（动态编排）

**涉及文件：**

- 新建: `backend/internal/services/step_builder.go` — calldata 动态拼接
- 新建: `contracts/src/Simple7702Account.sol` — executeBatch 支持

**核心理念：** 不做固定路由（如 `swap_and_stake`），而是让 AI 解析出任意多步路径，
后端动态拼接 calldata，一笔 UserOp 原子执行。

- [ ] **步骤 1：LLM 解析意图为多步 plan（非固定路径）**

```json
// DeepSeek 输出的意图结构（动态，不限于 swap_and_stake）
{
  "steps": [
    {
      "action": "approve",
      "target": "USDT",
      "spender": "router",
      "amount": "100"
    },
    { "action": "swap", "from": "USDT", "to": "MNT", "amount": "100" },
    { "action": "stake", "amount": "all", "protocol": "mETH" }
  ]
}
```

- [ ] **步骤 2：实现 calldata 动态构建器（Step Builder）**

根据 LLM 返回的 steps 数组，逐个生成 calldata 并用 `Simple7702Account.executeBatch()` 打包：

```go
func (b *StepBuilder) BuildCalldata(steps []Step) (targets []common.Address, values []*big.Int, datas [][]byte) {
    for _, step := range steps {
        switch step.Action {
        case "approve":
            targets = append(targets, USDTAddr)
            datas = append(datas, approveCalldata(step.Spender, step.Amount))
        case "swap":
            targets = append(targets, RouterAddr)
            datas = append(datas, swapCalldata(step.From, step.To, step.Amount))
        case "stake":
            targets = append(targets, mETHAddr)
            datas = append(datas, stakeCalldata(step.Amount))
        case "borrow":
            targets = append(targets, LendleAddr)
            datas = append(datas, borrowCalldata(step.Collateral, step.Token, step.Amount))
        }
        values = append(values, big.NewInt(0))
    }
    return
}
```

- [ ] **步骤 3：合约端 executeBatch 支持任意批量调用**

```solidity
function executeBatch(
    address[] calldata dests,
    uint256[] calldata values,
    bytes[] calldata datas
) external onlyEntryPointOrOwner {
    for (uint i = 0; i < dests.length; i++) {
        (bool ok, ) = dests[i].call{value: values[i]}(datas[i]);
        require(ok, "ExecuteBatch failed");
    }
}
```

- [ ] **步骤 4：运行测试确认通过**

运行：`go test ./internal/services/ -v -run TestBuildCalldata`
预期：PASS — 输入任意 steps JSON，输出正确的 calldata 数组

- [ ] **步骤 5：提交代码**

```bash
git add -A && git commit -m "feat: 动态多步 calldata 构建器 + executeBatch 合约"
```

> 💡 **多步原子交易的含金量延伸（后续迭代方向）：**
>
> 当前阶段实现 LLM 驱动、不限固定路径的动态 step 编排。
> 后续可扩展：
>
> - **条件执行**：上一步返回值决定下一步（如「兑换后价格低于阈值则取消质押」）
> - **跨协议自动发现**：AI 自动搜索 Mantle 上最佳路由（不限于代码预设的协议）
> - **用户自定义 Workflow**：前端拖拽组合步骤，保存为「我的剧本」
>
> 此项进阶功能不强制在当前阶段实现，但架构预留了扩展空间。

---

### 任务 7：意图 API 接口

**涉及文件：**

- 新建: `backend/internal/api/intent_handler.go`

- [ ] **步骤 1：创建 POST /api/intent/execute**

```go
func (h *IntentHandler) ExecuteIntent(c *gin.Context) {
    var req struct {
        Input    string `json:"input"`    // 自然语言输入
        WalletPK string `json:"walletPk"` // 前端传入的钱包私钥
    }
    c.ShouldBindJSON(&req)

    intent, _ := h.intentService.Parse(req.Input)
    hash, _ := h.executor.Execute(c.Request.Context(), intent)

    c.JSON(200, gin.H{"txHash": hash, "intent": intent})
}
```

- [ ] **步骤 2：提交代码**

```bash
git add -A && git commit -m "feat: POST /api/intent/execute 接口"
```

---

## 🟢 第三阶段：托管收益模式（第8-12天）

### 任务 8：收益数据抓取器

**涉及文件：**

- 新建: `backend/internal/services/yield_service.go`
- 测试: `backend/internal/services/yield_service_test.go`

- [ ] **步骤 1：编写失败测试**

```go
func TestFetchMantleYields(t *testing.T) {
    svc := NewYieldService()
    yields, err := svc.FetchCurrentYields()
    assert.NoError(t, err)
    assert.Greater(t, len(yields), 0)
    // 至少应包含 mETH 和 USDY
}
```

- [ ] **步骤 2：通过 DefiLlama API 获取收益数据**

```go
func (s *YieldService) FetchCurrentYields() ([]YieldInfo, error) {
    url := "https://yields.llama.fi/pools?chain=Mantle&active=true"
    // 筛选 mETH、USDY、Aave USDT 三项
}
```

- [ ] **步骤 3：运行测试确认通过**

运行：`go test ./internal/services/ -v -run TestFetchMantleYields`
预期：PASS，返回真实收益数据

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: 基于 DefiLlama API 的收益数据抓取器"
```

---

### 任务 9：调仓决策引擎

**涉及文件：**

- 新建: `backend/internal/services/rebalance_service.go`
- 测试: `backend/internal/services/rebalance_service_test.go`

- [ ] **步骤 1：编写失败测试**

```go
func TestShouldRebalance_BetterOpportunity(t *testing.T) {
    current := YieldInfo{Symbol: "mETH", APY: 1.0}
    available := []YieldInfo{
        {Symbol: "mETH", APY: 1.0},
        {Symbol: "USDY", APY: 3.55},
    }
    decision := engine.ShouldRebalance(current, available)
    assert.True(t, decision.ShouldRebalance)
    assert.Equal(t, "USDY", decision.ToAsset)
}

func TestShouldRebalance_NoChange(t *testing.T) {
    current := YieldInfo{Symbol: "USDY", APY: 3.55}
    available := []YieldInfo{
        {Symbol: "mETH", APY: 1.0},
        {Symbol: "USDY", APY: 3.55},
    }
    decision := engine.ShouldRebalance(current, available)
    assert.False(t, decision.ShouldRebalance)
}
```

- [ ] **步骤 2：实现调仓引擎**

决策规则：

- 新 APY > 当前 APY × 1.2（收益提升超过 20% 才触发）
- Gas 费 < 预期收益 × 0.05（Gas 成本低于收益的 5%）
- 协议 TVL > $1M（安全底线）
- 资产必须在白名单内（mETH、USDY、Aave USDT）

- [ ] **步骤 3：运行测试确认通过**

运行：`go test ./internal/services/ -v -run TestShouldRebalance`
预期：PASS

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: AI 辅助调仓决策引擎"
```

---

### 任务 10：托管模式 Cron 调度器

**涉及文件：**

- 新建: `backend/internal/scheduler/cron.go`

- [ ] **步骤 1：编写失败测试**

```go
func TestScheduler_RunsEvery30Minutes(t *testing.T) {
    sch := NewScheduler(30 * time.Minute)
    assert.Equal(t, 30*time.Minute, sch.interval)
}
```

- [ ] **步骤 2：实现 Cron 调度器（含手动触发）**

```go
func (s *Scheduler) Start() {
    ticker := time.NewTicker(s.interval)
    go func() {
        for range ticker.C {
            s.runRebalanceCycle()
        }
    }()
}

// ManualTrigger 供 Demo 和 API 手动调用（30分钟太慢，评审等不了）
func (s *Scheduler) ManualTrigger() {
    s.runRebalanceCycle()
}

func (s *Scheduler) runRebalanceCycle() {
    yields, _ := s.yieldService.FetchCurrentYields()
    for _, wallet := range s.managedWallets {
        if decision := s.rebalanceEngine.ShouldRebalance(wallet.CurrentYield, yields); decision.ShouldRebalance {
            s.sendNotification(wallet, decision)  // Telegram/Discord
            if wallet.Mode == "auto" {
                s.executeRebalance(wallet, decision)
            }
        }
    }
}
```

- [ ] **步骤 3：提交代码**

```bash
git add -A && git commit -m "feat: 托管模式 Cron 收益监控调度器"
```

---

### 任务 11：托管模式 API 接口

**涉及文件：**

- 新建: `backend/internal/api/yield_handler.go`

- [ ] **步骤 1：创建收益监控接口**

```go
// GET  /api/yield/current    — 获取当前全链收益数据
// POST /api/yield/manage     — 为指定钱包开启托管模式
// POST /api/yield/rebalance  — 手动触发一次调仓
// GET  /api/agent/status     — 查询 Agent 状态和操作历史
```

- [ ] **步骤 2：提交代码**

```bash
git add -A && git commit -m "feat: 托管模式收益管理 API 接口"
```

---

## 🔵 第四阶段：前端开发（第13-18天）

### 任务 12：React 项目初始化

**涉及文件：**

- 新建: `frontend/package.json`
- 新建: `frontend/vite.config.ts`
- 新建: `frontend/tailwind.config.js`

- [ ] **步骤 1：初始化 Vite + React + TailwindCSS**

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend && npm install && npm install tailwindcss @tailwindcss/vite ethers@6
```

- [ ] **步骤 2：配置 Tailwind 和 Vite**

- [ ] **步骤 3：启动开发服务器确认搭建成功**

运行：`npm run dev`
预期：localhost:5173 显示默认 Vite 页面

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: React + Vite + TailwindCSS 前端初始化"
```

---

### 任务 13：钱包连接组件

**涉及文件：**

- 新建: `frontend/src/components/WalletConnect.tsx`

- [ ] **步骤 1：用 ethers.js v6 实现钱包连接**

```tsx
export function WalletConnect() {
  const [account, setAccount] = useState<string | null>(null);

  const connect = async () => {
    if (window.ethereum) {
      const accounts = await window.ethereum.request({
        method: "eth_requestAccounts",
      });
      setAccount(accounts[0]);
    }
  };

  return (
    <button onClick={connect}>
      {account ? `${account.slice(0, 6)}...${account.slice(-4)}` : "连接钱包"}
    </button>
  );
}
```

- [ ] **步骤 2：提交代码**

```bash
git add -A && git commit -m "feat: 钱包连接组件"
```

---

### 任务 14：IntentInput 组件（即时模式）

**涉及文件：**

- 新建: `frontend/src/components/IntentInput.tsx`
- 新建: `frontend/src/lib/api.ts`

- [ ] **步骤 1：创建自然语言输入框及执行逻辑**

```tsx
export function IntentInput() {
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);

  const execute = async () => {
    setLoading(true);
    const res = await fetch("/api/intent/execute", {
      method: "POST",
      body: JSON.stringify({ input, walletPk: "..." }),
    });
    const data = await res.json();
    setLoading(false);
    // 展示交易回执
  };

  return (
    <div className="flex gap-2">
      <input
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="帮我把 100 USDT 换成 MNT 并质押生息..."
        className="flex-1 px-4 py-3 rounded-lg border"
      />
      <button
        onClick={execute}
        disabled={loading}
        className="px-6 py-3 bg-blue-600 text-white rounded-lg"
      >
        {loading ? "执行中..." : "执行"}
      </button>
    </div>
  );
}
```

- [ ] **步骤 2：提交代码**

```bash
git add -A && git commit -m "feat: 意图输入框组件，支持一键执行"
```

---

### 任务 15：模式切换 + 收益看板（托管模式）

**涉及文件：**

- 新建: `frontend/src/components/ModeSwitch.tsx`
- 新建: `frontend/src/components/YieldDashboard.tsx`

- [ ] **步骤 1：创建即时/托管模式切换按钮**

```tsx
export function ModeSwitch({ mode, setMode }: Props) {
  return (
    <div className="flex gap-2">
      <button
        className={mode === "instant" ? "bg-blue-600" : "bg-gray-200"}
        onClick={() => setMode("instant")}
      >
        ⚡ 即时模式
      </button>
      <button
        className={mode === "managed" ? "bg-blue-600" : "bg-gray-200"}
        onClick={() => setMode("managed")}
      >
        🤖 托管模式
      </button>
    </div>
  );
}
```

- [ ] **步骤 2：创建收益看板**

展示实时 mETH/USDY/Aave 收益，附调仓建议。

- [ ] **步骤 3：提交代码**

```bash
git add -A && git commit -m "feat: 模式切换和收益看板组件"
```

---

### 任务 16：交易回执分享卡片

**涉及文件：**

- 新建: `frontend/src/components/TxReceipt.tsx`

- [ ] **步骤 1：创建回执卡片，带 Twitter 分享按钮**

```tsx
export function TxReceipt({ txHash, savedGas, intent }: Props) {
  const shareOnTwitter = () => {
    const text = `我在 @Mantle_Official 上完成了 ${intent}，零 Gas 费！🚀`;
    window.open(
      `https://twitter.com/intent/tweet?text=${encodeURIComponent(text)}`,
    );
  };

  return (
    <div className="p-4 bg-white rounded-lg shadow">
      <p>交易哈希: {txHash.slice(0, 10)}...</p>
      <p>Gas 节省: ${savedGas}</p>
      <button onClick={shareOnTwitter}>分享到 Twitter</button>
    </div>
  );
}
```

- [ ] **步骤 2：提交代码**

```bash
git add -A && git commit -m "feat: 交易回执卡片，支持社交分享"
```

---

## 🟣 第五阶段：部署与打磨（第19-24天）

### 任务 17：将合约部署到 Mantle

**涉及文件：**

- 新建: `contracts/src/Simple7702Account.sol`
- 新建: `contracts/src/ERC20Paymaster.sol`

> 💡 **部署策略**：第 1 阶段后端需要合约地址联调，先部署临时版本。第 5 阶段部署最终版并替换配置。

- [ ] **步骤 1：编写 Simple7702Account（兼容 EIP-7702）**

```solidity
contract Simple7702Account is IAccount {
    function validateUserOp(PackedUserOperation calldata userOp, bytes32 userOpHash, uint256 missingAccountFunds)
        external override returns (uint256 validationData) {
        // 验证 EOA 签名
    }

    function execute(address dest, uint256 value, bytes calldata func) external {
        (bool ok, ) = dest.call{value: value}(func);
        require(ok, "Execute failed");
    }
}
```

- [ ] **步骤 2：在 Mantle 上部署并验证合约**

运行：`forge script Deploy --rpc-url mantle --broadcast --verify`
（EntryPoint 地址用 Pimlico 公共的 `0x0000000071727De22E5E9d8BAf0edAc6f37da032`）

- [ ] **步骤 3：更新后端 config，填入部署后的合约地址**

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: 合约部署至 Mantle 主网"
```

---

### 任务 18：前端部署（Vercel/IPFS）

**涉及文件：**

- 编辑: `frontend/vite.config.ts`（更新 API 基地址）

- [ ] **步骤 1：将 API 地址更新为生产环境**

- [ ] **步骤 2：构建并部署前端**

运行：`npm run build`

- [ ] **步骤 3：确认可公网访问**

---

### 任务 19：ERC-8004 代理身份集成

**涉及文件：**

- 编辑: `backend/internal/services/agent_service.go`

> ⚠️ **Mantle 官方已确认**：ERC-8004 Agent 身份 NFT 由 Mantle 提供，无需自行部署。只需在交易中带上 Agent ID。

- [ ] **步骤 1：从 Mantle 获取 Agent ID，存入配置**

```go
// config/mantle.go
const MantleAgentID = "0x..." // 从 Mantle 黑客松后台获取
```

- [ ] **步骤 2：在所有链上交易的 calldata 中附加 Agent ID**

```go
func (e *IntentExecutor) Execute(ctx context.Context, intent *IntentResult) (string, error) {
    // 在交易 calldata 中附加 Agent ID（ERC-8004 标准）
    txData := attachAgentID(calldata, config.MantleAgentID)
    return e.sendTx(ctx, txData)
}
```

- [ ] **步骤 3：Agent 决策日志记录（链下 + 链上事件）**

```go
func (s *AgentService) LogDecision(decision AgentDecision) {
    // 1. 链下日志（数据库/文件）
    s.logger.Info("Agent decision", decision)
    // 2. 链上 emit 事件（合约内 event AgentAction）
    s.emitOnChain(decision)
}
```

- [ ] **步骤 4：提交代码**

```bash
git add -A && git commit -m "feat: ERC-8004 Agent 身份集成（Mantle 提供）"
```

---

### 任务 20：演示视频（≥ 2 分钟）

- [ ] 展示钱包连接
- [ ] 意图输入 → 一键执行 → 交易回执
- [ ] 托管模式：收益监控 → 调仓执行
- [ ] Explorer 上查看 ERC-8004 Agent 身份和链上记录

### 任务 21：文档与 README

**涉及文件：**

- 新建: `README.md`

- [ ] 架构图（Mermaid）
- [ ] 环境搭建指南
- [ ] 已部署合约地址
- [ ] DoraHacks 提交清单

---

## 总览

| 阶段           | 任务  | 天数      | 备注                                              |
| -------------- | ----- | --------- | ------------------------------------------------- |
| 🔴 Mantle 迁移 | 1-3   | 第1-3天   | Go 后端从零搭建（独立项目，不依赖 web3 旧项目）   |
| 🟡 意图引擎    | 4-7   | 第4-7天   | LLM 意图解析 + calldata 构建 + 多步原子执行       |
| 🟢 托管收益    | 8-11  | 第8-12天  | DefiLlama API + 调仓引擎 + Cron（含手动触发）     |
| 🔵 前端开发    | 12-16 | 第13-18天 | Vite + React + TailwindCSS + ethers.js v6         |
| 🟣 部署打磨    | 17-21 | 第19-24天 | 合约部署 + 前端上线 + ERC-8004 集成（Mantle提供） |

**总计：21 个任务，24 天。**

**与原始计划的关键调整：**

1. 后端为独立项目，不依赖 web3 旧项目（若 eth 交互代码可参考则复用）
2. ERC-8004 Agent ID 由 Mantle 官方提供，无需自行部署合约
3. Cron 调度器增加 `ManualTrigger()` 接口，Demo 时手动触发
4. 合约采用两阶段部署：临时版（联调用）→ 最终版（第5阶段）
5. LLM 选型明确为 DeepSeek V4 Pro（`sk-2c8554e3ee8c4f0d9c53310772f4556a`，Base URL: `https://api.deepseek.com/v1`）
