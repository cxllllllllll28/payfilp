interface ManagedPanelProps {
  privateKey: string;
  walletAddress: string;
}

export function ManagedPanel({ privateKey, walletAddress }: ManagedPanelProps) {
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

      {/* 私钥状态 — 复用顶层私钥，无需再次输入 */}
      {privateKey && walletAddress ? (
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
            <span className="font-medium">已连接钱包</span>
          </div>

          <div className="bg-surface-800/60 rounded-xl p-4 space-y-3 border border-white/5">
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">钱包地址</span>
              <code className="font-mono text-xs text-surface-200 bg-surface-800 px-2 py-0.5 rounded border border-white/5">
                {walletAddress.slice(0, 6)}...{walletAddress.slice(-4)}
              </code>
            </div>
            <div className="h-px bg-white/5" />
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">运行状态</span>
              <span className="flex items-center gap-1.5 text-xs text-success">
                <span className="w-1.5 h-1.5 rounded-full bg-success animate-pulse" />
                就绪 — 点击下方"自动调仓"按钮执行
              </span>
            </div>
            <div className="h-px bg-white/5" />
            <div className="flex justify-between items-center">
              <span className="text-xs text-surface-400">检查间隔</span>
              <span className="text-xs text-surface-300">每次手动触发</span>
            </div>
          </div>

          <div className="flex items-center gap-4 text-xs text-surface-500">
            <div className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-success/50" />
              私钥在顶层管理
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-brand-500/50" />
              后端签名执行
            </div>
          </div>
        </div>
      ) : (
        <div className="p-4 rounded-lg bg-amber-500/8 border border-amber-500/20 text-xs text-amber-300/80 space-y-2">
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
            需要先连接钱包
          </div>
          <p>
            请在顶栏输入钱包私钥，即可使用自动调仓功能。私钥仅用于本地签名，不会上传。
          </p>
        </div>
      )}
    </div>
  );
}
