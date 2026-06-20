import { Alert, Spin, Typography } from 'antd';
import type { ReactNode } from 'react';

type Props = {
  title: string;
  loading?: boolean;
  error?: Error | null;
  actions?: ReactNode;
  children: ReactNode;
};

export default function Page({ title, loading, error, actions, children }: Props) {
  return (
    <div>
      <div className="page-heading">
        <Typography.Title level={2}>{title}</Typography.Title>
        {actions}
      </div>
      {error ? <Alert type="error" showIcon message={error.message} className="page-alert" /> : null}
      <Spin spinning={Boolean(loading)}>
        {children}
      </Spin>
    </div>
  );
}
