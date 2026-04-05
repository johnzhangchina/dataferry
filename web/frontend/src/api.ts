const BASE = '/api';

export interface Target {
  url: string;
  method: string;
  headers?: Record<string, string>;
  timeout?: number;
  retry_count?: number;
  retry_delay?: number;
}

export interface WebhookConfig {
  secret?: string;
}

export interface Condition {
  field: string;
  operator: string;
  value?: string;
}

export interface Mapping {
  source?: string;
  target: string;
  transform?: string;
  value?: string;
}

export interface Flow {
  id: string;
  name: string;
  description?: string;
  webhook_path: string;
  target: Target;
  conditions?: Condition[];
  condition_logic?: string;
  mappings: Mapping[];
  webhook_config?: WebhookConfig;
  source_example?: string;
  target_example?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ExecutionLog {
  id: string;
  flow_id: string;
  received_at: string;
  source_payload: string;
  mapped_payload: string;
  target_url: string;
  response_status: number;
  response_body?: string;
  retry_attempts?: number;
  error?: string;
}

export interface FlowWithStatus extends Flow {
  last_log?: ExecutionLog | null;
}

export interface LogsResponse {
  logs: ExecutionLog[];
  total: number;
  page: number;
  size: number;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  listFlows: () => request<FlowWithStatus[]>('/flows'),
  getFlow: (id: string) => request<Flow>(`/flows/${id}`),
  createFlow: (f: Partial<Flow>) => request<Flow>('/flows', { method: 'POST', body: JSON.stringify(f) }),
  updateFlow: (id: string, f: Partial<Flow>) => request<Flow>(`/flows/${id}`, { method: 'PUT', body: JSON.stringify(f) }),
  deleteFlow: (id: string) => request<void>(`/flows/${id}`, { method: 'DELETE' }),
  listLogs: (flowId: string, page = 1, size = 20, status = '') =>
    request<LogsResponse>(`/flows/${flowId}/logs?page=${page}&size=${size}&status=${status}`),
  retryLog: (flowId: string, logId: string) =>
    request<{ status: string; target_status: number; execution_id: string }>(
      `/flows/${flowId}/logs/${logId}/retry`, { method: 'POST' }),
};
