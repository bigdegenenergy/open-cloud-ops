// Cerebra API client for the Open Cloud Ops dashboard.
// All endpoints are proxied through Vite dev server to localhost:8080.

const BASE = '';

export interface CostSummaryItem {
  dimension: string;
  key: string;
  total_cost_usd: number;
  request_count: number;
}

export interface CostRequest {
  id: string;
  timestamp: string;
  model: string;
  provider: string;
  team: string;
  input_tokens: number;
  output_tokens: number;
  cost_usd: number;
  latency_ms: number;
  status: string;
}

export interface Budget {
  scope: string;
  entity_id: string;
  limit_usd: number;
  spent_usd: number;
  period_days: number;
  utilization_pct: number;
}

export interface CreateBudgetPayload {
  scope: string;
  entity_id: string;
  limit_usd: number;
  period_days: number;
}

export interface Insight {
  id: string;
  severity: 'info' | 'warning' | 'critical';
  category: string;
  title: string;
  description: string;
  recommendation: string;
  created_at: string;
}

export interface ReportSummary {
  total_cost_usd: number;
  total_requests: number;
  avg_latency_ms: number;
  savings_usd: number;
  period_start: string;
  period_end: string;
}

export interface HealthStatus {
  status: string;
  version: string;
  uptime_seconds: number;
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => 'Unknown error');
    throw new Error(`API ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

export function getHealth(): Promise<HealthStatus> {
  return request<HealthStatus>('/health');
}

export function getCostSummary(
  dimension: 'model' | 'provider' | 'team',
  from: string,
  to: string,
): Promise<CostSummaryItem[]> {
  const params = new URLSearchParams({ dimension, from, to });
  return request<CostSummaryItem[]>(`/api/v1/costs/summary?${params}`);
}

export function getRecentRequests(limit = 50): Promise<CostRequest[]> {
  return request<CostRequest[]>(`/api/v1/costs/requests?limit=${limit}`);
}

export function getBudgets(): Promise<Budget[]> {
  return request<Budget[]>('/api/v1/budgets');
}

export function getBudget(scope: string, entityId: string): Promise<Budget> {
  return request<Budget>(`/api/v1/budgets/${scope}/${entityId}`);
}

export function createBudget(payload: CreateBudgetPayload): Promise<Budget> {
  return request<Budget>('/api/v1/budgets', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function getInsights(): Promise<Insight[]> {
  return request<Insight[]>('/api/v1/insights');
}

export function getReport(): Promise<ReportSummary> {
  return request<ReportSummary>('/api/v1/report');
}
