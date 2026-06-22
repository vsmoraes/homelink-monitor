import { Alert, Card, Col, Row, Select, Space, Statistic, Table } from 'antd';
import { ArrowDownOutlined, ArrowUpOutlined } from '@ant-design/icons';
import { Area, AreaChart, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import ResultTag from '../components/ResultTag';
import type { RouterTrafficClient, RouterTrafficCurrent, RouterTrafficSample } from '../types';
import { bpsToMbps, bytes, localTime } from '../utils/format';

type LivePoint = {
  time: string;
  download?: number;
  upload?: number;
};

const pointMbps = (value?: number) => (value === undefined ? undefined : (value * 8) / 1_000_000);
const refreshOptions = [
  { label: '2 seconds', value: 2000 },
  { label: '5 seconds', value: 5000 },
  { label: '10 seconds', value: 10000 },
  { label: '30 seconds', value: 30000 },
];

const clientSortKey = (client: RouterTrafficClient) =>
  `${client.hostname || ''}|${client.ip || ''}|${client.mac || ''}`.toLowerCase();
const textSorter = <T,>(pick: (row: T) => string | undefined) => (a: T, b: T) =>
  (pick(a) || '').localeCompare(pick(b) || '');
const rateTotal = (client: RouterTrafficClient) => (client.downloadBps ?? 0) + (client.uploadBps ?? 0);
const usageTotal = (client: RouterTrafficClient) => (client.downloadBytes ?? 0) + (client.uploadBytes ?? 0);
const downColor = '#18c98f';
const upColor = '#13b8c8';

export default function RouterTraffic() {
  const [items, setItems] = useState<RouterTrafficSample[]>([]);
  const [current, setCurrent] = useState<RouterTrafficCurrent>();
  const [live, setLive] = useState<LivePoint[]>([]);
  const [refreshMs, setRefreshMs] = useState(5000);
  const [hiddenLiveSeries, setHiddenLiveSeries] = useState<string[]>([]);
  const [hiddenHistorySeries, setHiddenHistorySeries] = useState<string[]>([]);
  const [probingStopped, setProbingStopped] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const loadHistory = async (showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      const [history, now] = await Promise.all([api.routerTraffic(), api.currentRouterTraffic()]);
      setItems(history.items);
      setCurrent(now);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadHistory();
  }, []);

  useEffect(() => {
    let stopped = false;
    let intervalID: number | undefined;
    const stopProbing = () => {
      stopped = true;
      setProbingStopped(true);
      if (intervalID !== undefined) {
        window.clearInterval(intervalID);
        intervalID = undefined;
      }
    };
    const tick = async () => {
      if (stopped) return;
      try {
        const next = await api.probeRouterTraffic();
        if (stopped) return;
        setCurrent(next);
        if (next.capability.error || next.latest?.success === false) {
          stopProbing();
          return;
        }
        setLive((points) => [
          ...points.slice(-119),
          {
            time: new Date().toLocaleTimeString(),
            download: pointMbps(next.latest?.downloadBps),
            upload: pointMbps(next.latest?.uploadBps),
          },
        ]);
      } catch (err) {
        if (!stopped) {
          setError(err as Error);
          stopProbing();
        }
      }
    };
    void tick();
    intervalID = window.setInterval(() => void tick(), refreshMs);
    return () => {
      stopped = true;
      if (intervalID !== undefined) window.clearInterval(intervalID);
    };
  }, [refreshMs]);

  const clients = useMemo(
    () => [...(current?.clients ?? [])].sort((a, b) => clientSortKey(a).localeCompare(clientSortKey(b))),
    [current?.clients],
  );
  const capability = current?.capability;
  const toggleSeries = (key: string, hidden: string[], setHidden: (next: string[]) => void) => {
    setHidden(hidden.includes(key) ? hidden.filter((item) => item !== key) : [...hidden, key]);
  };
  const historyChart = useMemo(() => [...items].reverse().filter((item) => item.success).map((item) => ({
    time: localTime(item.checkedAt),
    download: pointMbps(item.downloadBps),
    upload: pointMbps(item.uploadBps),
  })), [items]);

  return (
    <Page title="Router Traffic" loading={loading} error={error} actions={null}>
      <Space direction="vertical" size="large" className="full-width">
        {capability?.error ? (
          <Alert
            type={capability.authenticated ? 'warning' : 'error'}
            showIcon
            message={capability.error}
            description={probingStopped ? 'Live probing stopped after this error. Refresh the page to try again.' : `The live chart probes the router every ${refreshMs / 1000} seconds without storing those debug samples.`}
          />
        ) : null}

        <Card
          title="Live router usage"
          extra={
            <Select
              value={refreshMs}
              onChange={setRefreshMs}
              options={refreshOptions}
              size="small"
              aria-label="Refresh interval"
            />
          }
        >
          <Row gutter={[16, 16]} className="overview-strip">
            <Col xs={24} md={8}><Statistic title="Clients" value={current?.latest?.clientCount ?? clients.length} /></Col>
            <Col xs={12} md={8}><Statistic title="Download" value={capability?.downloadAvailable ? bpsToMbps(current?.latest?.downloadBps) : 'Unavailable'} /></Col>
            <Col xs={12} md={8}><Statistic title="Upload" value={capability?.uploadAvailable ? bpsToMbps(current?.latest?.uploadBps) : 'Unavailable'} /></Col>
          </Row>
          <ResponsiveContainer width="100%" height={280}>
            <AreaChart data={live}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenLiveSeries, setHiddenLiveSeries)} />
              {capability?.downloadAvailable ? <Area type="monotone" dataKey="download" name="Download Mbps" stroke="#18c98f" fill="#dffbea" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls isAnimationActive={false} hide={hiddenLiveSeries.includes('download')} /> : null}
              {capability?.uploadAvailable ? <Area type="monotone" dataKey="upload" name="Upload Mbps" stroke="#13b8c8" fill="#dff8fb" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls isAnimationActive={false} hide={hiddenLiveSeries.includes('upload')} /> : null}
            </AreaChart>
          </ResponsiveContainer>
        </Card>

        <Card title="Connected clients">
          <Table<RouterTrafficClient>
            rowKey={(row, index) => row.mac || row.ip || row.hostname || `client-${index}`}
            dataSource={clients}
            scroll={{ x: 'max-content' }}
            pagination={{ showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100] }}
            columns={[
              { title: 'Host', render: (_, r) => r.hostname || '—', sorter: textSorter<RouterTrafficClient>((r) => r.hostname) },
              { title: 'IP', dataIndex: 'ip', sorter: textSorter<RouterTrafficClient>((r) => r.ip) },
              { title: 'MAC', dataIndex: 'mac', sorter: textSorter<RouterTrafficClient>((r) => r.mac) },
              { title: 'Connection', dataIndex: 'connection', sorter: textSorter<RouterTrafficClient>((r) => r.connection) },
              {
                title: 'Current rate',
                sorter: (a, b) => rateTotal(a) - rateTotal(b),
                render: (_, r) => (
                  <Space direction="vertical" size={0}>
                    <span style={{ color: downColor }}><ArrowDownOutlined /> {bpsToMbps(r.downloadBps)}</span>
                    <span style={{ color: upColor }}><ArrowUpOutlined /> {bpsToMbps(r.uploadBps)}</span>
                  </Space>
                ),
              },
              {
                title: 'Today',
                sorter: (a, b) => usageTotal(a) - usageTotal(b),
                render: (_, r) => (
                  <Space direction="vertical" size={0}>
                    <span style={{ color: downColor }}><ArrowDownOutlined /> {bytes(r.downloadBytes)}</span>
                    <span style={{ color: upColor }}><ArrowUpOutlined /> {bytes(r.uploadBytes)}</span>
                  </Space>
                ),
              },
            ]}
          />
        </Card>

        <Card title="Stored samples">
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={historyChart}>
              <XAxis dataKey="time" hide />
              <YAxis />
              <Tooltip />
              <Legend onClick={(item) => toggleSeries(String(item.dataKey), hiddenHistorySeries, setHiddenHistorySeries)} />
              <Area type="monotone" dataKey="download" name="Download Mbps" stroke="#18c98f" fill="#dffbea" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls isAnimationActive={false} hide={hiddenHistorySeries.includes('download')} />
              <Area type="monotone" dataKey="upload" name="Upload Mbps" stroke="#13b8c8" fill="#dff8fb" strokeWidth={1.5} dot={false} activeDot={{ r: 5 }} connectNulls isAnimationActive={false} hide={hiddenHistorySeries.includes('upload')} />
            </AreaChart>
          </ResponsiveContainer>
          <Table<RouterTrafficSample>
            rowKey="id"
            dataSource={items}
            scroll={{ x: 'max-content' }}
            pagination={{ showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100] }}
            columns={[
              { title: 'Checked', dataIndex: 'checkedAt', render: localTime },
              { title: 'Clients', dataIndex: 'clientCount' },
              { title: 'Download', render: (_, r) => bpsToMbps(r.downloadBps) },
              { title: 'Upload', render: (_, r) => bpsToMbps(r.uploadBps) },
              { title: 'Result', render: (_, r) => <ResultTag success={r.success} /> },
              { title: 'Error', dataIndex: 'error' },
            ]}
          />
        </Card>
      </Space>
    </Page>
  );
}
