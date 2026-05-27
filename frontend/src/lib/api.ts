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
  return res.json();
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
