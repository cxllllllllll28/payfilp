import { useState, useEffect, useCallback } from "react";
import { fetchCurrentYields, triggerRebalance } from "../lib/api";
import type { YieldPool } from "../lib/api";

interface YieldDashboardProps {
  walletPk: string;
}

export function YieldDashboard({ walletPk }: YieldDashboardProps) {
  const [pools, setPools] = useState<YieldPool[]>([]);
  const [loading, setLoading] = useState(true);
  const [rebalancing, setRebalancing] = useState(false);
  const [rebalanceResult, setRebalanceResult] = useState<string | null>(null);
  const [error, setError] = useState("");

  const loadYields = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchCurrentYields();
      setPools(data.pools || []);
    } catch (err) {
      setError(`获取收益数据失败: ${(err as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadYields();
  }, [loadYields]);

  const handleRebalance = async () => {
    if (!walletPk) {
      setRebalanceResult("请先输入钱包私钥");
      return;
    }
    setRebalancing(true);
    setRebalanceResult(null);
    try {
      const result = await triggerRebalance({
        walletPk,
        strategy: "highest_apy",
      });
      if (result.success) {
        setRebalanceResult(
          `调仓成功！交易哈希: ${result.txHash?.slice(0, 10)}... 查看: ${result.explorerUrl}`,
        );
      } else {
        setRebalanceResult(
          `调仓失败: ${result.error || result.recommendation}`,
        );
      }
    } catch (err) {
      setRebalanceResult(`请求失败: ${(err as Error).message}`);
    } finally {
      setRebalancing(false);
    }
  };

  const formatTVL = (tvl: number): string => {
    if (tvl >= 1_000_000_000) return `$${(tvl / 1_000_000_000).toFixed(2)}B`;
    if (tvl >= 1_000_000) return `$${(tvl / 1_000_000).toFixed(2)}M`;
    if (tvl >= 1_000) return `$${(tvl / 1_000).toFixed(2)}K`;
    return `$${tvl.toFixed(2)}`;
  };

  const topPools = pools.slice(0, 10);

  return (
    <div className="space-y-4">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-brand-500/10 flex items-center justify-center border border-brand-500/20">
            <svg
              className="w-4 h-4 text-brand-400"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
            >
              <line x1="12" y1="1" x2="12" y2="23" />
              <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
            </svg>
          </div>
          <div>
            <h3 className="text-base font-semibold text-surface-100">
              Mantle 收益池
            </h3>
            <p className="text-xs text-surface-400">Top 10 高收益池实时排行</p>
          </div>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadYields}
            disabled={loading}
            className="px-3 py-1.5 text-xs font-medium bg-surface-800/60 text-surface-300 rounded-lg hover:bg-surface-700/60 hover:text-surface-100 border border-white/5 disabled:opacity-40 transition-all flex items-center gap-1.5"
          >
            <svg
              className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="23 4 23 10 17 10" />
              <polyline points="1 20 1 14 7 14" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
            刷新
          </button>
          <button
            onClick={handleRebalance}
            disabled={rebalancing || !walletPk}
            className="group relative px-3 py-1.5 text-xs font-semibold text-white rounded-lg overflow-hidden disabled:opacity-40 disabled:cursor-not-allowed transition-all"
          >
            <span className="absolute inset-0 bg-gradient-to-r from-brand-600 to-mantle-600 rounded-lg" />
            <span className="relative z-10 flex items-center gap-1.5">
              <svg
                className={`w-3.5 h-3.5 ${rebalancing ? "animate-spin" : ""}`}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <polyline points="23 4 23 10 17 10" />
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
              </svg>
              {rebalancing ? "调仓中..." : "自动调仓"}
            </span>
          </button>
        </div>
      </div>

      {/* 错误 */}
      {error && (
        <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-error/10 border border-error/20 text-error text-sm animate-fade-in-up">
          <svg
            className="w-4 h-4 shrink-0"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>{error}</span>
        </div>
      )}

      {/* 调仓结果 */}
      {rebalanceResult && (
        <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-success/8 border border-success/20 text-success text-sm animate-fade-in-up">
          <span>{rebalanceResult}</span>
        </div>
      )}

      {/* 表格 / 加载 / 空状态 */}
      {loading ? (
        <div className="flex flex-col items-center justify-center py-12 gap-3">
          <svg
            className="animate-spin h-8 w-8 text-brand-400"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
              fill="none"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            />
          </svg>
          <span className="text-xs text-surface-400 animate-shimmer px-6 py-1 rounded">
            正在获取收益数据...
          </span>
        </div>
      ) : topPools.length === 0 ? (
        <div className="text-center py-10">
          <svg
            className="w-12 h-12 mx-auto text-surface-600 mb-3"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1"
          >
            <line x1="12" y1="1" x2="12" y2="23" />
            <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
          </svg>
          <p className="text-surface-500 text-sm">暂无收益数据</p>
          <p className="text-surface-600 text-xs mt-1">请稍后刷新重试</p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-white/5">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-surface-800/40 text-surface-400">
                <th className="text-left px-4 py-3 font-medium text-xs">#</th>
                <th className="text-left px-4 py-3 font-medium text-xs">
                  协议
                </th>
                <th className="text-left px-4 py-3 font-medium text-xs">
                  代币
                </th>
                <th className="text-right px-4 py-3 font-medium text-xs">
                  APY
                </th>
                <th className="text-right px-4 py-3 font-medium text-xs">
                  TVL
                </th>
              </tr>
            </thead>
            <tbody>
              {topPools.map((pool, i) => (
                <tr
                  key={pool.pool}
                  className="border-t border-white/[3%] hover:bg-white/[2%] transition-colors"
                >
                  <td className="px-4 py-3.5 text-surface-500 text-xs font-mono">
                    {i + 1}
                  </td>
                  <td className="px-4 py-3.5 font-medium text-surface-200">
                    <div className="flex items-center gap-2">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-br from-brand-500/20 to-mantle-500/20 flex items-center justify-center text-[8px] text-brand-400 font-bold">
                        {pool.project.charAt(0).toUpperCase()}
                      </div>
                      {pool.project}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    <span className="px-2 py-0.5 bg-brand-500/10 text-brand-300 rounded text-[11px] font-mono font-medium border border-brand-500/15">
                      {pool.symbol}
                    </span>
                  </td>
                  <td className="px-4 py-3.5 text-right">
                    <span
                      className={`font-semibold text-sm ${
                        pool.apy >= 3
                          ? "text-success"
                          : pool.apy >= 1
                            ? "text-warning"
                            : "text-surface-400"
                      }`}
                    >
                      {pool.apy.toFixed(2)}%
                    </span>
                  </td>
                  <td className="px-4 py-3.5 text-right text-surface-400 text-xs font-mono">
                    {formatTVL(pool.tvlUsd)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
