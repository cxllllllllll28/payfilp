import { useState, useEffect } from "react";
import { BrowserProvider } from "ethers";

declare global {
  interface Window {
    ethereum?: {
      isMetaMask?: boolean;
      request: (args: {
        method: string;
        params?: unknown[];
      }) => Promise<unknown>;
      on: (event: string, handler: (...args: unknown[]) => void) => void;
      removeListener: (
        event: string,
        handler: (...args: unknown[]) => void,
      ) => void;
    };
  }
}

interface WalletConnectProps {
  onAccountChange?: (account: string) => void;
}

export function WalletConnect({ onAccountChange }: WalletConnectProps) {
  const [account, setAccount] = useState<string | null>(null);
  const [connecting, setConnecting] = useState(false);
  const [chainId, setChainId] = useState<number | null>(null);
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);

  useEffect(() => {
    if (!window.ethereum) return;
    const handleAccountsChanged = (accounts: unknown) => {
      const accs = accounts as string[];
      if (accs.length === 0) {
        setAccount(null);
        onAccountChange?.("");
      } else {
        setAccount(accs[0]);
        onAccountChange?.(accs[0]);
      }
    };
    const handleChainChanged = (chainIdHex: unknown) => {
      setChainId(Number(chainIdHex as string));
    };
    window.ethereum.on("accountsChanged", handleAccountsChanged);
    window.ethereum.on("chainChanged", handleChainChanged);
    return () => {
      window.ethereum?.removeListener("accountsChanged", handleAccountsChanged);
      window.ethereum?.removeListener("chainChanged", handleChainChanged);
    };
  }, [onAccountChange]);

  const connect = async () => {
    if (!window.ethereum) {
      alert("请安装 MetaMask 钱包");
      return;
    }
    setConnecting(true);
    try {
      const accounts = (await window.ethereum.request({
        method: "eth_requestAccounts",
      })) as string[];
      setAccount(accounts[0]);
      onAccountChange?.(accounts[0]);
      const netId = await window.ethereum.request({ method: "eth_chainId" });
      setChainId(Number(netId as string));
    } catch (err) {
      console.error("连接钱包失败:", err);
    } finally {
      setConnecting(false);
    }
  };

  const switchToMantle = async () => {
    if (!window.ethereum) return;
    try {
      await window.ethereum.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: "0x1388" }],
      });
    } catch {
      await window.ethereum.request({
        method: "wallet_addEthereumChain",
        params: [
          {
            chainId: "0x1388",
            chainName: "Mantle",
            nativeCurrency: { name: "MNT", symbol: "MNT", decimals: 18 },
            rpcUrls: ["https://rpc.mantle.xyz"],
            blockExplorerUrls: ["https://mantlescan.io/"],
          },
        ],
      });
    }
  };

  const isCorrectChain = chainId === 5000;

  return (
    <div className="flex items-center gap-3">
      {account ? (
        <div className="flex items-center gap-2 relative">
          {!isCorrectChain && (
            <button
              onClick={switchToMantle}
              className="px-3 py-1.5 text-xs font-medium bg-amber-500/15 text-amber-400 rounded-lg hover:bg-amber-500/25 transition-all border border-amber-500/20 whitespace-nowrap"
            >
              ⚠️ 切换 Mantle
            </button>
          )}
          <button
            onClick={() => setAccountMenuOpen(!accountMenuOpen)}
            className={`
              flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-sm font-medium
              transition-all duration-200 border
              ${
                isCorrectChain
                  ? "bg-surface-800/60 text-surface-200 border-white/5 hover:border-brand-500/30 hover:bg-surface-800/80"
                  : "bg-amber-500/10 text-amber-300 border-amber-500/20"
              }
            `}
          >
            <span
              className={`w-2 h-2 rounded-full ${isCorrectChain ? "bg-green-400 shadow-sm shadow-green-400/50" : "bg-amber-400"}`}
            />
            {account.slice(0, 6)}...{account.slice(-4)}
          </button>
          {accountMenuOpen && (
            <>
              <div
                className="fixed inset-0 z-40"
                onClick={() => setAccountMenuOpen(false)}
              />
              <div className="absolute right-0 top-full mt-2 z-50 w-64 glass-card rounded-xl p-3 shadow-2xl shadow-black/40 animate-fade-in-up">
                <div className="text-xs text-surface-400 mb-2">钱包地址</div>
                <code className="block text-xs font-mono text-surface-200 bg-surface-800/60 rounded-lg px-3 py-2 break-all">
                  {account}
                </code>
                <div className="mt-3 flex items-center gap-2 text-xs text-surface-400">
                  <span
                    className={`w-1.5 h-1.5 rounded-full ${isCorrectChain ? "bg-green-400" : "bg-amber-400"}`}
                  />
                  {isCorrectChain ? "Mantle 主网" : "未知网络"}
                </div>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(account);
                    setAccountMenuOpen(false);
                  }}
                  className="mt-3 w-full px-3 py-2 text-xs font-medium text-surface-300 bg-surface-800/60 rounded-lg hover:bg-surface-700/60 transition-colors"
                >
                  复制地址
                </button>
              </div>
            </>
          )}
        </div>
      ) : (
        <button
          onClick={connect}
          disabled={connecting}
          className="group relative px-5 py-2 rounded-lg text-sm font-semibold text-white overflow-hidden transition-all duration-200"
        >
          <span className="absolute inset-0 bg-gradient-to-r from-brand-600 to-mantle-600 rounded-lg" />
          <span className="absolute inset-0 bg-gradient-to-r from-brand-500 to-mantle-500 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
          <span className="relative z-10 flex items-center gap-2">
            {connecting ? (
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
                连接中...
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
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
                连接钱包
              </>
            )}
          </span>
        </button>
      )}
    </div>
  );
}

export async function getWalletProvider() {
  if (!window.ethereum) return null;
  return new BrowserProvider(window.ethereum);
}
