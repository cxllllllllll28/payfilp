import { useState, useRef, useEffect } from "react";
import { Wallet } from "ethers";
import { IntentInput } from "./components/IntentInput";
import { YieldDashboard } from "./components/YieldDashboard";
import { TxReceipt } from "./components/TxReceipt";
import type { IntentResponse } from "./lib/api";

function App() {
  const [intentResult, setIntentResult] = useState<IntentResponse | null>(null);
  const [pendingTx, setPendingTx] = useState<string | null>(null);
  const [scrollY, setScrollY] = useState(0);
  const mainRef = useRef<HTMLDivElement>(null);
  const [privateKey, setPrivateKey] = useState("");
  const [walletAddress, setWalletAddress] = useState("");
  const [showPkInput, setShowPkInput] = useState(true);
  const [showPk, setShowPk] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrollY(window.scrollY);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const handlePending = (txHash: string) => {
    setPendingTx(txHash);
    setIntentResult(null);
  };
  const handleResult = (r: IntentResponse | null) => {
    setIntentResult(r);
    if (r !== null) setPendingTx(null);
  };

  return (
    <div className="min-h-screen relative bg-grid">
      {/* 背景光晕 */}
      <div className="fixed inset-0 pointer-events-none z-0">
        <div className="absolute top-[-20%] left-[-10%] w-[60%] h-[60%] rounded-full bg-brand-500/5 blur-[120px]" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] rounded-full bg-mantle-400/5 blur-[100px]" />
      </div>

      {/* 顶栏 */}
      <header
        className={`sticky top-0 z-50 transition-all duration-300 ${
          scrollY > 10
            ? "bg-surface-950/80 backdrop-blur-xl border-b border-white/5 shadow-lg shadow-black/20"
            : "bg-transparent"
        }`}
      >
        <div className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-4">
            {/* Logo */}
            <div className="flex items-center gap-2.5">
              <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center text-white text-base font-extrabold shadow-lg shadow-brand-500/25">
                P
              </div>
              <div>
                <span className="text-lg font-bold text-white tracking-tight">
                  PayFlip
                </span>
                <span className="text-[10px] ml-2 px-2 py-0.5 bg-brand-500/15 text-brand-300 rounded-md font-medium align-middle">
                  Mantle
                </span>
              </div>
            </div>
          </div>

          {/* 私钥输入框（替换 WalletConnect） */}
          {showPkInput ? (
            <div className="flex items-center gap-2">
              <div className="relative">
                <input
                  type={showPk ? "text" : "password"}
                  value={privateKey}
                  onChange={(e) => setPrivateKey(e.target.value)}
                  placeholder="输入私钥 (或 0x...) 进行链上操作"
                  className="w-[280px] px-3 py-1.5 text-xs font-mono rounded-lg bg-surface-800/80 border border-white/10 text-surface-100 placeholder-surface-500 focus:border-brand-500/50 outline-none transition-all"
                />
                <button
                  type="button"
                  onClick={() => setShowPk(!showPk)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-200"
                >
                  {showPk ? (
                    <svg
                      className="w-3.5 h-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19M1 1l22 22" />
                      <line x1="1" y1="1" x2="23" y2="23" />
                    </svg>
                  ) : (
                    <svg
                      className="w-3.5 h-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                      <circle cx="12" cy="12" r="3" />
                    </svg>
                  )}
                </button>
              </div>
              <button
                onClick={() => {
                  const pk = privateKey.trim();
                  if (!pk) return;
                  try {
                    const w = pk.startsWith("0x")
                      ? new Wallet(pk)
                      : new Wallet("0x" + pk);
                    setWalletAddress(w.address);
                    setShowPkInput(false);
                  } catch {
                    alert("私钥格式错误");
                  }
                }}
                disabled={!privateKey.trim()}
                className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-gradient-to-r from-brand-600 to-mantle-600 text-white disabled:opacity-40 transition-all hover:from-brand-500 hover:to-mantle-500"
              >
                连接
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-success/10 border border-success/20 text-xs">
                <span className="w-1.5 h-1.5 rounded-full bg-success animate-pulse" />
                <span className="text-surface-300 font-mono">
                  {walletAddress.slice(0, 6)}...{walletAddress.slice(-4)}
                </span>
              </div>
              <button
                onClick={() => {
                  setPrivateKey("");
                  setWalletAddress("");
                  setShowPkInput(true);
                }}
                className="text-xs text-surface-400 hover:text-error transition-colors"
              >
                断开
              </button>
            </div>
          )}
        </div>
      </header>

      <main
        ref={mainRef}
        className="relative z-10 max-w-5xl mx-auto px-6 py-8 space-y-6"
      >
        {/* 标题区域 */}
        <div className="text-center pt-4 pb-2 animate-fade-in-up">
          <h1 className="text-3xl md:text-4xl font-extrabold bg-gradient-to-r from-white via-brand-200 to-mantle-300 bg-clip-text text-transparent">
            AI DeFi 副驾驶
          </h1>
          <p className="mt-2 text-surface-400 text-sm max-w-md mx-auto">
            用自然语言驱动你在 Mantle 上的所有 DeFi 操作
          </p>
        </div>

        {/* 模式提示 — AI 自动判断是否需要长期监控 */}
        <div className="flex justify-center">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-surface-800/60 backdrop-blur-sm border border-white/5 text-xs text-surface-400">
            <svg
              className="w-3.5 h-3.5 text-brand-400"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
            <span>AI 自动判断：单次操作 or 质押到最高收益池（自动调仓）</span>
          </div>
        </div>

        {/* Pending 条 */}
        {pendingTx && (
          <div className="glass-card rounded-xl px-5 py-3.5 flex items-center gap-3 animate-fade-in-up border-amber-500/20">
            <svg
              className="animate-spin h-4 w-4 text-amber-400 shrink-0"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
              />
            </svg>
            <div className="flex-1 min-w-0">
              <span className="text-sm text-amber-300">交易确认中</span>
              <span className="ml-2 font-mono text-xs text-amber-400/70">
                {pendingTx.slice(0, 8)}…{pendingTx.slice(-6)}
              </span>
            </div>
            <a
              href={`https://explorer.sepolia.mantle.xyz/tx/${pendingTx}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-brand-400 hover:text-brand-300 underline shrink-0"
            >
              查看
            </a>
          </div>
        )}

        {/* 私钥未连接时提示 */}
        {!walletAddress && (
          <div className="glass-card rounded-2xl p-8 text-center space-y-3 animate-fade-in-up">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-gradient-to-br from-brand-500/10 to-mantle-600/10 flex items-center justify-center border border-white/5">
              <svg
                className="w-7 h-7 text-surface-400"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
              >
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-surface-200">
              请连接钱包
            </h2>
            <p className="text-sm text-surface-400 max-w-sm mx-auto">
              在顶栏输入钱包私钥即可开始使用。私钥仅用于本地签名交易，不会上传。
            </p>
          </div>
        )}

        {/* 统一入口 — 自然语言输入 + 执行结果 */}
        {walletAddress && (
          <div className="glass-card rounded-2xl p-6 space-y-5 animate-fade-in-up">
            <IntentInput
              privateKey={privateKey}
              walletAddress={walletAddress}
              onResult={handleResult}
              onPending={handlePending}
            />
            <TxReceipt result={intentResult} />

            {/* 如果是托管模式，额外显示托管状态 */}
            {intentResult?.managed && (
              <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-brand-500/10 border border-brand-500/20 text-sm animate-fade-in-up">
                <svg
                  className="w-5 h-5 text-brand-400 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                >
                  <circle cx="12" cy="12" r="3" />
                  <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
                </svg>
                <span className="text-surface-200">
                  🤖 已自动注册钱包到 <strong>收益监控调度器</strong>
                  ，系统将持续监控最高 APY 并自动调仓
                </span>
              </div>
            )}

            {/* 收益池一览 — 始终显示，供参考 */}
            <div className="border-t border-white/5 pt-5">
              <YieldDashboard walletPk={privateKey} />
            </div>
          </div>
        )}

        {/* 底部 */}
        <footer className="text-center text-xs text-surface-500 py-8 space-y-2 border-t border-white/5 mt-10">
          <p>Powered by DeepSeek V4 Pro &amp; Mantle Network</p>
          <p className="space-x-3">
            <a
              href="https://mantlescan.io/"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-surface-300 transition-colors"
            >
              Mantle Explorer
            </a>
            <span className="text-surface-600">·</span>
            <a
              href="https://defillama.com/chain/Mantle"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-surface-300 transition-colors"
            >
              DefiLlama
            </a>
            <span className="text-surface-600">·</span>
            <a
              href="https://github.com"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-surface-300 transition-colors"
            >
              GitHub
            </a>
          </p>
        </footer>
      </main>
    </div>
  );
}

export default App;
