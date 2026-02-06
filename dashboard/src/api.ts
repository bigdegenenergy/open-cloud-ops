// Cerebra API client for the Open Cloud Ops dashboard.
// All endpoints are proxied through Vite dev server to localhost:8080.

const BASE = "";

export interface CostSummaryItem {
  dimension: string;
  dimension_id: string;
  dimension_name: string;
  total_cost_usd: number;
  total_requests: number;
  total_tokens: number;
  avg_latency_ms: number;
  total_savings_usd: number;
}

export interface CostRequest {
  id: string;
  timestamp: string;
  model: string;
  provider: string;
  agent_id: string;
  team_id: string;
  org_id: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cost_usd: number;
  latency_ms: number;
  status_code: number;
  was_routed: boolean;
  original_model: string;
  routed_model: string;
  savings_usd: number;
}

export interface Budget {
  id: string;
  scope: string;
  entity_id: string;
  limit_usd: number;
  spent_usd: number;
  period_days: number;
  created_at: string;
  updated_at: string;
}

export interface CreateBudgetPayload {
  scope: string;
  entity_id: string;
  limit_usd: number;
  period_days: number;
}

export interface Insight {
  id: string;
  type: string;
  severity: "info" | "warning" | "critical";
  title: string;
  description: string;
  estimated_saving: number;
  affected_entity: string;
  created_at: string;
  dismissed: boolean;
}

export interface ReportSummary {
  from: string;
  to: string;
  total_cost_usd: number;
  total_requests: number;
  total_tokens: number;
  avg_latency_ms: number;
  total_savings_usd: number;
}

export interface HealthStatus {
  status: string;
  service: string;
  version: string;
}

// Envelope types matching backend responses.
interface ListEnvelope<T> {
  count: number;
  data: T[];
}

interface CostSummaryEnvelope {
  dimension: string;
  from: string;
  to: string;
  data: CostSummaryItem[];
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { "Content-Type": "application/json" },
    signal: AbortSignal.timeout(30_000),
    ...options,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "Unknown error");
    throw new Error(`API ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

export function getHealth(): Promise<HealthStatus> {
  return request<HealthStatus>("/health");
}

export async function getCostSummary(
  dimension: "model" | "provider" | "team",
  from: string,
  to: string,
): Promise<CostSummaryItem[]> {
  const params = new URLSearchParams({ dimension, from, to });
  const envelope = await request<CostSummaryEnvelope>(
    `/api/v1/costs/summary?${params}`,
  );
  return envelope.data;
}

export async function getRecentRequests(limit = 50): Promise<CostRequest[]> {
  const envelope = await request<ListEnvelope<CostRequest>>(
    `/api/v1/costs/requests?limit=${limit}`,
  );
  return envelope.data;
}

export async function getBudgets(): Promise<Budget[]> {
  const envelope = await request<ListEnvelope<Budget>>("/api/v1/budgets");
  return envelope.data;
}

export function getBudget(scope: string, entityId: string): Promise<Budget> {
  return request<Budget>(`/api/v1/budgets/${scope}/${entityId}`);
}

export function createBudget(payload: CreateBudgetPayload): Promise<Budget> {
  return request<Budget>("/api/v1/budgets", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function getInsights(): Promise<Insight[]> {
  const envelope = await request<ListEnvelope<Insight>>("/api/v1/insights");
  return envelope.data;
}

export function getReport(): Promise<ReportSummary> {
  return request<ReportSummary>("/api/v1/report");
}
