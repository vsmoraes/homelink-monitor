import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider } from 'antd';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import './styles.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#18c98f',
          colorInfo: '#13b8c8',
          colorSuccess: '#7be84d',
          colorWarning: '#f5a524',
          colorBgLayout: '#f4f7f8',
          colorLink: '#0fa98f',
          borderRadius: 8,
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        },
        components: {
          Button: {
            primaryShadow: 'none',
          },
          Menu: {
            horizontalItemSelectedColor: '#18c98f',
            horizontalItemHoverColor: '#18c98f',
          },
        },
      }}
    >
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ConfigProvider>
  </React.StrictMode>,
);
