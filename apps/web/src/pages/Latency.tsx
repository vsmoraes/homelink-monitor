import { Card, Input, Space, Table } from 'antd';
import { Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { LatencyCheck, LatencySummary } from '../types';
import { localTime, ms, number } from '../utils/format';

const colors = ['#18c98f', '#13b8c8'];
const seriesKey = (value: string) => value.replace(/[^a-zA-Z0-9]/g, '_');

export default function Latency() {
  const [target, setTarget] = useState('');
  const [items, setItems] = useState<LatencyCheck[]>([]);
  const [summary, setSummary] = useState<LatencySummary>();
  const [hiddenSeries, setHiddenSeries] = useState<string[]>([]);
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
  const targets = Array.from(new Set(items.map((item) => item.target)));
  const allChart = [...items].reverse().filter((item) => item.success).map((item) => ({
    time: localTime(item.checkedAt),
    [seriesKey(item.target)]: item.latencyMs,
  }));
  const toggleSeries = (key: string) => {
    setHiddenSeries(hiddenSeries.includes(key) ? hiddenSeries.filter((item) => item !== key) : [...hiddenSeries, key]);
  };
  return (
    <Page title="Latency" loading={loading} error={error} actions={<Input.Search placeholder="Filter target" allowClear onSearch={setTarget} className="target-search" />}>
      <Space direction="vertical" size="large" className="full-width">
        <Card title={`Min ${ms(summary?.minMs)}   Avg ${ms(summary?.avgMs)}   Max ${ms(summary?.maxMs)}   Loss ${number(summary?.packetLoss)}%`}>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={allChart}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              <Legend onClick={(item) => toggleSeries(String(item.dataKey))} />
              {targets.map((target, index) => {
                const key = seriesKey(target);
                return (
                  <Line key={target} type="monotone" dataKey={key} name={target} stroke={colors[index % colors.length]} strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls hide={hiddenSeries.includes(key)} />
                );
              })}
            </LineChart>
          </ResponsiveContainer>
        </Card>
        <Table rowKey="id" dataSource={items} scroll={{ x: 'max-content' }} columns={[
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
