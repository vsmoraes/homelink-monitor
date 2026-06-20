import { DashboardOutlined, DatabaseOutlined, GlobalOutlined, LogoutOutlined, SettingOutlined, TeamOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
import { Button, Layout, Menu, Space, Spin, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { Link, Route, Routes, useLocation } from 'react-router-dom';
import { api } from './api/client';
import Dashboard from './pages/Dashboard';
import DNS from './pages/DNS';
import Latency from './pages/Latency';
import Login from './pages/Login';
import Outages from './pages/Outages';
import SettingsPage from './pages/Settings';
import SpeedTests from './pages/SpeedTests';
import Users from './pages/Users';
import type { User } from './types';

const { Header, Sider, Content } = Layout;

const items = [
  { key: '/', icon: <DashboardOutlined />, label: <Link to="/">Dashboard</Link> },
  { key: '/speed-tests', icon: <ThunderboltOutlined />, label: <Link to="/speed-tests">Speed Tests</Link> },
  { key: '/latency', icon: <GlobalOutlined />, label: <Link to="/latency">Latency</Link> },
  { key: '/dns', icon: <DatabaseOutlined />, label: <Link to="/dns">DNS</Link> },
  { key: '/outages', icon: <WarningOutlined />, label: <Link to="/outages">Outages</Link> },
  { key: '/settings', icon: <SettingOutlined />, label: <Link to="/settings">Settings</Link> },
  { key: '/users', icon: <TeamOutlined />, label: <Link to="/users">Users</Link> },
];

export default function App() {
  const location = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [loadingUser, setLoadingUser] = useState(true);

  useEffect(() => {
    api.me()
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setLoadingUser(false));
  }, []);

  const logout = async () => {
    try {
      await api.logout();
      setUser(null);
    } catch (err) {
      message.error((err as Error).message);
    }
  };

  if (loadingUser) {
    return <div className="center-screen"><Spin /></div>;
  }

  if (!user) {
    return <Login onLogin={setUser} />;
  }

  return (
    <Layout className="app-shell">
      <Sider breakpoint="lg" collapsedWidth="0" theme="light" className="app-sider">
        <Typography.Title level={4} className="app-title">Connection Monitor</Typography.Title>
        <Menu mode="inline" selectedKeys={[location.pathname]} items={items} />
      </Sider>
      <Layout>
        <Header className="app-header">
          <Space className="header-user">
            <Typography.Text>{user.username}</Typography.Text>
            <Button icon={<LogoutOutlined />} onClick={() => void logout()}>Logout</Button>
          </Space>
        </Header>
        <Content className="app-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/speed-tests" element={<SpeedTests />} />
            <Route path="/latency" element={<Latency />} />
            <Route path="/dns" element={<DNS />} />
            <Route path="/outages" element={<Outages />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/users" element={<Users />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
}
