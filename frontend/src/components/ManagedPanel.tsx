import { useState } from "react";
import { registerManaged } from "../lib/api";

export function ManagedPanel() {
  const [pk, setPk] = useState("");
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(false);
  const [address, setAddress] = useState("");
  const [showPk, setShowPk] = useState(false);

  const handleRegister = async () => {
    if (!pk.trim()) return;
    setLoading(true);
    try {
      const res = await registerManaged({ privateKey: pk });
      if (res.address) {
        setAddress(res.address);
        setSaved(true);
        setPk("");
      }
    } catch (err) {
      console.error("注册失败:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleUnregister = () => {
    setSaved(false);
    setAddress("");
  };

  return (
    <div className="glass-card rounded-2xl p-6 space-y-5">
      {/* 头部 */}
      <div className="flex items-start gap-4">
        <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-brand-500/20 to-mantle-600/20 flex items-center justify-center shrink-0 border border-white/5">
          <svg
            className="w-5 h-5 text-brand-400"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <circle cx="12" cy="12" r="3" />
            <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
          </svg>
        </div>
        <div>
          <h3 className="text-lg font-semibold text-surface-100">自动调仓</h3>
          <p className="text-sm text-surface-400 mt-0.5">
            全天候监控收益，自动迁移到最高 APY 池
          </p>
        </div>
      </div>

      {!saved ? (
        <div className="space-y-4">
          <div className="p-3 rounded-lg bg-amber-500/8 border border-amber-500/20 text-xs text-amber-300/80 space-y-1">
            <div className="flex items-center gap-1.5 font-medium text-amber-300">
              <svg
                className="w-3.5 h-3.5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
              安全提示
            </div>
            <p>
              私钥仅存储在当前会话中，用于自动执行调仓交易，不会上传到任何服务器
            </p>
          </div>

          <div className="flex gap-2">
            <div className="relative flex-1">
              <input
                type={showPk ? "text" : "password"}
                value={pk}
                onChange={(e) => setPk(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleRegister();
                }}
                placeholder="输入钱包私钥"
                className="input-glow w-full px-4 py-3 rounded-xl bg-surface-800/80 border border-white/10 text-surface-100 placeholder-surface-500 focus:border-brand-500/50 outline-none text-sm font-mono transition-all duration-200 pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPk(!showPk)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-200 transition-colors"
              >
                {showPk ? (
                  <svg
                    className="w-4 h-4"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                  >
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                    <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                ) : (
                  <svg
                    className="w-4 h-4"
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
              onClick={handleRegister}
              disabled={loading || !pk.trim()}
              className="group relative px-6 py-3 rounded-xl text-sm font-semibold text-white overflow-hidden disabled:opacity-40 disabled:cursor-not-allowed transition-all duration-200 shrink-0"
            >
              <span className="absolute inset-0 bg-gradient-to-r from-brand-600 to-mantle-600 rounded-xl" />
              <span className="absolute inset-0 bg-gradient-to-r from-brand-500 to-mantle-500 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
              <span className="relative z-10 flex items-center gap-1.5">
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
                    注册中...
                  </>
                ) : (
                  "注册托管"
                )}
              </span>
            </button>
          </div>

          <div className="flex items-center gap-4 text-xs text-surface-500">
            <div className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-success/50" />
              30分钟检查间隔
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-brand-500/50" />
              自动调仓
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-mantle-400/50" />
              通知推送
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-4 animate-fade-in-up">
          <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-success/8 border border-success/20 text-success text-sm">
            <svg
              className="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <span className="font-medium">已注册托管</span>
          </div>

          <div className="bg-surface-800/60 rounded-xl p-4 space-y-3 border border-white/5">
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">钱包地址</span>
              <code className="font-mono text-xs text-surface-200 bg-surface-800 px-2 py-0.5 rounded border border-white/5">
                {address.slice(0, 6)}...{address.slice(-4)}
              </code>
            </div>
            <div className="h-px bg-white/5" />
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">运行状态</span>
              <span className="flex items-center gap-1.5 text-xs text-success">
                <span className="w-1.5 h-1.5 rounded-full bg-success animate-pulse" />
                运行中
              </span>
            </div>
            <div className="h-px bg-white/5" />
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">检查间隔</span>
              <span className="text-xs text-surface-300">30 分钟</span>
            </div>
          </div>

          <button
            onClick={handleUnregister}
            className="flex items-center gap-1.5 text-xs text-surface-400 hover:text-error transition-colors"
          >
            <svg
              className="w-3.5 h-3.5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
            取消托管
          </button>
        </div>
      )}
    </div>
  );
}
