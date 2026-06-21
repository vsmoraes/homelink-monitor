import { Card, Space, Table } from 'antd';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { DNSCheck } from '../types';
import { localTime, ms } from '../utils/format';

const colors = ['#13b8c8', '#18c98f', '#7be84d', '#f5a524', '#597ef7', '#eb2f96'];
const seriesKey = (value: string) => value.replace(/[^a-zA-Z0-9]/g, '_');

export default function DNS() {
  const [items, setItems] = useState<DNSCheck[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  useEffect(() => {
    api.dnsChecks().then((res) => { setItems(res.items); setError(null); }).catch(setError).finally(() => setLoading(false));
  }, []);
  const domains = Array.from(new Set(items.map((item) => item.domain)));
  const chart = [...items].reverse().filter((item) => item.success).map((item) => ({
    time: localTime(item.checkedAt),
    [seriesKey(item.domain)]: item.durationMs,
  }));
  return (
    <Page title="DNS" loading={loading} error={error}>
      <Space direction="vertical" size="large" className="full-width">
        <Card>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chart}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              {domains.map((domain, index) => (
                <Line key={domain} type="monotone" dataKey={seriesKey(domain)} name={domain} stroke={colors[index % colors.length]} dot={false} connectNulls />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </Card>
        <Table rowKey="id" dataSource={items} scroll={{ x: 'max-content' }} columns={[
          { title: 'Checked', dataIndex: 'checkedAt', render: localTime },
          { title: 'Domain', dataIndex: 'domain' },
          { title: 'Duration', render: (_, r) => ms(r.durationMs) },
          { title: 'Result', render: (_, r) => <ResultTag success={r.success} /> },
          { title: 'Error', dataIndex: 'error' },
        ]} />
      </Space>
    </Page>
  );
}
