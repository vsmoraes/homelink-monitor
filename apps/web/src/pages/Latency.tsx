import { Card, Input, Select, Space, Table } from 'antd';
import { Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { LatencyCheck, LatencySummary } from '../types';
import { localTime, ms, number } from '../utils/format';

const colors = ['#18c98f', '#13b8c8'];
const allSeries = '__all__';
const seriesKey = (value: string) => value.replace(/[^a-zA-Z0-9]/g, '_');

export default function Latency() {
  const [target, setTarget] = useState('');
  const [items, setItems] = useState<LatencyCheck[]>([]);
  const [summary, setSummary] = useState<LatencySummary>();
  const [focusedTarget, setFocusedTarget] = useState(allSeries);
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
  const focusedChart = [...items]
    .reverse()
    .filter((item) => item.success && item.target === focusedTarget)
    .map((item) => ({ time: localTime(item.checkedAt), value: item.latencyMs }));
  const chart = focusedTarget === allSeries ? allChart : focusedChart;
  return (
    <Page title="Latency" loading={loading} error={error} actions={<Input.Search placeholder="Filter target" allowClear onSearch={setTarget} className="target-search" />}>
      <Space direction="vertical" size="large" className="full-width">
        <Card
          title={`Min ${ms(summary?.minMs)}   Avg ${ms(summary?.avgMs)}   Max ${ms(summary?.maxMs)}   Loss ${number(summary?.packetLoss)}%`}
          extra={
            <Select
              value={focusedTarget}
              onChange={setFocusedTarget}
              className="chart-focus-select"
              options={[{ value: allSeries, label: 'All targets' }, ...targets.map((value) => ({ value, label: value }))]}
            />
          }
        >
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chart}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              {focusedTarget === allSeries ? <Legend /> : null}
              {focusedTarget === allSeries
                ? targets.map((target, index) => (
                  <Line key={target} type="monotone" dataKey={seriesKey(target)} name={target} stroke={colors[index % colors.length]} strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls />
                ))
                : <Line type="monotone" dataKey="value" name={focusedTarget} stroke={colors[0]} strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} />}
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
