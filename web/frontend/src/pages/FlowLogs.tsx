import { useEffect, useState, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api';
import type { ExecutionLog } from '../api';

const PAGE_SIZE = 20;

export default function FlowLogs() {
  const { id } = useParams<{ id: string }>();
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [retrying, setRetrying] = useState<Set<string>>(new Set());
  const [toast, setToast] = useState('');

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(''), 2500);
  }, []);

  const fetchLogs = (p: number, status: string) => {
    if (!id) return;
    setLoading(true);
    api.listLogs(id, p, PAGE_SIZE, status).then(res => {
      setLogs(res.logs);
      setTotal(res.total);
      setPage(res.page);
    }).finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchLogs(1, statusFilter);
  }, [id, statusFilter]);

  const toggle = (logId: string) => {
    const next = new Set(expanded);
    if (next.has(logId)) next.delete(logId);
    else next.add(logId);
    setExpanded(next);
  };

  const handleRetry = async (e: React.MouseEvent, logId: string) => {
    e.stopPropagation();
    if (!id) return;
    setRetrying(prev => new Set(prev).add(logId));
    try {
      const result = await api.retryLog(id, logId);
      if (result.target_status >= 200 && result.target_status < 300) {
        showToast(`重发成功 (${result.target_status})`);
      } else {
        showToast(`重发完成，目标返回 ${result.target_status}`);
      }
      // Refresh logs to show new entry
      fetchLogs(page, statusFilter);
    } catch (err: any) {
      showToast(`重发失败: ${err.message}`);
    } finally {
      setRetrying(prev => {
        const next = new Set(prev);
        next.delete(logId);
        return next;
      });
    }
  };

  const handleRetryAll = async () => {
    if (!id) return;
    const failedLogs = logs.filter(l => l.error || l.response_status >= 400);
    if (failedLogs.length === 0) return;
    if (!confirm(`确认重发 ${failedLogs.length} 条失败记录？`)) return;

    let success = 0;
    let fail = 0;
    for (const log of failedLogs) {
      try {
        await api.retryLog(id, log.id);
        success++;
      } catch {
        fail++;
      }
    }
    showToast(`重发完成: ${success} 成功, ${fail} 失败`);
    fetchLogs(page, statusFilter);
  };

  const formatJSON = (s: string) => {
    try { return JSON.stringify(JSON.parse(s), null, 2); }
    catch { return s; }
  };

  const isError = (log: ExecutionLog) => !!(log.error || log.response_status >= 400);
  const totalPages = Math.ceil(total / PAGE_SIZE);
  const hasFailedLogs = logs.some(isError);

  return (
    <div>
      {toast && <div className="toast">{toast}</div>}

      <div className="card-header">
        <Link to={`/flows/${id}`} className="btn btn-ghost">&larr; 返回流程</Link>
        <div className="actions">
          <span className="section-title" style={{ margin: 0 }}>执行日志</span>
          <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>共 {total} 条</span>
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 12, alignItems: 'center' }}>
        {(['', 'success', 'error'] as const).map(s => (
          <button
            key={s}
            className={`btn btn-sm ${statusFilter === s ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setStatusFilter(s)}
          >
            {s === '' ? '全部' : s === 'success' ? '成功' : '失败'}
          </button>
        ))}
        {hasFailedLogs && (
          <button
            className="btn btn-sm btn-danger"
            style={{ marginLeft: 'auto' }}
            onClick={handleRetryAll}
          >
            重发所有失败
          </button>
        )}
      </div>

      <div className="card">
        {loading ? (
          <div className="empty"><p>加载中...</p></div>
        ) : logs.length === 0 ? (
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
                {(log.retry_attempts ?? 0) > 0 && (
                  <span className="badge badge-muted">
                    重试 {log.retry_attempts}
                  </span>
                )}
                <span style={{ flex: 1 }} />
                {isError(log) && (
                  <button
                    className="btn btn-sm btn-secondary"
                    style={{ padding: '2px 8px', fontSize: 11 }}
                    onClick={(e) => handleRetry(e, log.id)}
                    disabled={retrying.has(log.id)}
                  >
                    {retrying.has(log.id) ? '重发中...' : '重发'}
                  </button>
                )}
                <span style={{ fontSize: 12, color: 'var(--text-tertiary)', minWidth: 32, textAlign: 'right' }}>
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

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 16 }}>
          <button
            className="btn btn-secondary btn-sm"
            disabled={page <= 1}
            onClick={() => fetchLogs(page - 1, statusFilter)}
          >
            上一页
          </button>
          <span style={{ fontSize: 13, color: 'var(--text-tertiary)', lineHeight: '28px' }}>
            {page} / {totalPages}
          </span>
          <button
            className="btn btn-secondary btn-sm"
            disabled={page >= totalPages}
            onClick={() => fetchLogs(page + 1, statusFilter)}
          >
            下一页
          </button>
        </div>
      )}
    </div>
  );
}
