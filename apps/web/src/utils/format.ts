export function number(value?: number, digits = 1): string {
  return value === undefined || value === null ? '—' : value.toFixed(digits);
}

export function mbps(value?: number): string {
  return value === undefined || value === null ? '—' : `${number(value)} Mbps`;
}

export function ms(value?: number): string {
  return value === undefined || value === null ? '—' : `${number(value)} ms`;
}

export function localTime(value?: string): string {
  if (!value) return '—';
  return new Date(value).toLocaleString();
}

export function duration(start: string, end?: string): string {
  const endTime = end ? new Date(end).getTime() : Date.now();
  const seconds = Math.max(0, Math.round((endTime - new Date(start).getTime()) / 1000));
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  return `${(minutes / 60).toFixed(1)}h`;
}
