import { Table, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { Outage } from '../types';
import { duration, localTime } from '../utils/format';

export default function Outages() {
  const [items, setItems] = useState<Outage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  useEffect(() => {
    api.outages().then((res) => { setItems(res.items); setError(null); }).catch(setError).finally(() => setLoading(false));
  }, []);
  return (
    <Page title="Outages" loading={loading} error={error}>
      <Table rowKey="id" dataSource={items} scroll={{ x: 'max-content' }} columns={[
        { title: 'Start', dataIndex: 'startedAt', render: localTime },
        { title: 'End', dataIndex: 'endedAt', render: localTime },
        { title: 'Duration', render: (_, r) => duration(r.startedAt, r.endedAt) },
        { title: 'Status', render: (_, r) => <Tag color={r.endedAt ? 'default' : 'error'}>{r.endedAt ? 'Resolved' : 'Active'}</Tag> },
        { title: 'Reason', dataIndex: 'reason' },
      ]} />
    </Page>
  );
}
