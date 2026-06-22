import { Alert, Button, Card, Col, Form, Input, InputNumber, Row, Space, Switch, Typography, message } from 'antd';
import { FieldTimeOutlined, ReloadOutlined, SaveOutlined, ThunderboltOutlined, WifiOutlined, ApartmentOutlined } from '@ant-design/icons';
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
  const [monitoringEnabled, setMonitoringEnabled] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const settings = await api.settings();
      form.setFieldsValue({ ...settings, latencyTargetsText: joinLines(settings.latencyTargets), dnsDomainsText: joinLines(settings.dnsDomains) });
      setMonitoringEnabled(settings.monitoringEnabled);
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
      const nextMonitoringEnabled = typeof values.monitoringEnabled === 'boolean' ? values.monitoringEnabled : monitoringEnabled;
      const settings: Settings = {
        ...values,
        monitoringEnabled: nextMonitoringEnabled,
        latencyTargets: splitLines(values.latencyTargetsText),
        dnsDomains: splitLines(values.dnsDomainsText),
      };
      const savedSettings = await api.saveSettings(settings);
      form.setFieldsValue({ ...savedSettings, latencyTargetsText: joinLines(savedSettings.latencyTargets), dnsDomainsText: joinLines(savedSettings.dnsDomains) });
      setMonitoringEnabled(savedSettings.monitoringEnabled);
      setSaved(true);
      message.success('Settings saved');
      return true;
    } catch (err) {
      setSaveError(err as Error);
      message.error((err as Error).message);
      return false;
    } finally {
      setSaving(false);
    }
  };

  const saveMonitoringEnabled = async (nextMonitoringEnabled: boolean) => {
    const previous = form.getFieldValue('monitoringEnabled') as boolean | undefined;
    setMonitoringEnabled(nextMonitoringEnabled);
    form.setFieldValue('monitoringEnabled', nextMonitoringEnabled);
    const ok = await save({ ...form.getFieldsValue(true), monitoringEnabled: nextMonitoringEnabled });
    if (!ok) {
      setMonitoringEnabled(Boolean(previous));
      form.setFieldValue('monitoringEnabled', previous);
    }
  };

  return (
    <Page title="Settings" loading={loading} error={error} actions={<Button icon={<ReloadOutlined />} onClick={() => void load()}>Reload</Button>}>
      <Space direction="vertical" size="large" className="full-width">
        {saved ? <Alert type="success" showIcon message="Settings saved" /> : null}
        {saveError ? <Alert type="error" showIcon message={saveError.message} /> : null}
        <Form layout="vertical" form={form} onFinish={save}>
          <Card className="settings-toolbar">
            <Row gutter={[16, 16]} align="middle">
              <Col xs={24} md={14}>
                <Space direction="vertical" size={2}>
                  <Typography.Text strong>Monitoring service</Typography.Text>
                  <Typography.Text type="secondary">Enable or pause all background checks without losing history.</Typography.Text>
                </Space>
              </Col>
              <Col xs={24} md={10} className="settings-actions">
                <Switch
                  checked={Boolean(monitoringEnabled)}
                  checkedChildren="Enabled"
                  unCheckedChildren="Paused"
                  loading={saving}
                  disabled={saving}
                  onChange={(checked) => void saveMonitoringEnabled(checked)}
                />
                <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving} disabled={saving}>
                  {saving ? 'Saving' : 'Save changes'}
                </Button>
              </Col>
            </Row>
          </Card>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title={<Space><ThunderboltOutlined />Speed test</Space>} className={`settings-card ${monitoringEnabled ? '' : 'settings-card-disabled'}`}>
                <Form.Item name="speedTestCommand" label="Command">
                  <Input disabled={!monitoringEnabled} placeholder="speedtest --accept-license --accept-gdpr --format=json" />
                </Form.Item>
                <Row gutter={16}>
                  <Col xs={24} sm={12}>
                    <Form.Item name="speedTestScheduleMinutes" label="Run every">
                      <InputNumber min={0} addonAfter="minutes" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} sm={12}>
                    <Form.Item name="outageFailureThreshold" label="Outage threshold">
                      <InputNumber min={1} addonAfter="failed rounds" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                </Row>
              </Card>
            </Col>

            <Col xs={24} lg={12}>
              <Card title={<Space><FieldTimeOutlined />Intervals</Space>} className={`settings-card ${monitoringEnabled ? '' : 'settings-card-disabled'}`}>
                <Row gutter={16}>
                  <Col xs={24} sm={12}>
                    <Form.Item name="latencyIntervalSeconds" label="Latency checks">
                      <InputNumber min={10} addonAfter="seconds" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} sm={12}>
                    <Form.Item name="dnsIntervalSeconds" label="DNS checks">
                      <InputNumber min={10} addonAfter="seconds" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col xs={24} sm={8}>
                    <Form.Item name="minDownloadMbps" label="Min download">
                      <InputNumber min={0} addonAfter="Mbps" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} sm={8}>
                    <Form.Item name="minUploadMbps" label="Min upload">
                      <InputNumber min={0} addonAfter="Mbps" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} sm={8}>
                    <Form.Item name="maxLatencyMs" label="Max latency">
                      <InputNumber min={1} addonAfter="ms" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                </Row>
              </Card>
            </Col>

            <Col xs={24}>
              <Card title={<Space><WifiOutlined />Targets</Space>} className={`settings-card ${monitoringEnabled ? '' : 'settings-card-disabled'}`}>
                <Row gutter={16}>
                  <Col xs={24} lg={8}>
                    <Form.Item name="routerIp" label="Router target">
                      <Input disabled={!monitoringEnabled} placeholder="192.168.1.1:80" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} lg={8}>
                    <Form.Item name="latencyTargetsText" label="Latency targets">
                      <Input.TextArea rows={5} disabled={!monitoringEnabled} placeholder={'1.1.1.1:53\n8.8.8.8:53'} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} lg={8}>
                    <Form.Item name="dnsDomainsText" label="DNS domains">
                      <Input.TextArea rows={5} disabled={!monitoringEnabled} placeholder={'google.com\ncloudflare.com'} />
                    </Form.Item>
                  </Col>
                </Row>
              </Card>
            </Col>

            <Col xs={24}>
              <Card title={<Space><ApartmentOutlined />TP-Link router traffic</Space>} className={`settings-card ${monitoringEnabled ? '' : 'settings-card-disabled'}`}>
                <Row gutter={16}>
                  <Col xs={24} md={8}>
                    <Form.Item name="routerTrafficEnabled" label="Collect router traffic" valuePropName="checked">
                      <Switch disabled={!monitoringEnabled} checkedChildren="Enabled" unCheckedChildren="Off" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name="routerTrafficIntervalSeconds" label="Run every">
                      <InputNumber min={10} addonAfter="seconds" className="full-width" disabled={!monitoringEnabled} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name="routerTrafficUsername" label="Local username">
                      <Input disabled={!monitoringEnabled} placeholder="admin" autoComplete="off" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="routerTrafficUrl" label="Router URL">
                      <Input disabled={!monitoringEnabled} placeholder="http://192.168.1.1" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="routerTrafficPassword" label="Local password">
                      <Input.Password disabled={!monitoringEnabled} autoComplete="new-password" />
                    </Form.Item>
                  </Col>
                </Row>
                <Alert
                  type="info"
                  showIcon
                  message="Traffic dimensions are detected from the router response."
                  description="Download and upload charts appear only when the TP-Link API returns separate values. If the router exposes only total traffic, HomeLink Monitor stores and displays total traffic only."
                />
              </Card>
            </Col>
          </Row>
        </Form>
      </Space>
    </Page>
  );
}
