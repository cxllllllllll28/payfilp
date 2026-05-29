const API_BASE = "/api";

export interface IntentRequest {
  input: string;
  walletPk: string;
}

export interface IntentResponse {
  success: boolean;
  txHash?: string;
  explorerUrl?: string;
  error?: string;
  steps?: string;
  mode?: string;
  managed?: boolean;
}

export interface YieldPool {
  pool: string;
  project: string;
  symbol: string;
  apy: number;
  tvlUsd: number;
}

export interface YieldResponse {
  pools: YieldPool[];
  updatedAt: string;
}

export interface RebalanceRequest {
  walletPk: string;
  strategy: "highest_apy" | "balanced";
}

export interface RebalanceResponse {
  success: boolean;
  txHash?: string;
  explorerUrl?: string;
  error?: string;
  recommendation?: string;
}

export async function executeIntent(
  req: IntentRequest,
): Promise<IntentResponse> {
  const res = await fetch(`${API_BASE}/intent/execute`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `后端错误(${res.status}): ${text || "后端未响应，请确认后端已启动"}`,
    );
  }
  const text = await res.text();
  if (!text) throw new Error("后端返回空响应，请确认后端已启动");
  return JSON.parse(text);
}

export async function fetchCurrentYields(): Promise<YieldResponse> {
  const res = await fetch(`${API_BASE}/yield/current`);
  return res.json();
}

export async function triggerRebalance(
  req: RebalanceRequest,
): Promise<RebalanceResponse> {
  const res = await fetch(`${API_BASE}/yield/rebalance`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return res.json();
}

export interface ManagedRegisterRequest {
  privateKey: string;
}

export interface ManagedRegisterResponse {
  success: boolean;
  address?: string;
  error?: string;
}

export async function registerManaged(
  req: ManagedRegisterRequest,
): Promise<ManagedRegisterResponse> {
  const res = await fetch(`${API_BASE}/managed/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return res.json();
}
