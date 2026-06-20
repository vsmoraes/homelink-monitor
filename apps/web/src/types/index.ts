export type SpeedTest = {
  id: number;
  startedAt: string;
  finishedAt?: string;
  downloadMbps?: number;
  uploadMbps?: number;
  pingMs?: number;
  jitterMs?: number;
  serverName?: string;
  serverLocation?: string;
  success: boolean;
  error?: string;
};

export type LatencyCheck = {
  id: number;
  checkedAt: string;
  target: string;
  latencyMs?: number;
  success: boolean;
  error?: string;
};

export type DNSCheck = {
  id: number;
  checkedAt: string;
  domain: string;
  resolver?: string;
  durationMs?: number;
  success: boolean;
  error?: string;
};

export type Outage = {
  id: number;
  startedAt: string;
  endedAt?: string;
  reason: string;
};

export type Settings = {
  speedTestScheduleMinutes: number;
  speedTestCommand: string;
  latencyIntervalSeconds: number;
  dnsIntervalSeconds: number;
  latencyTargets: string[];
  dnsDomains: string[];
  routerIp: string;
  minDownloadMbps: number;
  minUploadMbps: number;
  maxLatencyMs: number;
  outageFailureThreshold: number;
  monitoringEnabled: boolean;
};

export type LatencySummary = {
  minMs?: number;
  avgMs?: number;
  maxMs?: number;
  packetLoss: number;
  count: number;
};

export type Summary = {
  status: 'healthy' | 'degraded' | 'down';
  latestSpeedTest?: SpeedTest;
  latestLatency?: LatencyCheck;
  latestDnsCheck?: DNSCheck;
  latency24h: LatencySummary;
  outageCount24h: number;
  activeOutage?: Outage;
  minDownload24hMbps?: number;
  maxDownload24hMbps?: number;
  minUpload24hMbps?: number;
  maxUpload24hMbps?: number;
  settings: Settings;
  speedTestIsRunning: boolean;
};

export type User = {
  id: number;
  username: string;
  role: string;
  createdAt: string;
  updatedAt: string;
};
