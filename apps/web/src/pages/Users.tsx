import { Button, Form, Input, Modal, Select, Space, Table, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import Page from '../components/Page';
import type { User } from '../types';
import { localTime } from '../utils/format';

type FormValues = {
  username: string;
  password?: string;
  role: string;
};

export default function Users() {
  const [items, setItems] = useState<User[]>([]);
  const [editing, setEditing] = useState<User | null>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [form] = Form.useForm<FormValues>();

  const load = async () => {
    setLoading(true);
    try {
      setItems((await api.users()).items);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void load(); }, []);

  const startCreate = () => {
    setEditing(null);
    form.setFieldsValue({ username: '', password: '', role: 'admin' });
    setOpen(true);
  };

  const startEdit = (user: User) => {
    setEditing(user);
    form.setFieldsValue({ username: user.username, password: '', role: user.role });
    setOpen(true);
  };

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await api.updateUser(editing.id, values);
      } else {
        await api.createUser({ username: values.username, password: values.password ?? '', role: values.role });
      }
      message.success('User saved');
      setOpen(false);
      await load();
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const remove = async (user: User) => {
    try {
      await api.deleteUser(user.id);
      message.success('User deleted');
      await load();
    } catch (err) {
      message.error((err as Error).message);
    }
  };

  return (
    <Page title="Users" loading={loading} error={error} actions={<Button type="primary" icon={<PlusOutlined />} onClick={startCreate}>Add User</Button>}>
      <Table rowKey="id" dataSource={items} scroll={{ x: 'max-content' }} columns={[
        { title: 'Username', dataIndex: 'username' },
        { title: 'Role', dataIndex: 'role' },
        { title: 'Created', dataIndex: 'createdAt', render: localTime },
        { title: 'Updated', dataIndex: 'updatedAt', render: localTime },
        {
          title: 'Actions',
          render: (_, user) => (
            <Space>
              <Button onClick={() => startEdit(user)}>Edit</Button>
              <Button danger onClick={() => void remove(user)}>Delete</Button>
            </Space>
          ),
        },
      ]} />
      <Modal title={editing ? 'Edit User' : 'Add User'} open={open} onOk={save} confirmLoading={saving} onCancel={() => setOpen(false)}>
        <Form layout="vertical" form={form}>
          <Form.Item
            name="username"
            label="Username"
            rules={[
              { required: true },
              { max: 64 },
              { pattern: /^[A-Za-z0-9._@-]+$/, message: 'Use letters, numbers, dot, dash, underscore, or @' },
            ]}
          >
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="password"
            label={editing ? 'New password' : 'Password'}
            rules={editing ? [{ max: 72 }] : [{ required: true }, { min: 8 }, { max: 72 }]}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="role" label="Role" rules={[{ required: true }]}>
            <Select options={[{ value: 'admin', label: 'Admin' }]} />
          </Form.Item>
        </Form>
      </Modal>
    </Page>
  );
}
