import type { IntentResponse } from "../lib/api";

interface TxReceiptProps {
  result: IntentResponse | null;
}

export function TxReceipt({ result }: TxReceiptProps) {
  if (!result) return null;

  const shareOnTwitter = () => {
    const text = result.success
      ? `我在 @Mantle_Official 上完成了 AI DeFi 交易！🚀\n${result.explorerUrl}`
      : `PayFlip AI 意图执行结果：${result.error || "未知"}`;
    window.open(
      `https://twitter.com/intent/tweet?text=${encodeURIComponent(text)}`,
      "_blank",
      "noopener,noreferrer",
    );
  };

  return (
    <div
      className={`rounded-xl border p-5 animate-fade-in-up ${
        result.success
          ? "bg-success/8 border-success/20"
          : "bg-error/8 border-error/20"
      }`}
    >
      <div className="flex items-start gap-4">
        {/* 状态图标 */}
        <div
          className={`w-10 h-10 rounded-full flex items-center justify-center shrink-0 ${
            result.success
              ? "bg-success/15 text-success"
              : "bg-error/15 text-error"
          }`}
        >
          {result.success ? (
            <svg
              className="w-5 h-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          ) : (
            <svg
              className="w-5 h-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          )}
        </div>

        <div className="flex-1 space-y-3 min-w-0">
          <div className="flex items-center gap-2">
            <h4
              className={`font-semibold text-sm ${
                result.success ? "text-success" : "text-error"
              }`}
            >
              {result.success ? "交易成功" : "交易失败"}
            </h4>
            {result.success && (
              <span className="px-2 py-0.5 text-[10px] font-mono font-medium bg-success/10 text-success rounded border border-success/20">
                Confirmed
              </span>
            )}
          </div>

          {result.success && result.txHash && (
            <div className="space-y-2">
              {/* 交易哈希 */}
              <div className="flex items-center gap-2 text-sm">
                <span className="text-surface-400 text-xs shrink-0">
                  交易哈希
                </span>
                <div className="flex items-center gap-1.5 min-w-0">
                  <code className="font-mono text-xs text-surface-300 bg-surface-800/60 px-2.5 py-1 rounded-lg border border-white/5 truncate max-w-[180px]">
                    {result.txHash.slice(0, 10)}...{result.txHash.slice(-8)}
                  </code>
                  <button
                    onClick={() =>
                      navigator.clipboard.writeText(result.txHash!)
                    }
                    className="p-1 text-surface-400 hover:text-surface-200 transition-colors"
                    title="复制"
                  >
                    <svg
                      className="w-3.5 h-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                    </svg>
                  </button>
                </div>
              </div>

              {result.explorerUrl && (
                <a
                  href={result.explorerUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-brand-400 hover:text-brand-300 transition-colors"
                >
                  <svg
                    className="w-3.5 h-3.5"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                    <polyline points="15 3 21 3 21 9" />
                    <line x1="10" y1="14" x2="21" y2="3" />
                  </svg>
                  在浏览器中查看
                </a>
              )}

              {result.steps && (
                <details className="group">
                  <summary className="cursor-pointer text-xs text-surface-400 hover:text-surface-300 transition-colors flex items-center gap-1.5">
                    <svg
                      className="w-3 h-3 group-open:rotate-90 transition-transform"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <polyline points="9 18 15 12 9 6" />
                    </svg>
                    AI 编排步骤
                  </summary>
                  <pre className="mt-2 p-3 bg-surface-800/60 rounded-lg border border-white/5 overflow-x-auto whitespace-pre-wrap text-xs text-surface-300 font-mono leading-relaxed">
                    {typeof result.steps === "string"
                      ? result.steps
                      : JSON.stringify(result.steps, null, 2)}
                  </pre>
                </details>
              )}
            </div>
          )}

          {!result.success && result.error && (
            <p className="text-sm text-error flex items-center gap-1.5">
              <span>{result.error}</span>
            </p>
          )}

          {result.success && (
            <button
              onClick={shareOnTwitter}
              className="mt-1 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-surface-800/60 text-surface-300 rounded-lg hover:bg-surface-700/60 hover:text-surface-100 border border-white/5 transition-all"
            >
              <svg
                className="w-3.5 h-3.5"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
              </svg>
              分享到 X / Twitter
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
