import { Alert, Button, Card, Col, Progress, Row, Space, Statistic, Tag, Typography, message } from 'antd';
import { CloudServerOutlined, GlobalOutlined, ReloadOutlined, ThunderboltOutlined, WifiOutlined } from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { Summary } from '../types';
import { localTime, mbps, ms, number } from '../utils/format';
import { statusColor, statusText } from '../utils/status';

export default function Dashboard() {
  const [summary, setSummary] = useState<Summary>();
  const [loading, setLoading] = useState(true);
  const [startingSpeedTest, setStartingSpeedTest] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = async (showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      setSummary(await api.summary());
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

  return (
    <Page title="Dashboard" loading={loading} error={error} actions={<Button icon={<ReloadOutlined />} onClick={() => void load()}>Refresh</Button>}>
      {summary ? (
        <Space direction="vertical" size="large" className="full-width dashboard-grid">
          {summary.speedTestIsRunning ? <Alert type="info" showIcon message="Speed test is running" description="Results will appear here automatically when the command finishes." /> : null}
          {summary.latestSpeedTest?.success === false ? <Alert type="warning" showIcon message={summary.latestSpeedTest.error} /> : null}
          <Row gutter={[16, 16]} align="stretch">
            <Col xs={24} xl={14}>
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
                <div className="network-path">
                  <div className="network-node"><CloudServerOutlined /><span>NAS</span></div>
                  <div className="network-link" />
                  <div className="network-node"><WifiOutlined /><span>Router</span></div>
                  <div className="network-link" />
                  <div className="network-node"><GlobalOutlined /><span>Internet</span></div>
                </div>
                <Row gutter={[16, 16]}>
                  <Col xs={24} sm={8}><Statistic title="Download" value={mbps(summary.latestSpeedTest?.downloadMbps)} /></Col>
                  <Col xs={24} sm={8}><Statistic title="Upload" value={mbps(summary.latestSpeedTest?.uploadMbps)} /></Col>
                  <Col xs={24} sm={8}><Statistic title="Ping" value={ms(summary.latestSpeedTest?.pingMs ?? summary.latestLatency?.latencyMs)} /></Col>
                </Row>
              </Card>
            </Col>
            <Col xs={24} xl={10}>
              <Card className="quality-panel">
                <Space direction="vertical" size="large" className="full-width">
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
                </Space>
              </Card>
            </Col>
          </Row>
          <Row gutter={[16, 16]}>
            <Col xs={24} md={6}><Card className="metric-tile"><Statistic title="Average Latency 24h" value={ms(summary.latency24h.avgMs)} /></Card></Col>
            <Col xs={24} md={6}><Card className="metric-tile"><Statistic title="Packet Loss 24h" value={`${number(summary.latency24h.packetLoss)}%`} /></Card></Col>
            <Col xs={24} md={6}><Card className="metric-tile"><Statistic title="Outages 24h" value={summary.outageCount24h} /></Card></Col>
            <Col xs={24} md={6}><Card className="metric-tile"><Statistic title="DNS" value={summary.latestDnsCheck?.success ? 'Healthy' : 'Failed'} /></Card></Col>
          </Row>
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}><Card title="24h Download Range">{mbps(summary.minDownload24hMbps)} - {mbps(summary.maxDownload24hMbps)}</Card></Col>
            <Col xs={24} md={12}><Card title="24h Upload Range">{mbps(summary.minUpload24hMbps)} - {mbps(summary.maxUpload24hMbps)}</Card></Col>
          </Row>
        </Space>
      ) : null}
    </Page>
  );
}
