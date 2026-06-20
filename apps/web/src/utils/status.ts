import type { Summary } from '../types';

export function statusColor(status: Summary['status']): 'success' | 'warning' | 'error' {
  if (status === 'healthy') return 'success';
  if (status === 'degraded') return 'warning';
  return 'error';
}

export function statusText(status: Summary['status']): string {
  if (status === 'healthy') return 'Healthy';
  if (status === 'degraded') return 'Degraded';
  return 'Down';
}
