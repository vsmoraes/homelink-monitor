import type { DNSCheck, LatencyCheck, LatencySummary, Outage, RouterTrafficCurrent, RouterTrafficSample, Settings, SpeedTest, Summary, User } from '../types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(body.error ?? response.statusText);
  }
  return response.json() as Promise<T>;
}

export const api = {
  me: () => request<{ user: User }>('/api/auth/me'),
  login: (username: string, password: string) => request<{ user: User }>('/api/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  logout: () => request<{ status: string }>('/api/auth/logout', { method: 'POST' }),
  summary: () => request<Summary>('/api/summary'),
  speedTests: () => request<{ items: SpeedTest[] }>('/api/speed-tests?limit=100'),
  runSpeedTest: () => request<{ status: string }>('/api/speed-tests/run', { method: 'POST' }),
  latency: (target = '') => request<{ items: LatencyCheck[] }>(`/api/latency?limit=300&target=${encodeURIComponent(target)}`),
  latencySummary: () => request<LatencySummary>('/api/latency/summary'),
  dnsChecks: () => request<{ items: DNSCheck[] }>('/api/dns-checks?limit=200'),
  routerTraffic: () => request<{ items: RouterTrafficSample[] }>('/api/router-traffic?limit=100'),
  currentRouterTraffic: () => request<RouterTrafficCurrent>('/api/router-traffic/current'),
  probeRouterTraffic: () => request<RouterTrafficCurrent>('/api/router-traffic/probe', { method: 'POST' }),
  outages: () => request<{ items: Outage[] }>('/api/outages?limit=200'),
  settings: () => request<Settings>('/api/settings'),
  saveSettings: (settings: Settings) => request<Settings>('/api/settings', { method: 'PUT', body: JSON.stringify(settings) }),
  users: () => request<{ items: User[] }>('/api/users'),
  createUser: (input: { username: string; password: string; role: string }) => request<User>('/api/users', { method: 'POST', body: JSON.stringify(input) }),
  updateUser: (id: number, input: { username: string; password?: string; role: string }) => request<User>(`/api/users/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteUser: (id: number) => request<{ status: string }>(`/api/users/${id}`, { method: 'DELETE' }),
};
