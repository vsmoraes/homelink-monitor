import { Button, Card, Form, Input, Typography, Alert } from 'antd';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { api } from '../api/client';
import type { User } from '../types';

type Props = {
  onLogin: (user: User) => void;
};

export default function Login({ onLogin }: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const submit = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await api.login(values.username, values.password);
      setError(null);
      onLogin(res.user);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <Card className="login-card">
        <div className="login-brand">
          <img src="/logo.png" alt="" />
          <Typography.Title level={3}>HomeLink Monitor</Typography.Title>
        </div>
        {error ? <Alert type="error" showIcon message={error.message} className="page-alert" /> : null}
        <Form layout="vertical" onFinish={submit}>
          <Form.Item name="username" label="Username" rules={[{ required: true }]}>
            <Input prefix={<UserOutlined />} autoComplete="username" />
          </Form.Item>
          <Form.Item name="password" label="Password" rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} autoComplete="current-password" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            Sign In
          </Button>
        </Form>
      </Card>
    </div>
  );
}
