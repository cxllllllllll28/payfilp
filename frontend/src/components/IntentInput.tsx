import { useState } from "react";
import { executeIntent } from "../lib/api";
import type { IntentResponse } from "../lib/api";

interface IntentInputProps {
  walletPk: string;
  onResult: (result: IntentResponse) => void;
}

export function IntentInput({ walletPk, onResult }: IntentInputProps) {
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const execute = async () => {
    if (!input.trim()) return;
    if (!walletPk) {
      setError("请先输入钱包私钥（测试用）");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const result = await executeIntent({ input, walletPk });
      onResult(result);
      if (result.error) setError(result.error);
    } catch (err) {
      setError(`请求失败: ${(err as Error).message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      execute();
    }
  };

  return (
    <div className="space-y-3">
      <label className="block text-sm font-medium text-gray-700">
        用自然语言描述你的操作
      </label>
      <div className="flex gap-2">
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="例如：帮我把 1 MNT 换成 USDT"
          className="flex-1 px-4 py-3 rounded-xl border border-gray-200 bg-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none text-sm transition-shadow"
        />
        <button
          onClick={execute}
          disabled={loading || !input.trim()}
          className="px-6 py-3 bg-indigo-600 text-white rounded-xl hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium text-sm whitespace-nowrap shadow-sm"
        >
          {loading ? (
            <span className="flex items-center gap-2">
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
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
              执行中...
            </span>
          ) : (
            "🚀 执行"
          )}
        </button>
      </div>
      {error && (
        <p className="text-red-500 text-sm flex items-center gap-1">
          <span>⚠️</span> {error}
        </p>
      )}
      <p className="text-xs text-gray-400">
        支持：swap、stake、approve 等操作，AI 自动编排多步交易
      </p>
    </div>
  );
}
