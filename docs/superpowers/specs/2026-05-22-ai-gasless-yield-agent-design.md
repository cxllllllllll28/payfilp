# AI Gasless 收益代理 — 设计规格

> 创建日期：2026-05-22 | 黑客松：Mantle Turing Test Hackathon 2026 Phase 2
> 参赛赛道：AI 交易与策略 + AI 开发工具

---

## 痛点

新 Web3 用户在 Mantle 上面临三大障碍：

1. **Gas 代币门槛** — 必须持有 $MNT 才能交易
2. **DeFi 太复杂** — 协议太多，不知道去哪里赚收益
3. **无法全天候管理** — 不能 7×24 盯盘，错过最佳收益时机

## 解决方案

一个基于意图的 AI 代理，让用户能够：

- **用自然语言说出想要什么**（比如"帮我买 MNT 并质押生息"）
- **用 USDT 付 Gas**（EIP-7702 + ERC-4337 Paymaster）
- **两种模式可选**：即时模式（一键执行）和托管模式（自动调仓）

## 核心功能

### 功能 1：意图驱动执行（即时模式）

- 自然语言 → AI 解析意图 → 生成 calldata → Gasless 执行
- 多步原子化：USDT 授权 → 兑换 → 质押（基于 EIP-7702 批量执行）
- 支持路径：USDT→mETH、USDT→USDY、USDT→Aave USDT

### 功能 2：托管收益模式

- Cron 定时器每 30 分钟监控 mETH/USDY/Aave 的 APY
- 调仓引擎：新 APY 比当前高 20% → 自动挪钱
- Telegram/Discord 通知
- 权限分级：仅提醒 / 自动调仓（设上限）

### 功能 3：Gasless 交易（核心技术）

- EIP-7702：EOA 在一笔交易中委托给智能合约账户
- ERC-4337：Paymaster 代付 Gas，用户用 USDT 结算
- ERC-8004：Agent 身份 NFT，链上记录所有决策

## 收益资产

| 资产      | 年化   | 协议          | 锁仓量 | 风险 |
| --------- | ------ | ------------- | ------ | ---- |
| mETH      | ~2.28% | mETH Protocol | $225M  | 低   |
| USDY      | ~3.55% | Ondo          | $29M   | 低   |
| Aave USDT | ~5.94% | Aave V3       | $25M   | 低   |

## 架构

```
用户（Telegram / 网页）
    │ "帮我把 100 USDT 换成 MNT 并生息"
    ▼
意图解析器（LLM）
    │ { 动作: "兑换并质押", 从代币: "USDT", 目标代币: "MNT" }
    ▼
意图执行器
    │
    ├─ 即时模式 → SendTx（7702+4337 Gasless）
    │
    └─ 托管模式 → 调度器 → 调仓引擎 → SendTx
                       │
                       ▼
               收益数据抓取（DefiLlama API）
```

## 技术栈

- 后端：Go + Gin + go-ethereum
- 前端：React + Vite + TailwindCSS + ethers.js v6
- 智能合约：Solidity 0.8.23（Foundry）
- 区块链：Mantle 主网（ChainID 5000）
- Bundler：Pimlico
- 收益数据：DefiLlama API
