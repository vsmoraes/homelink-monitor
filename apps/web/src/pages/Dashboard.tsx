import { Alert, Button, Card, Col, Progress, Row, Space, Statistic, Tag, Typography, message } from 'antd';
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Area, AreaChart, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { DNSCheck, LatencyCheck, SpeedTest, Summary } from '../types';
import { localTime, mbps, ms, number } from '../utils/format';
import { statusColor, statusText } from '../utils/status';

const latencyColors = ['#18c98f', '#13b8c8', '#7be84d', '#f5a524', '#597ef7', '#eb2f96'];
const dnsColors = ['#13b8c8', '#18c98f', '#7be84d', '#f5a524', '#597ef7', '#eb2f96'];
const seriesKey = (value: string) => value.replace(/[^a-zA-Z0-9]/g, '_');

export default function Dashboard() {
  const [summary, setSummary] = useState<Summary>();
  const [speedTests, setSpeedTests] = useState<SpeedTest[]>([]);
  const [latencyChecks, setLatencyChecks] = useState<LatencyCheck[]>([]);
  const [dnsChecks, setDnsChecks] = useState<DNSCheck[]>([]);
  const [loading, setLoading] = useState(true);
  const [startingSpeedTest, setStartingSpeedTest] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = async (showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      const [nextSummary, speedHistory, latencyHistory, dnsHistory] = await Promise.all([
        api.summary(),
        api.speedTests(),
        api.latency(),
        api.dnsChecks(),
      ]);
      setSummary(nextSummary);
      setSpeedTests(speedHistory.items);
      setLatencyChecks(latencyHistory.items);
      setDnsChecks(dnsHistory.items);
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
  const latencyChart = [...latencyChecks]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({ time: localTime(item.checkedAt), [seriesKey(item.target)]: item.latencyMs }));
  const dnsDomains = Array.from(new Set(dnsChecks.map((item) => item.domain)));
  const dnsChart = [...dnsChecks]
    .reverse()
    .filter((item) => item.success)
    .map((item) => ({ time: localTime(item.checkedAt), [seriesKey(item.domain)]: item.durationMs }));
  const failureCount = latencyChecks.filter((item) => !item.success).length
    + dnsChecks.filter((item) => !item.success).length
    + speedTests.filter((item) => !item.success).length;

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
            <Col xs={24} xl={16}>
              <Card title="Recent Trends" className="trend-card">
                <div className="trend-section">
                  <div className="trend-heading">
                    <Typography.Text strong>Speed</Typography.Text>
                    <Typography.Text type="secondary">
                      Down {mbps(summary.minDownload24hMbps)} - {mbps(summary.maxDownload24hMbps)} · Up {mbps(summary.minUpload24hMbps)} - {mbps(summary.maxUpload24hMbps)}
                    </Typography.Text>
                  </div>
                  <ResponsiveContainer width="100%" height={190}>
                  <AreaChart data={speedChart}>
                    <XAxis dataKey="time" hide />
                    <YAxis />
                    <Tooltip />
                    <Area type="monotone" dataKey="download" stroke="#18c98f" fill="#dffbea" name="Download" />
                    <Area type="monotone" dataKey="upload" stroke="#13b8c8" fill="#dff8fb" name="Upload" />
                  </AreaChart>
                </ResponsiveContainer>
                </div>
                <Row gutter={[16, 16]}>
                  <Col xs={24} lg={12}>
                    <div className="trend-section compact">
                      <div className="trend-heading">
                        <Typography.Text strong>Latency</Typography.Text>
                        <Typography.Text type="secondary">Avg {ms(summary.latency24h.avgMs)}</Typography.Text>
                      </div>
                      <ResponsiveContainer width="100%" height={150}>
                  <LineChart data={latencyChart}>
                    <XAxis dataKey="time" hide />
                    <YAxis />
                    <Tooltip />
                    {latencyTargets.map((target, index) => (
                      <Line key={target} type="monotone" dataKey={seriesKey(target)} stroke={latencyColors[index % latencyColors.length]} dot={false} name={target} connectNulls />
                    ))}
                  </LineChart>
                </ResponsiveContainer>
                    </div>
                  </Col>
                  <Col xs={24} lg={12}>
                    <div className="trend-section compact">
                      <div className="trend-heading">
                        <Typography.Text strong>DNS</Typography.Text>
                        <Typography.Text type="secondary">{dnsChecks.length} samples</Typography.Text>
                      </div>
                      <ResponsiveContainer width="100%" height={150}>
                  <LineChart data={dnsChart}>
                    <XAxis dataKey="time" hide />
                    <YAxis />
                    <Tooltip />
                    {dnsDomains.map((domain, index) => (
                      <Line key={domain} type="monotone" dataKey={seriesKey(domain)} stroke={dnsColors[index % dnsColors.length]} dot={false} name={domain} connectNulls />
                    ))}
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
