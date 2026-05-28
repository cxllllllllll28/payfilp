import { useState, useRef, useEffect } from "react";
import { WalletConnect } from "./components/WalletConnect";
import { IntentInput } from "./components/IntentInput";
import { YieldDashboard } from "./components/YieldDashboard";
import { TxReceipt } from "./components/TxReceipt";
import { ManagedPanel } from "./components/ManagedPanel";
import type { IntentResponse } from "./lib/api";

type Mode = "instant" | "managed";

function App() {
  const [mode, setMode] = useState<Mode>("instant");
  const [intentResult, setIntentResult] = useState<IntentResponse | null>(null);
  const [pendingTx, setPendingTx] = useState<string | null>(null);
  const [scrollY, setScrollY] = useState(0);
  const mainRef = useRef<HTMLDivElement>(null);

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
          <WalletConnect />
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

        {/* 模式切换 */}
        <div className="flex justify-center">
          <div className="inline-flex items-center gap-1 bg-surface-800/80 backdrop-blur-sm rounded-xl p-1 border border-white/5 shadow-lg">
            {(["instant", "managed"] as const).map((m) => (
              <button
                key={m}
                className={`relative px-5 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                  mode === m
                    ? "text-white"
                    : "text-surface-400 hover:text-surface-200"
                }`}
                onClick={() => setMode(m)}
              >
                {mode === m && (
                  <span className="absolute inset-0 rounded-lg bg-gradient-to-r from-brand-600 to-mantle-600 shadow-lg shadow-brand-500/20 animate-fade-in-up" />
                )}
                <span className="relative z-10 flex items-center gap-1.5">
                  {m === "instant" ? (
                    <>
                      <svg
                        className="w-4 h-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
                      </svg>
                      即时模式
                    </>
                  ) : (
                    <>
                      <svg
                        className="w-4 h-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <circle cx="12" cy="12" r="3" />
                        <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
                      </svg>
                      托管模式
                    </>
                  )}
                </span>
              </button>
            ))}
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

        {/* 即时模式 */}
        {mode === "instant" && (
          <div className="glass-card rounded-2xl p-6 space-y-5 animate-fade-in-up">
            <IntentInput onResult={handleResult} onPending={handlePending} />
            <TxReceipt result={intentResult} />
          </div>
        )}

        {/* 托管模式 */}
        {mode === "managed" && (
          <div className="space-y-5 animate-fade-in-up">
            <ManagedPanel />
            <div className="glass-card rounded-2xl p-6">
              <YieldDashboard />
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
