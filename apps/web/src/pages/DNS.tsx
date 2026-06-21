import { Card, Space, Table } from 'antd';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { DNSCheck } from '../types';
import { localTime, ms } from '../utils/format';

export default function DNS() {
  const [items, setItems] = useState<DNSCheck[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  useEffect(() => {
    api.dnsChecks().then((res) => { setItems(res.items); setError(null); }).catch(setError).finally(() => setLoading(false));
  }, []);
  const chart = [...items].reverse().filter((item) => item.success).map((item) => ({ time: localTime(item.checkedAt), duration: item.durationMs, domain: item.domain }));
  return (
    <Page title="DNS" loading={loading} error={error}>
      <Space direction="vertical" size="large" className="full-width">
        <Card>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chart}><XAxis dataKey="time" hide /><YAxis /><Tooltip /><Line type="monotone" dataKey="duration" stroke="#13c2c2" dot={false} /></LineChart>
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
