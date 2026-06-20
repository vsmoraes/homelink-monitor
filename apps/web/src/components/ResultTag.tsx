import { Tag } from 'antd';

export default function ResultTag({ success }: { success: boolean }) {
  return <Tag color={success ? 'success' : 'error'}>{success ? 'Success' : 'Failed'}</Tag>;
}
