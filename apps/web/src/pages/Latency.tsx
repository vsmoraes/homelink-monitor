import { Card, Input, Space, Table } from 'antd';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { LatencyCheck, LatencySummary } from '../types';
import { localTime, ms, number } from '../utils/format';

export default function Latency() {
  const [target, setTarget] = useState('');
  const [items, setItems] = useState<LatencyCheck[]>([]);
  const [summary, setSummary] = useState<LatencySummary>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const [latency, stats] = await Promise.all([api.latency(target), api.latencySummary()]);
        setItems(latency.items);
        setSummary(stats);
        setError(null);
      } catch (err) {
        setError(err as Error);
      } finally {
        setLoading(false);
      }
    };
    void load();
  }, [target]);
  const chart = [...items].reverse().filter((item) => item.success).map((item) => ({ time: localTime(item.checkedAt), latency: item.latencyMs }));
  return (
    <Page title="Latency" loading={loading} error={error} actions={<Input.Search placeholder="Filter target" allowClear onSearch={setTarget} className="target-search" />}>
      <Space direction="vertical" size="large" className="full-width">
        <Card title={`Min ${ms(summary?.minMs)}   Avg ${ms(summary?.avgMs)}   Max ${ms(summary?.maxMs)}   Loss ${number(summary?.packetLoss)}%`}>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chart}><XAxis dataKey="time" hide /><YAxis /><Tooltip /><Line type="monotone" dataKey="latency" stroke="#1677ff" dot={false} /></LineChart>
          </ResponsiveContainer>
        </Card>
        <Table rowKey="id" dataSource={items} columns={[
          { title: 'Checked', dataIndex: 'checkedAt', render: localTime },
          { title: 'Target', dataIndex: 'target' },
          { title: 'Latency', render: (_, r) => ms(r.latencyMs) },
          { title: 'Result', render: (_, r) => <ResultTag success={r.success} /> },
          { title: 'Error', dataIndex: 'error' },
        ]} />
      </Space>
    </Page>
  );
}
