import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, message } from 'antd';
import { ReloadOutlined, SaveOutlined } from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { Settings } from '../types';

const splitLines = (value?: string) => value?.split('\n').map((v) => v.trim()).filter(Boolean) ?? [];
const joinLines = (value?: string[]) => value?.join('\n') ?? '';

export default function SettingsPage() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [saveError, setSaveError] = useState<Error | null>(null);
  const [saved, setSaved] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const settings = await api.settings();
      form.setFieldsValue({ ...settings, latencyTargetsText: joinLines(settings.latencyTargets), dnsDomainsText: joinLines(settings.dnsDomains) });
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [form]);

  const save = async (values: Settings & { latencyTargetsText: string; dnsDomainsText: string }) => {
    setSaving(true);
    setSaved(false);
    setSaveError(null);
    try {
      const settings: Settings = { ...values, latencyTargets: splitLines(values.latencyTargetsText), dnsDomains: splitLines(values.dnsDomainsText) };
      const savedSettings = await api.saveSettings(settings);
      form.setFieldsValue({ ...savedSettings, latencyTargetsText: joinLines(savedSettings.latencyTargets), dnsDomainsText: joinLines(savedSettings.dnsDomains) });
      setSaved(true);
      message.success('Settings saved');
    } catch (err) {
      setSaveError(err as Error);
      message.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Page title="Settings" loading={loading} error={error} actions={<Button icon={<ReloadOutlined />} onClick={() => void load()}>Reload</Button>}>
      <Card>
        {saved ? <Alert type="success" showIcon message="Settings saved" className="page-alert" /> : null}
        {saveError ? <Alert type="error" showIcon message={saveError.message} className="page-alert" /> : null}
        <Form layout="vertical" form={form} onFinish={save}>
          <Form.Item name="monitoringEnabled" label="Monitoring" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="speedTestCommand" label="Speed test command"><Input /></Form.Item>
          <Space wrap size="large">
            <Form.Item name="speedTestScheduleMinutes" label="Speed test schedule minutes"><InputNumber min={0} /></Form.Item>
            <Form.Item name="latencyIntervalSeconds" label="Latency interval seconds"><InputNumber min={10} /></Form.Item>
            <Form.Item name="dnsIntervalSeconds" label="DNS interval seconds"><InputNumber min={10} /></Form.Item>
            <Form.Item name="outageFailureThreshold" label="Outage failure threshold"><InputNumber min={1} /></Form.Item>
          </Space>
          <Space wrap size="large">
            <Form.Item name="minDownloadMbps" label="Minimum download Mbps"><InputNumber min={0} /></Form.Item>
            <Form.Item name="minUploadMbps" label="Minimum upload Mbps"><InputNumber min={0} /></Form.Item>
            <Form.Item name="maxLatencyMs" label="Max latency ms"><InputNumber min={1} /></Form.Item>
          </Space>
          <Form.Item name="routerIp" label="Router target"><Input placeholder="192.168.1.1:80" /></Form.Item>
          <Form.Item name="latencyTargetsText" label="Latency targets"><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="dnsDomainsText" label="DNS domains"><Input.TextArea rows={4} /></Form.Item>
          <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving} disabled={saving}>
            {saving ? 'Saving' : 'Save'}
          </Button>
        </Form>
      </Card>
    </Page>
  );
}
