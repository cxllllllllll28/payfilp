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
        params: [{ chainId: "0x1388" }], // Mantle Mainnet 5000
      });
    } catch {
      // 链不存在则添加
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
        <div className="flex items-center gap-2">
          {!isCorrectChain && (
            <button
              onClick={switchToMantle}
              className="px-3 py-1.5 text-sm bg-amber-500 text-white rounded-lg hover:bg-amber-600 transition-colors"
            >
              切换到 Mantle
            </button>
          )}
          <span className="flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-lg text-sm">
            <span
              className={`w-2 h-2 rounded-full ${isCorrectChain ? "bg-green-500" : "bg-amber-500"}`}
            />
            {account.slice(0, 6)}...{account.slice(-4)}
          </span>
        </div>
      ) : (
        <button
          onClick={connect}
          disabled={connecting}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors text-sm font-medium"
        >
          {connecting ? "连接中..." : "🦊 连接钱包"}
        </button>
      )}
    </div>
  );
}

export async function getWalletProvider() {
  if (!window.ethereum) return null;
  return new BrowserProvider(window.ethereum);
}
