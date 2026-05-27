import { useState } from "react";
import { WalletConnect } from "./components/WalletConnect";
import { IntentInput } from "./components/IntentInput";
import { YieldDashboard } from "./components/YieldDashboard";
import { TxReceipt } from "./components/TxReceipt";
import type { IntentResponse } from "./lib/api";

type Mode = "instant" | "managed";

function App() {
  const [mode, setMode] = useState<Mode>("instant");
  const [walletPk, setWalletPk] = useState("");
  const [showPkInput, setShowPkInput] = useState(false);
  const [intentResult, setIntentResult] = useState<IntentResponse | null>(null);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-indigo-50">
      {/* 顶部导航 */}
      <header className="border-b border-gray-200 bg-white/80 backdrop-blur-sm sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">
              PayFlip
            </h1>
            <span className="text-xs px-2 py-0.5 bg-indigo-100 text-indigo-700 rounded-full font-medium">
              Mantle AI 副驾驶
            </span>
          </div>
          <WalletConnect />
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-4 py-6 space-y-6">
        {/* 模式切换 */}
        <div className="flex gap-2 bg-gray-100 p-1 rounded-xl w-fit">
          <button
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              mode === "instant"
                ? "bg-white text-indigo-700 shadow-sm"
                : "text-gray-500 hover:text-gray-700"
            }`}
            onClick={() => setMode("instant")}
          >
            ⚡ 即时模式
          </button>
          <button
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              mode === "managed"
                ? "bg-white text-indigo-700 shadow-sm"
                : "text-gray-500 hover:text-gray-700"
            }`}
            onClick={() => setMode("managed")}
          >
            🤖 托管模式
          </button>
        </div>

        {/* 钱包私钥输入（测试用） */}
        <div>
          <button
            onClick={() => setShowPkInput(!showPkInput)}
            className="text-xs text-gray-400 hover:text-gray-600 flex items-center gap-1"
          >
            {showPkInput ? "▼" : "▶"} 测试钱包私钥（可选）
          </button>
          {showPkInput && (
            <div className="mt-2">
              <input
                type="password"
                value={walletPk}
                onChange={(e) => setWalletPk(e.target.value)}
                placeholder="输入钱包私钥（测试用，不会存储）"
                className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg bg-white focus:ring-2 focus:ring-indigo-500 outline-none"
              />
              <p className="mt-1 text-xs text-gray-400">
                私钥仅用于本次会话，不会发送到前端服务器
              </p>
            </div>
          )}
        </div>

        {/* 即时模式：意图输入 */}
        {mode === "instant" && (
          <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100 space-y-4">
            <IntentInput walletPk={walletPk} onResult={setIntentResult} />
            <TxReceipt result={intentResult} />
          </div>
        )}

        {/* 托管模式：收益看板 */}
        {mode === "managed" && (
          <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
            <YieldDashboard walletPk={walletPk} />
          </div>
        )}

        {/* 底部信息 */}
        <footer className="text-center text-xs text-gray-400 py-4 space-y-1">
          <p>Powered by DeepSeek V4 Pro &amp; Mantle Network</p>
          <p>
            <a
              href="https://mantlescan.io/"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-gray-600"
            >
              Mantle Explorer
            </a>
            {" · "}
            <a
              href="https://defillama.com/chain/Mantle"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-gray-600"
            >
              DefiLlama
            </a>
          </p>
        </footer>
      </main>
    </div>
  );
}

export default App;
