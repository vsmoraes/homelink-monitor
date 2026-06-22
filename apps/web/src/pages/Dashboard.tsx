import { Alert, Button, Card, Col, Progress, Row, Select, Space, Statistic, Tag, Typography, message } from 'antd';
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Area, AreaChart, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { DNSCheck, LatencyCheck, RouterTrafficCurrent, RouterTrafficSample, SpeedTest, Summary } from '../types';
import { localTime, mbps, ms, number } from '../utils/format';
import { statusColor, statusText } from '../utils/status';

const latencyColors = ['#18c98f', '#13b8c8'];
const dnsColors = ['#18c98f', '#13b8c8'];
const seriesKey = (value: string) => value.replace(/[^a-zA-Z0-9]/g, '_');
const routerRefreshOptions = [
  { label: 'Historical', value: 'historical' },
  { label: '2 seconds', value: 'live-2000' },
  { label: '5 seconds', value: 'live-5000' },
  { label: '10 seconds', value: 'live-10000' },
  { label: '30 seconds', value: 'live-30000' },
];
const pointMbps = (value?: number) => (value === undefined ? undefined : (value * 8) / 1_000_000);
const plainNumber = (value?: number) => (value === undefined || value === null ? '—' : number(value));
const routerInterval = (value: string) => Number(value.replace('live-', ''));

export default function Dashboard() {
  const [summary, setSummary] = useState<Summary>();
  const [speedTests, setSpeedTests] = useState<SpeedTest[]>([]);
  const [latencyChecks, setLatencyChecks] = useState<LatencyCheck[]>([]);
  const [dnsChecks, setDnsChecks] = useState<DNSCheck[]>([]);
  const [routerTraffic, setRouterTraffic] = useState<RouterTrafficSample[]>([]);
  const [routerCurrent, setRouterCurrent] = useState<RouterTrafficCurrent>();
  const [routerLive, setRouterLive] = useState<{ time: string; download?: number; upload?: number }[]>([]);
  const [routerChartMode, setRouterChartMode] = useState('historical');
  const [hiddenSpeedSeries, setHiddenSpeedSeries] = useState<string[]>([]);
  const [hiddenRouterSeries, setHiddenRouterSeries] = useState<string[]>([]);
  const [hiddenLatencySeries, setHiddenLatencySeries] = useState<string[]>([]);
  const [hiddenDNSSeries, setHiddenDNSSeries] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [startingSpeedTest, setStartingSpeedTest] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = async (showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      const [nextSummary, speedHistory, latencyHistory, dnsHistory, routerHistory, routerNow] = await Promise.all([
        api.summary(),
        api.speedTests(),
        api.latency(),
        api.dnsChecks(),
        api.routerTraffic(),
        api.currentRouterTraffic(),
      ]);
      setSummary(nextSummary);
      setSpeedTests(speedHistory.items);
      setLatencyChecks(latencyHistory.items);
      setDnsChecks(dnsHistory.items);
      setRouterTraffic(routerHistory.items);
      setRouterCurrent(routerNow);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(false), summary?.speedTestIsRunning ? 3000 : 30000);
    return () => window.clearInterval(id);
  }, [summary?.speedTestIsRunning]);

  useEffect(() => {
    if (routerChartMode === 'historical') return undefined;
    let stopped = false;
    let intervalID: number | undefined;
    const tick = async () => {
      if (stopped) return;
      try {
        const next = await api.probeRouterTraffic();
        if (stopped) return;
        setRouterCurrent(next);
        if (next.capability.error || next.latest?.success === false) return;
        setRouterLive((points) => [
          ...points.slice(-119),
          {
            time: new Date().toLocaleTimeString(),
            download: pointMbps(next.latest?.downloadBps),
            upload: pointMbps(next.latest?.uploadBps),
          },
        ]);
      } catch {
        // Dashboard keeps the last known router state; the Router page shows detailed probe errors.
      }
    };
    void tick();
    intervalID = window.setInterval(() => void tick(), routerInterval(routerChartMode));
    return () => {
      stopped = true;
      if (intervalID !== undefined) window.clearInterval(intervalID);
    };
  }, [routerChartMode]);

  const runSpeedTest = async () => {
    setStartingSpeedTest(true);
    try {
      await api.runSpeedTest();
      message.success('Speed test started');
      await load(false);
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setStartingSpeedTest(false);
    }
  };

  const latencyRawPercent = summary?.latency24h.avgMs && summary.settings.maxLatencyMs
    ? Math.round((summary.latency24h.avgMs / summary.settings.maxLatencyMs) * 100)
    : 0;
  const latencyPercent = Math.min(100, latencyRawPercent);
  const downloadPercent = summary?.latestSpeedTest?.downloadMbps && summary.settings.minDownloadMbps
    ? Math.min(100, Math.round((summary.latestSpeedTest.downloadMbps / summary.settings.minDownloadMbps) * 100))
    : 0;
  const speedChart = [...speedTests]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({ time: localTime(item.startedAt), download: item.downloadMbps, upload: item.uploadMbps }));
  const latencyTargets = Array.from(new Set(latencyChecks.map((item) => item.target)));
  const allLatencyChart = [...latencyChecks]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({ time: localTime(item.checkedAt), [seriesKey(item.target)]: item.latencyMs }));
  const dnsDomains = Array.from(new Set(dnsChecks.map((item) => item.domain)));
  const allDNSChart = [...dnsChecks]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({ time: localTime(item.checkedAt), [seriesKey(item.domain)]: item.durationMs }));
  const routerUsageChart = [...routerTraffic]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({
      time: localTime(item.checkedAt),
      download: pointMbps(item.downloadBps),
      upload: pointMbps(item.uploadBps),
    }));
  const routerChart = routerChartMode === 'historical' ? routerUsageChart : routerLive;
  const failureCount = latencyChecks.filter((item) => !item.success).length
    + dnsChecks.filter((item) => !item.success).length
    + speedTests.filter((item) => !item.success).length;
  const toggleSeries = (key: string, hidden: string[], setHidden: (next: string[]) => void) => {
    setHidden(hidden.includes(key) ? hidden.filter((item) => item !== key) : [...hidden, key]);
  };

  return (
    <Page title="Dashboard" loading={loading} error={error} actions={<Button icon={<ReloadOutlined />} onClick={() => void load()}>Refresh</Button>}>
      {summary ? (
        <div className="dashboard-grid">
          {summary.speedTestIsRunning ? <Alert type="info" showIcon message="Speed test is running" description="Results will appear here automatically when the command finishes." /> : null}
          {summary.latestSpeedTest?.success === false ? <Alert type="warning" showIcon message={summary.latestSpeedTest.error} /> : null}
          <Row gutter={[16, 16]} align="top">
            <Col xs={24} xl={16}>
              <Card className="network-hero">
                <div className="hero-topline">
                  <div>
                    <Typography.Text type="secondary">Connection state</Typography.Text>
                    <div className="hero-status">
                      <Tag color={statusColor(summary.status)}>{statusText(summary.status)}</Tag>
                      {summary.activeOutage ? <Typography.Text type="danger">Active outage</Typography.Text> : null}
                    </div>
                  </div>
                  <Button
                    type="primary"
                    icon={<ThunderboltOutlined />}
                    loading={startingSpeedTest || summary.speedTestIsRunning}
                    disabled={summary.speedTestIsRunning}
                    onClick={runSpeedTest}
                  >
                    {summary.speedTestIsRunning ? 'Running Speed Test' : 'Run Speed Test'}
                  </Button>
                </div>
                <Row gutter={[16, 16]} className="overview-strip">
                  <Col xs={12} lg={6}><Statistic title="Download" value={mbps(summary.latestSpeedTest?.downloadMbps)} /></Col>
                  <Col xs={12} lg={6}><Statistic title="Upload" value={mbps(summary.latestSpeedTest?.uploadMbps)} /></Col>
                  <Col xs={12} lg={6}><Statistic title="Ping" value={ms(summary.latestSpeedTest?.pingMs ?? summary.latestLatency?.latencyMs)} /></Col>
                  <Col xs={12} lg={6}><Statistic title="DNS" value={summary.latestDnsCheck?.success ? 'Healthy' : 'Failed'} /></Col>
                </Row>
                <Row gutter={[16, 16]} className="check-volume-strip">
                  <Col xs={12} md={6}><Statistic title="Latency Checks" value={latencyChecks.length} /></Col>
                  <Col xs={12} md={6}><Statistic title="DNS Checks" value={dnsChecks.length} /></Col>
                  <Col xs={12} md={6}><Statistic title="Speed Tests" value={speedTests.length} /></Col>
                  <Col xs={12} md={6}><Statistic title="Failures" value={failureCount} /></Col>
                </Row>
              </Card>
            </Col>
            <Col xs={24} xl={8}>
              <Card className="quality-panel">
                <Space direction="vertical" size="middle" className="full-width">
                  <div>
                    <Typography.Text type="secondary">Download target</Typography.Text>
                    <Progress percent={downloadPercent} status={downloadPercent >= 100 ? 'success' : 'normal'} />
                  </div>
                  <div>
                    <Typography.Text type="secondary">Latency budget</Typography.Text>
                    <Progress percent={latencyPercent} status={latencyRawPercent > 100 ? 'exception' : 'normal'} />
                  </div>
                  <div className="quality-meta">
                    <span>Last speed test</span>
                    <strong>{localTime(summary.latestSpeedTest?.startedAt)}</strong>
                  </div>
                  <div className="quality-meta">
                    <span>Outages 24h</span>
                    <strong>{summary.outageCount24h}</strong>
                  </div>
                  <div className="quality-meta">
                    <span>Packet loss 24h</span>
                    <strong>{number(summary.latency24h.packetLoss)}%</strong>
                  </div>
                </Space>
              </Card>
            </Col>
          </Row>
          <Row gutter={[16, 16]}>
            <Col xs={24} xl={12}>
              <Card title="Recent Trends" className="trend-card">
                <div className="trend-section">
                  <div className="trend-heading">
                    <Typography.Text strong>Speed</Typography.Text>
                    <Typography.Text type="secondary">
                      Down {mbps(summary.minDownload24hMbps)} - {mbps(summary.maxDownload24hMbps)} · Up {mbps(summary.minUpload24hMbps)} - {mbps(summary.maxUpload24hMbps)}
                    </Typography.Text>
                  </div>
                  <Row gutter={[16, 16]} className="overview-strip">
                    <Col xs={12} md={8}><Statistic title="Download Mbps" value={plainNumber(summary.latestSpeedTest?.downloadMbps)} /></Col>
                    <Col xs={12} md={8}><Statistic title="Upload Mbps" value={plainNumber(summary.latestSpeedTest?.uploadMbps)} /></Col>
                    <Col xs={12} md={8}><Statistic title="Ping ms" value={plainNumber(summary.latestSpeedTest?.pingMs)} /></Col>
                  </Row>
                  <ResponsiveContainer width="100%" height={220}>
                  <AreaChart data={speedChart}>
                    <XAxis dataKey="time" hide />
                    <YAxis />
                    <Tooltip />
                    <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenSpeedSeries, setHiddenSpeedSeries)} />
                    <Area type="monotone" dataKey="download" stroke="#18c98f" fill="#dffbea" name="Download" hide={hiddenSpeedSeries.includes('download')} />
                    <Area type="monotone" dataKey="upload" stroke="#13b8c8" fill="#dff8fb" name="Upload" hide={hiddenSpeedSeries.includes('upload')} />
                  </AreaChart>
                </ResponsiveContainer>
                </div>
              </Card>
            </Col>
            <Col xs={24} xl={12}>
              <Card
                title="Router Usage"
                extra={<Select value={routerChartMode} onChange={setRouterChartMode} options={routerRefreshOptions} size="small" />}
              >
                <div className="trend-heading">
                  <Typography.Text type="secondary">
                    {routerCurrent?.latest?.success
                      ? `${routerCurrent.latest.clientCount} clients · ${localTime(routerCurrent.latest.checkedAt)}`
                      : routerCurrent?.capability.error ?? 'Enable TP-Link traffic monitoring in Settings'}
                  </Typography.Text>
                </div>
                <Row gutter={[16, 16]} className="overview-strip">
                  <Col xs={24} md={8}><Statistic title="Clients" value={routerCurrent?.latest?.clientCount ?? 0} /></Col>
                  <Col xs={12} md={8}><Statistic title="Download Mbps" value={routerCurrent?.capability.downloadAvailable ? plainNumber(pointMbps(routerCurrent.latest?.downloadBps)) : 'Unavailable'} /></Col>
                  <Col xs={12} md={8}><Statistic title="Upload Mbps" value={routerCurrent?.capability.uploadAvailable ? plainNumber(pointMbps(routerCurrent.latest?.uploadBps)) : 'Unavailable'} /></Col>
                </Row>
                <ResponsiveContainer width="100%" height={220}>
                  <AreaChart data={routerChart}>
                    <XAxis dataKey="time" hide />
                    <YAxis />
                    <Tooltip />
                    <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenRouterSeries, setHiddenRouterSeries)} />
                    {routerCurrent?.capability.downloadAvailable ? <Area type="monotone" dataKey="download" stroke="#18c98f" fill="#dffbea" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} name="Download Mbps" connectNulls isAnimationActive={false} hide={hiddenRouterSeries.includes('download')} /> : null}
                    {routerCurrent?.capability.uploadAvailable ? <Area type="monotone" dataKey="upload" stroke="#13b8c8" fill="#dff8fb" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} name="Upload Mbps" connectNulls isAnimationActive={false} hide={hiddenRouterSeries.includes('upload')} /> : null}
                  </AreaChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>
          <Row gutter={[16, 16]}>
            <Col xs={24}>
              <Card title="Latency & DNS" className="trend-card">
                <Row gutter={[16, 16]}>
                  <Col xs={24} lg={12}>
                    <div className="trend-section compact">
                      <Typography.Text strong>Latency</Typography.Text>
                      <ResponsiveContainer width="100%" height={220}>
                        <LineChart data={allLatencyChart}>
                          <XAxis dataKey="time" hide />
                          <YAxis />
                          <Tooltip />
                          <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenLatencySeries, setHiddenLatencySeries)} />
                          {latencyTargets.map((target, index) => {
                            const key = seriesKey(target);
                            return (
                              <Line key={target} type="monotone" dataKey={key} stroke={latencyColors[index % latencyColors.length]} strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} name={target} connectNulls hide={hiddenLatencySeries.includes(key)} />
                            );
                          })}
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  </Col>
                  <Col xs={24} lg={12}>
                    <div className="trend-section compact">
                      <Typography.Text strong>DNS</Typography.Text>
                      <ResponsiveContainer width="100%" height={220}>
                        <LineChart data={allDNSChart}>
                          <XAxis dataKey="time" hide />
                          <YAxis />
                          <Tooltip />
                          <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenDNSSeries, setHiddenDNSSeries)} />
                          {dnsDomains.map((domain, index) => {
                            const key = seriesKey(domain);
                            return (
                              <Line key={domain} type="monotone" dataKey={key} stroke={dnsColors[index % dnsColors.length]} strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} name={domain} connectNulls hide={hiddenDNSSeries.includes(key)} />
                            );
                          })}
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  </Col>
                </Row>
              </Card>
            </Col>
          </Row>
        </div>
      ) : null}
    </Page>
  );
}
