# MantleVault — Mantle 链收益聚合器

> The Turing Test Hackathon 2026 · AI x RWA Track  
> 赛道：AI x RWA（Dynamic yield strategies, automated risk management）

---

## 一、项目定位

一句话：**"Mantle 链上的智能收益聚合器，Go 引擎分析链上数据，自动推荐最优 DeFi 策略"**

用户存入 USDC → Solidity Vault 合约 → Go 后端策略引擎实时分析 Mantle 生态各协议的 APY → 推荐最优配置 → 合约自动执行并复利 → 前端展示实时收益。

---

## 二、技术栈

| 层 | 技术选型 | 用途 |
|---|---|---|
| **智能合约** | Solidity + Hardhat + OpenZeppelin | ERC-4626 Vault、策略合约 |
| **后端引擎** | Go（Gin + GORM + go-ethereum） | 策略分析、APY 聚合、链上事件监听 |
| **前端** | React + Vite + wagmi + viem + TailwindCSS | 仪表盘、存入/提取操作 |
| **目标链** | Mantle Network（EVM L2） | 部署链 |
| **数据源** | Mantle RPC + DefiLlama API | 协议 APY 数据 |
| **数据库** | PostgreSQL | 历史 APY、用户持仓 |

---

## 三、核心架构

```
用户存入 USDC
    ↓
┌─ ERC-4626 Vault（Solidity） ─────────────────┐
│  接收用户存款 → 铸造 vault share              │
│  根据 Go 引擎推荐的策略分配资金                   │
│  ├─ mETH 质押 → 获取 mETH 收益                │
│  ├─ Lendle 借贷 → 存款吃利息                    │
│  ├─ USDY 金库 → 获取 RWA 收益                  │
│  └─ Merchant Moe LP → 做市赚手续费              │
│  定时复投 → 收益自动再投资                       │
└──────────────────────────────────────────────┘
    ↑
┌─ Go 策略引擎 ─────────────────────────────────┐
│  定时任务（cron）                               │
│  ├─ 拉取 Mantle RPC → 获取链上各协议利率         │
│  ├─ 拉取 DefiLlama API → 补充 APY 数据          │
│  ├─ 策略评分算法 → 按 收益/风险/流动性 加权排名    │
│  ├─ 更新推荐策略到合约                           │
│  └─ 触发复投操作                                 │
└──────────────────────────────────────────────┘
    ↑
┌─ 前端仪表盘 ──────────────────────────────────┐
│  实时 APY 展示                                   │
│  各协议收益率对比（排行榜）                        │
│  用户持仓 + 收益曲线                              │
│  一键存入/提取                                    │
└──────────────────────────────────────────────┘
```

---

## 四、数据流

```
1. Go 后端定时从 Mantle RPC 拉取以下协议数据：
   - Lendle: 存款利率（supply APY）
   - mETH: 质押收益率
   - USDY (Ondo): RWA 收益率
   - Merchant Moe: LP 池 APR
   - Dolomite: 借贷利率

2. 策略引擎计算加权评分（权重可配置）：
   score = APY × 0.5 + 流动性 × 0.2 + 安全性 × 0.3

3. 最佳策略写入合约（通过 owner 账户调用 setStrategy）

4. Vault 合约按策略比例分配资金

5. 用户在前端查看：
   - 当前总 APY
   - 已获收益（USD + %）
   - 各协议贡献占比
   - 历史收益曲线
```

---

## 五、"AI"元素（不假大空版）

不用 LLM，用 **确定性规则引擎**，但满足黑客松的"AI"命题要求：

| 模块 | 实现 | 面试能说 |
|---|---|---|
| **APY 预测** | 简单加权移动平均 → 预测短期趋势 | "我用历史数据做了收益率趋势预测" |
| **风险评分** | 协议 TVL + 审计状态 + 资金利用率 → 风险分 | "建立了多因子风险评分模型" |
| **策略推荐** | 收益/风险 Pareto 最优排序 | "实现了组合优化算法推荐最优配置" |
| **异常检测** | 利率突变超过 3σ → 触发告警 | "用统计方法检测协议异常" |

---

## 六、为什么 Mantle 需要这个

| Mantle 现状 | MantleVault 解决 |
|---|---|
| 协议多但分散（Lendle/Dolomite/Moe...） | 一个入口查看所有协议 APY |
| 用户手动操作繁琐 | 一键最优配置 + 自动复投 |
| mETH/USDY 买了放着不管 | 智能分配到最高收益处 |
| 缺乏 RWA 类的聚合器 | 第一个聚焦 Mantle RWA 的收益聚合器 |

---

## 七、简历价值

| 技术点 | 简历写法 |
|---|---|
| **Solidity Vault** | "实现 ERC-4626 收益聚合 Vault，支持多策略资金分配和自动复投" |
| **Go 策略引擎** | "用 Go 构建定时数据采集 + 多因子评分系统，聚合 6+ 协议的实时 APY" |
| **链上数据采集** | "通过 Mantle RPC 实时监听链上协议利率变更事件" |
| **DeFi 全栈** | "完整实现从链上数据到策略决策到合约执行的全链路" |
| **Mantle 生态** | "深度集成 mETH/USDY/Lendle/Merchant Moe 等 Mantle 核心协议" |

---

## 八、目录结构规划

```
mantleVault-hacker/
├── contracts/               # Solidity 合约
│   ├── src/
│   │   ├── MantleVault.sol           # ERC-4626 Vault
│   │   ├── StrategyManager.sol        # 策略管理
│   │   └── interfaces/
│   │       ├── ILendle.sol
│   │       ├── ImETH.sol
│   │       └── IUSDY.sol
│   ├── test/
│   ├── hardhat.config.ts
│   └── package.json
├── engine/                   # Go 策略引擎
│   ├── cmd/
│   │   └── engine/main.go
│   ├── internal/
│   │   ├── collector/     # 数据采集（RPC/API）
│   │   ├── scorer/        # 策略评分
│   │   ├── executor/      # 链上执行（合约调用）
│   │   └── model/         # 数据模型
│   ├── go.mod
│   └── go.sum
├── frontend/                # React 前端
│   ├── src/
│   │   ├── pages/          # 仪表盘
│   │   ├── components/     # 图表/卡片
│   │   └── hooks/
│   ├── package.json
│   └── vite.config.ts
└── sign.md
```
