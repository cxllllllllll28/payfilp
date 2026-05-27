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
          `✅ 调仓成功！交易哈希: ${result.txHash?.slice(0, 10)}... ` +
            `查看: ${result.explorerUrl}`,
        );
      } else {
        setRebalanceResult(
          `❌ 调仓失败: ${result.error || result.recommendation}`,
        );
      }
    } catch (err) {
      setRebalanceResult(`❌ 请求失败: ${(err as Error).message}`);
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
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-800">
          📊 Mantle 收益池 Top 10
        </h3>
        <div className="flex gap-2">
          <button
            onClick={loadYields}
            disabled={loading}
            className="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
          >
            {loading ? "刷新中..." : "🔄 刷新"}
          </button>
          <button
            onClick={handleRebalance}
            disabled={rebalancing || !walletPk}
            className="px-3 py-1.5 text-sm bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 disabled:opacity-50 transition-colors"
          >
            {rebalancing ? "调仓中..." : "⚡ 自动调仓"}
          </button>
        </div>
      </div>

      {error && <p className="text-red-500 text-sm">{error}</p>}

      {rebalanceResult && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-lg text-sm text-emerald-800">
          {rebalanceResult}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-8">
          <svg
            className="animate-spin h-8 w-8 text-indigo-500"
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
        </div>
      ) : topPools.length === 0 ? (
        <p className="text-gray-400 text-center py-4">暂无收益数据</p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 text-gray-600">
                <th className="text-left px-4 py-3 font-medium">#</th>
                <th className="text-left px-4 py-3 font-medium">协议</th>
                <th className="text-left px-4 py-3 font-medium">代币</th>
                <th className="text-right px-4 py-3 font-medium">APY</th>
                <th className="text-right px-4 py-3 font-medium">TVL</th>
              </tr>
            </thead>
            <tbody>
              {topPools.map((pool, i) => (
                <tr
                  key={pool.pool}
                  className="border-t border-gray-100 hover:bg-gray-50 transition-colors"
                >
                  <td className="px-4 py-3 text-gray-400">{i + 1}</td>
                  <td className="px-4 py-3 font-medium">{pool.project}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded text-xs font-mono">
                      {pool.symbol}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <span
                      className={`font-semibold ${pool.apy >= 3 ? "text-emerald-600" : pool.apy >= 1 ? "text-amber-600" : "text-gray-500"}`}
                    >
                      {pool.apy.toFixed(2)}%
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-gray-500">
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
