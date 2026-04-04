import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { FlowWithStatus } from '../api';

export default function FlowList() {
  const [flows, setFlows] = useState<FlowWithStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    api.listFlows().then(setFlows).finally(() => setLoading(false));
  }, []);

  const handleCreate = async () => {
    const flow = await api.createFlow({
      name: '新建流程',
      target: { url: '', method: 'POST' },
      mappings: [],
    });
    navigate(`/flows/${flow.id}`);
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.preventDefault();
    e.stopPropagation();
    if (!confirm('确认删除此流程？')) return;
    await api.deleteFlow(id);
    setFlows(flows.filter(f => f.id !== id));
  };

  if (loading) return <div className="empty"><p>加载中...</p></div>;

  return (
    <div>
      <div className="card-header" style={{ marginBottom: 20 }}>
        <span className="section-title">全部流程</span>
        <button className="btn btn-primary" onClick={handleCreate}>
          + 新建
        </button>
      </div>

      {flows.length === 0 ? (
        <div className="card">
          <div className="empty">
            <p>还没有流程</p>
            <p>创建第一个 Webhook 转发流程</p>
          </div>
        </div>
      ) : (
        flows.map(flow => {
          const lastLog = flow.last_log;
          const lastStatus = lastLog
            ? lastLog.error
              ? 'error'
              : lastLog.response_status >= 200 && lastLog.response_status < 300
                ? 'success'
                : 'warning'
            : null;

          return (
            <Link key={flow.id} to={`/flows/${flow.id}`} className="flow-link">
              <div className="card">
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div>
                    <span className="flow-name">{flow.name}</span>
                    {flow.description && <span className="flow-desc">{flow.description}</span>}
                  </div>
                  <div className="actions">
                    <span className={`badge ${flow.enabled ? 'badge-success' : 'badge-muted'}`}>
                      {flow.enabled ? '运行中' : '已停用'}
                    </span>
                    <button className="btn btn-danger btn-sm" onClick={(e) => handleDelete(e, flow.id)}>
                      删除
                    </button>
                  </div>
                </div>
                <div className="flow-meta">
                  <span>{flow.mappings.length} 个映射</span>
                  <span>{flow.target.method} {flow.target.url || '未配置目标'}</span>
                  {lastLog && (
                    <>
                      <span style={{ marginLeft: 'auto' }}>
                        <span className={`badge badge-${lastStatus === 'error' ? 'error' : lastStatus === 'success' ? 'success' : 'muted'}`} style={{ marginRight: 6 }}>
                          {lastLog.error ? 'ERR' : lastLog.response_status}
                        </span>
                        {new Date(lastLog.received_at).toLocaleString()}
                      </span>
                    </>
                  )}
                </div>
              </div>
            </Link>
          );
        })
      )}
    </div>
  );
}
