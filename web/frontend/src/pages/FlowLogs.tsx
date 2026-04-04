import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api';
import type { ExecutionLog } from '../api';

export default function FlowLogs() {
  const { id } = useParams<{ id: string }>();
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (id) {
      api.listLogs(id).then(setLogs).finally(() => setLoading(false));
    }
  }, [id]);

  const toggle = (logId: string) => {
    const next = new Set(expanded);
    if (next.has(logId)) next.delete(logId);
    else next.add(logId);
    setExpanded(next);
  };

  const formatJSON = (s: string) => {
    try { return JSON.stringify(JSON.parse(s), null, 2); }
    catch { return s; }
  };

  if (loading) return <div className="empty"><p>加载中...</p></div>;

  return (
    <div>
      <div className="card-header">
        <Link to={`/flows/${id}`} className="btn btn-ghost">&larr; 返回流程</Link>
        <span className="section-title">执行日志</span>
      </div>

      <div className="card">
        {logs.length === 0 ? (
          <div className="empty"><p>暂无执行记录</p></div>
        ) : (
          logs.map(log => (
            <div key={log.id} className="log-item">
              <div className="log-meta" onClick={() => toggle(log.id)}>
                <span className={`badge ${
                  log.error ? 'badge-error'
                  : log.response_status >= 200 && log.response_status < 300 ? 'badge-success'
                  : 'badge-muted'
                }`}>
                  {log.error ? 'ERROR' : log.response_status}
                </span>
                <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                  {new Date(log.received_at).toLocaleString()}
                </span>
                <span style={{ fontSize: 12, color: 'var(--text-tertiary)', flex: 1 }}>
                  {log.target_url}
                </span>
                <span style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>
                  {expanded.has(log.id) ? '收起' : '详情'}
                </span>
              </div>

              {expanded.has(log.id) && (
                <div style={{ marginTop: 8 }}>
                  {log.error && <div className="error-msg">{log.error}</div>}
                  <div className="log-label">接收数据</div>
                  <div className="log-payload">{formatJSON(log.source_payload)}</div>
                  <div className="log-label">映射后数据</div>
                  <div className="log-payload">{formatJSON(log.mapped_payload)}</div>
                  {log.response_body && (
                    <>
                      <div className="log-label">目标响应</div>
                      <div className="log-payload">{formatJSON(log.response_body)}</div>
                    </>
                  )}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
