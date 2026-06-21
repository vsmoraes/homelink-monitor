import { Alert, Button, Card, Space, Table, Typography, message } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { SpeedTest, Summary } from '../types';
import { localTime, mbps, ms } from '../utils/format';

export default function SpeedTests() {
  const [items, setItems] = useState<SpeedTest[]>([]);
  const [summary, setSummary] = useState<Summary>();
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const load = async (showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      const [history, current] = await Promise.all([api.speedTests(), api.summary()]);
      setItems(history.items);
      setSummary(current);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    const id = window.setInterval(() => void load(false), summary?.speedTestIsRunning ? 3000 : 30000);
    return () => window.clearInterval(id);
  }, [summary?.speedTestIsRunning]);

  const run = async () => {
    setStarting(true);
    try {
      await api.runSpeedTest();
      message.success('Speed test started');
      await load(false);
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setStarting(false);
    }
  };
  const chart = [...items].reverse().map((item) => ({ time: localTime(item.startedAt), download: item.downloadMbps, upload: item.uploadMbps }));
  return (
    <Page
      title="Speed Tests"
      loading={loading}
      error={error}
      actions={
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          loading={starting || summary?.speedTestIsRunning}
          disabled={summary?.speedTestIsRunning}
          onClick={run}
        >
          {summary?.speedTestIsRunning ? 'Running Speed Test' : 'Run Speed Test'}
        </Button>
      }
    >
      <Space direction="vertical" size="large" className="full-width">
        {summary?.speedTestIsRunning ? (
          <Alert type="info" showIcon message="Speed test is running" description="This can take a minute or more. The table and chart refresh automatically." />
        ) : null}
        {summary?.latestSpeedTest ? (
          <Card>
            <Space direction="vertical" size="small">
              <Typography.Text strong>Latest result</Typography.Text>
              <Typography.Text>
                {summary.latestSpeedTest.success ? 'Success' : 'Failed'} at {localTime(summary.latestSpeedTest.startedAt)}
                {' '}· Download {mbps(summary.latestSpeedTest.downloadMbps)}
                {' '}· Upload {mbps(summary.latestSpeedTest.uploadMbps)}
                {' '}· Ping {ms(summary.latestSpeedTest.pingMs)}
              </Typography.Text>
              {!summary.latestSpeedTest.success && summary.latestSpeedTest.error ? (
                <Alert type="warning" showIcon message={summary.latestSpeedTest.error} />
              ) : null}
            </Space>
          </Card>
        ) : null}
        <Card>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chart}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="download" stroke="#1677ff" dot={false} />
              <Line type="monotone" dataKey="upload" stroke="#52c41a" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </Card>
        <Table
          rowKey="id"
          dataSource={items}
          scroll={{ x: 'max-content' }}
          pagination={{ showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100] }}
          columns={[
            { title: 'Started', dataIndex: 'startedAt', render: localTime },
            { title: 'Download', render: (_, r) => mbps(r.downloadMbps) },
            { title: 'Upload', render: (_, r) => mbps(r.uploadMbps) },
            { title: 'Ping', render: (_, r) => ms(r.pingMs) },
            { title: 'Result', render: (_, r) => <ResultTag success={r.success} /> },
            { title: 'Error', dataIndex: 'error' },
          ]}
        />
      </Space>
    </Page>
  );
}
