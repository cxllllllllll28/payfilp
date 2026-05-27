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
      className={`rounded-xl border p-4 ${result.success ? "bg-emerald-50 border-emerald-200" : "bg-red-50 border-red-200"}`}
    >
      <div className="flex items-start gap-3">
        <span className="text-2xl">{result.success ? "✅" : "❌"}</span>
        <div className="flex-1 space-y-2">
          <h4 className="font-semibold text-sm">
            {result.success ? "交易成功" : "交易失败"}
          </h4>

          {result.success && result.txHash && (
            <div className="space-y-1">
              <div className="flex items-center gap-2 text-sm text-gray-600">
                <span className="text-gray-400">交易哈希:</span>
                <code className="font-mono text-xs bg-white px-2 py-0.5 rounded border">
                  {result.txHash.slice(0, 10)}...{result.txHash.slice(-8)}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(result.txHash!)}
                  className="text-indigo-500 hover:text-indigo-700 text-xs"
                  title="复制"
                >
                  📋
                </button>
              </div>

              {result.explorerUrl && (
                <a
                  href={result.explorerUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-sm text-indigo-600 hover:text-indigo-800"
                >
                  🔗 在浏览器中查看
                </a>
              )}

              {result.steps && (
                <details className="text-xs text-gray-500 mt-1">
                  <summary className="cursor-pointer hover:text-gray-700">
                    AI 编排步骤
                  </summary>
                  <pre className="mt-1 p-2 bg-gray-50 rounded overflow-x-auto whitespace-pre-wrap">
                    {result.steps}
                  </pre>
                </details>
              )}
            </div>
          )}

          {!result.success && result.error && (
            <p className="text-sm text-red-600">{result.error}</p>
          )}

          {result.success && (
            <button
              onClick={shareOnTwitter}
              className="mt-2 px-3 py-1.5 text-xs bg-black text-white rounded-lg hover:bg-gray-800 transition-colors flex items-center gap-1"
            >
              <span>𝕏</span> 分享到 Twitter
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
