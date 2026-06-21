import { DashboardOutlined, DatabaseOutlined, GlobalOutlined, LogoutOutlined, MenuOutlined, SettingOutlined, TeamOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
import { Button, Drawer, Grid, Layout, Menu, Space, Spin, Typography, message } from 'antd';
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

const { Header, Content } = Layout;

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
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.md;
  const [user, setUser] = useState<User | null>(null);
  const [loadingUser, setLoadingUser] = useState(true);
  const [navOpen, setNavOpen] = useState(false);

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
      <Header className="app-header">
        <div className="app-brand">
          <img src="/logo.png" alt="" />
          <Typography.Title level={4}>HomeLink Monitor</Typography.Title>
        </div>
        {isMobile ? (
          <Button className="mobile-menu-button" icon={<MenuOutlined />} onClick={() => setNavOpen(true)} />
        ) : (
          <Menu mode="horizontal" selectedKeys={[location.pathname]} items={items} className="app-menu" />
        )}
        <Space className="header-user desktop-user">
          <Typography.Text>{user.username}</Typography.Text>
          <Button icon={<LogoutOutlined />} onClick={() => void logout()}>Logout</Button>
        </Space>
      </Header>
      <Drawer
        title="HomeLink Monitor"
        placement="left"
        open={navOpen}
        onClose={() => setNavOpen(false)}
        width={300}
      >
        <Space direction="vertical" size="large" className="full-width">
          <Menu mode="inline" selectedKeys={[location.pathname]} items={items} onClick={() => setNavOpen(false)} />
          <Space className="mobile-user">
            <Typography.Text>{user.username}</Typography.Text>
            <Button icon={<LogoutOutlined />} onClick={() => void logout()}>Logout</Button>
          </Space>
        </Space>
      </Drawer>
      <Layout>
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
