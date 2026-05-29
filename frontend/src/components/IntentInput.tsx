import { useState } from "react";
import { executeIntent } from "../lib/api";
import type { IntentResponse } from "../lib/api";

interface IntentInputProps {
  privateKey: string;
  walletAddress: string;
  onResult: (result: IntentResponse) => void;
  onPending?: (txHash: string) => void;
}

export function IntentInput({
  privateKey,
  walletAddress,
  onResult,
  onPending,
}: IntentInputProps) {
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const execute = async () => {
    if (!input.trim()) return;
    if (!privateKey) {
      setError("请先在顶栏连接钱包");
      return;
    }
    setLoading(true);
    setError("");
    try {
      // 直接把私钥发给后端，后端用私钥签名并发送交易
      const result = await executeIntent({
        input,
        walletPk: privateKey.startsWith("0x") ? privateKey : "0x" + privateKey,
      });

      if (result.error) {
        setError(result.error);
        return;
      }

      if (result.txHash) {
        onPending?.(result.txHash);
      }
      onResult(result);
    } catch (err) {
      setError(`执行失败: ${(err as Error).message}`);
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
      <label className="block text-sm font-medium text-surface-300">
        用自然语言描述你的操作
      </label>
      <div className="flex gap-2">
        <div className="relative flex-1">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="例如：帮我把 1 MNT 换成 USDT"
            className="input-glow w-full px-4 py-3 rounded-xl bg-surface-800/80 border border-white/10 text-surface-100 placeholder-surface-500 focus:border-brand-500/50 outline-none text-sm transition-all duration-200"
          />
          {walletAddress && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
              <span className="text-[10px] text-surface-500 font-mono hidden sm:inline">
                {walletAddress.slice(0, 6)}...
              </span>
            </div>
          )}
        </div>
        <button
          onClick={execute}
          disabled={loading || !input.trim()}
          className="group relative px-6 py-3 rounded-xl text-sm font-semibold text-white overflow-hidden disabled:opacity-40 disabled:cursor-not-allowed transition-all duration-200"
        >
          <span className="absolute inset-0 bg-gradient-to-r from-brand-600 to-mantle-600 rounded-xl" />
          <span className="absolute inset-0 bg-gradient-to-r from-brand-500 to-mantle-500 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300 disabled:opacity-0" />
          <span className="relative z-10 flex items-center gap-2">
            {loading ? (
              <>
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
              </>
            ) : (
              <>
                <svg
                  className="w-4 h-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
                </svg>
                执行
              </>
            )}
          </span>
        </button>
      </div>
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
      <div className="flex items-center gap-3 text-xs text-surface-500">
        <div className="flex items-center gap-1.5">
          <span className="w-1.5 h-1.5 rounded-full bg-brand-500/50" />
          支持 swap / stake / approve
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-1.5 h-1.5 rounded-full bg-mantle-400/50" />
          AI 自动编排多步交易
        </div>
      </div>
    </div>
  );
}
