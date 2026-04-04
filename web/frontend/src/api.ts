const BASE = '/api';

export interface Target {
  url: string;
  method: string;
  headers?: Record<string, string>;
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
  mappings: Mapping[];
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
  error?: string;
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
  listFlows: () => request<Flow[]>('/flows'),
  getFlow: (id: string) => request<Flow>(`/flows/${id}`),
  createFlow: (f: Partial<Flow>) => request<Flow>('/flows', { method: 'POST', body: JSON.stringify(f) }),
  updateFlow: (id: string, f: Partial<Flow>) => request<Flow>(`/flows/${id}`, { method: 'PUT', body: JSON.stringify(f) }),
  deleteFlow: (id: string) => request<void>(`/flows/${id}`, { method: 'DELETE' }),
  listLogs: (flowId: string) => request<ExecutionLog[]>(`/flows/${flowId}/logs`),
};
