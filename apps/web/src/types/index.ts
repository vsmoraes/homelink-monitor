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
  routerTrafficEnabled: boolean;
  routerTrafficIntervalSeconds: number;
  routerTrafficUrl?: string;
  routerTrafficUsername?: string;
  routerTrafficPassword?: string;
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

export type RouterTrafficSample = {
  id: number;
  checkedAt: string;
  provider: string;
  success: boolean;
  error?: string;
  clientCount: number;
  downloadBps?: number;
  uploadBps?: number;
  totalBps?: number;
  downloadAvailable: boolean;
  uploadAvailable: boolean;
  totalTrafficAvailable: boolean;
};

export type RouterTrafficClient = {
  mac?: string;
  ip?: string;
  hostname?: string;
  connection?: string;
  downloadBps?: number;
  uploadBps?: number;
  totalBps?: number;
  downloadBytes?: number;
  uploadBytes?: number;
  totalBytes?: number;
};

export type RouterTrafficCapability = {
  provider: string;
  checkedAt: string;
  reachable: boolean;
  authenticated: boolean;
  clientListAvailable: boolean;
  downloadAvailable: boolean;
  uploadAvailable: boolean;
  totalTrafficAvailable: boolean;
  error?: string;
  sources?: string[];
};

export type RouterTrafficCurrent = {
  capability: RouterTrafficCapability;
  latest?: RouterTrafficSample;
  clients: RouterTrafficClient[];
};

export type User = {
  id: number;
  username: string;
  role: string;
  createdAt: string;
  updatedAt: string;
};
