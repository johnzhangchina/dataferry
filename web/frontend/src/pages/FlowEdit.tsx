import { useEffect, useState, useId, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { api } from '../api';
import type { Flow, Mapping } from '../api';
import { extractPaths, autoMatch } from '../fieldUtils';

const emptyMapping = (): Mapping => ({
  source: '',
  target: '',
  transform: 'direct',
  value: '',
});

// Simple mapping preview engine (mirrors Go engine logic)
function previewTransform(source: Record<string, any>, mappings: Mapping[]): Record<string, any> {
  const result: Record<string, any> = {};
  for (const m of mappings) {
    let value: any;
    const transform = m.transform || 'direct';
    if (transform === 'constant') {
      value = m.value;
    } else {
      if (!m.source) continue;
      value = getNestedValue(source, m.source);
      if (value === undefined) continue;
    }
    if (!m.target) continue;
    setNestedValue(result, m.target, value);
  }
  return result;
}

function getNestedValue(obj: any, path: string): any {
  const parts = path.split('.');
  let current = obj;
  for (const part of parts) {
    if (current == null || typeof current !== 'object') return undefined;
    if (Array.isArray(current)) {
      const idx = parseInt(part, 10);
      if (isNaN(idx) || idx < 0 || idx >= current.length) return undefined;
      current = current[idx];
    } else {
      if (!(part in current)) return undefined;
      current = current[part];
    }
  }
  return current;
}

function setNestedValue(obj: Record<string, any>, path: string, value: any) {
  const parts = path.split('.');
  let current = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    if (!(parts[i] in current) || typeof current[parts[i]] !== 'object') {
      current[parts[i]] = {};
    }
    current = current[parts[i]];
  }
  current[parts[parts.length - 1]] = value;
}

interface SortableMappingProps {
  mapping: Mapping;
  sortId: string;
  index: number;
  onChange: (index: number, patch: Partial<Mapping>) => void;
  onRemove: (index: number) => void;
}

function SortableMapping({ mapping: m, sortId, index, onChange, onRemove }: SortableMappingProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: sortId });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style} className={`mapping-row${isDragging ? ' dragging' : ''}`}>
      <div className="drag-handle" {...attributes} {...listeners}>
        &#x2630;
      </div>
      {m.transform === 'constant' || m.transform === 'template' || m.transform === 'expression' ? (
        <input
          className={m.transform === 'constant' ? 'mapping-constant-input' : ''}
          value={m.value || ''}
          onChange={e => onChange(index, { value: e.target.value })}
          placeholder={
            m.transform === 'constant' ? '固定值'
            : m.transform === 'template' ? '{{first}} {{last}}'
            : 'price * 100'
          }
        />
      ) : (
        <input
          value={m.source || ''}
          onChange={e => onChange(index, { source: e.target.value })}
          placeholder="data.field_name"
        />
      )}
      <div className="mapping-arrow">&rarr;</div>
      <input
        value={m.target}
        onChange={e => onChange(index, { target: e.target.value })}
        placeholder="target_field"
      />
      <select
        value={m.transform || 'direct'}
        onChange={e => onChange(index, { transform: e.target.value })}
      >
        <option value="direct">字段取值</option>
        <option value="constant">固定值</option>
        <option value="template">字符串拼接</option>
        <option value="expression">数值运算</option>
      </select>
      <button className="mapping-delete" onClick={() => onRemove(index)} title="删除">
        &times;
      </button>
    </div>
  );
}

export default function FlowEdit() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [flow, setFlow] = useState<Flow | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [toast, setToast] = useState('');
  const prefix = useId();

  // JSON paste state
  const [showJsonPanel, setShowJsonPanel] = useState(false);
  const [sourcePaths, setSourcePaths] = useState<string[]>([]);
  const [targetPaths, setTargetPaths] = useState<string[]>([]);
  const [parseError, setParseError] = useState('');

  // Webhook test state
  const [showTestPanel, setShowTestPanel] = useState(false);
  const [testPayload, setTestPayload] = useState('');
  const [testResult, setTestResult] = useState('');
  const [testing, setTesting] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  useEffect(() => {
    if (id) {
      api.getFlow(id).then((f) => {
        setFlow(f);
        if (f.source_example) {
          try { setSourcePaths(extractPaths(JSON.parse(f.source_example))); } catch {}
        }
        if (f.target_example) {
          try { setTargetPaths(extractPaths(JSON.parse(f.target_example))); } catch {}
        }
      }).catch(() => navigate('/'));
    }
  }, [id, navigate]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(''), 2500);
  }, []);

  if (!flow) return <div className="empty"><p>加载中...</p></div>;

  const update = (patch: Partial<Flow>) => setFlow({ ...flow, ...patch });
  const updateTarget = (patch: Partial<Flow['target']>) =>
    update({ target: { ...flow.target, ...patch } });

  const updateMapping = (index: number, patch: Partial<Mapping>) => {
    const mappings = [...flow.mappings];
    mappings[index] = { ...mappings[index], ...patch };
    update({ mappings });
  };

  const addMapping = () => update({ mappings: [...flow.mappings, emptyMapping()] });
  const removeMapping = (index: number) =>
    update({ mappings: flow.mappings.filter((_, i) => i !== index) });

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = Number(String(active.id).split('-').pop());
      const newIndex = Number(String(over.id).split('-').pop());
      update({ mappings: arrayMove(flow.mappings, oldIndex, newIndex) });
    }
  };

  const validate = (): string | null => {
    for (let i = 0; i < flow.mappings.length; i++) {
      const m = flow.mappings[i];
      if (!m.target) return `第 ${i + 1} 条映射：目标字段不能为空`;
      const t = m.transform || 'direct';
      if (t === 'direct' && !m.source)
        return `第 ${i + 1} 条映射：字段取值类型必须填写源字段`;
      if ((t === 'constant' || t === 'template' || t === 'expression') && !m.value)
        return `第 ${i + 1} 条映射：${t === 'constant' ? '固定值' : t === 'template' ? '字符串拼接' : '数值运算'}类型必须填写值`;
    }
    return null;
  };

  const handleSave = async () => {
    setError('');
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }
    setSaving(true);
    try {
      const updated = await api.updateFlow(flow.id, flow);
      setFlow(updated);
      showToast('保存成功');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const handleCopyUrl = () => {
    const url = `${window.location.origin}/webhook/${flow.webhook_path}`;
    navigator.clipboard.writeText(url).then(() => showToast('已复制'));
  };

  const handleToggleEnabled = async () => {
    const updated = { ...flow, enabled: !flow.enabled };
    try {
      const result = await api.updateFlow(flow.id, updated);
      setFlow(result);
      showToast(result.enabled ? '已启用' : '已停用');
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleTest = async () => {
    if (!testPayload.trim()) {
      setTestResult('请输入测试 JSON');
      return;
    }
    let parsed: any;
    try {
      parsed = JSON.parse(testPayload);
    } catch {
      setTestResult('JSON 格式无效');
      return;
    }

    // Show local preview
    const preview = previewTransform(parsed, flow.mappings);
    let resultText = `--- 映射预览 ---\n${JSON.stringify(preview, null, 2)}`;

    // Send real request if flow has target URL
    if (flow.target.url) {
      setTesting(true);
      try {
        const webhookUrl = `/webhook/${flow.webhook_path}`;
        const res = await fetch(webhookUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: testPayload,
        });
        const data = await res.json();
        resultText += `\n\n--- 实际响应 (${res.status}) ---\n${JSON.stringify(data, null, 2)}`;
      } catch (e: any) {
        resultText += `\n\n--- 请求失败 ---\n${e.message}`;
      } finally {
        setTesting(false);
      }
    }

    setTestResult(resultText);
  };

  // Preview mapping result from source_example
  const mappingPreview = (() => {
    if (!flow.source_example) return null;
    try {
      const source = JSON.parse(flow.source_example);
      const result = previewTransform(source, flow.mappings);
      if (Object.keys(result).length === 0) return null;
      return JSON.stringify(result, null, 2);
    } catch {
      return null;
    }
  })();

  const updateHeader = (key: string, value: string, oldKey?: string) => {
    const headers = { ...flow.target.headers };
    if (oldKey && oldKey !== key) delete headers[oldKey];
    if (key) headers[key] = value;
    updateTarget({ headers });
  };

  const removeHeader = (key: string) => {
    const headers = { ...flow.target.headers };
    delete headers[key];
    updateTarget({ headers });
  };

  // JSON parsing
  const handleParseSource = (json: string) => {
    update({ source_example: json });
    setParseError('');
    if (!json.trim()) { setSourcePaths([]); return; }
    try {
      const obj = JSON.parse(json);
      setSourcePaths(extractPaths(obj));
    } catch {
      setParseError('源 JSON 格式无效');
      setSourcePaths([]);
    }
  };

  const handleParseTarget = (json: string) => {
    update({ target_example: json });
    setParseError('');
    if (!json.trim()) { setTargetPaths([]); return; }
    try {
      const obj = JSON.parse(json);
      setTargetPaths(extractPaths(obj));
    } catch {
      setParseError('目标 JSON 格式无效');
      setTargetPaths([]);
    }
  };

  const handleAutoMatch = () => {
    if (sourcePaths.length === 0 && targetPaths.length === 0) return;
    const matched = autoMatch(sourcePaths, targetPaths);
    const usedSource = new Set(matched.map(m => m.source));
    const usedTarget = new Set(matched.map(m => m.target));
    const mappings: Mapping[] = [];
    for (const m of matched) {
      mappings.push({ source: m.source, target: m.target, transform: 'direct' });
    }
    for (const s of sourcePaths) {
      if (!usedSource.has(s)) mappings.push({ source: s, target: '', transform: 'direct' });
    }
    for (const t of targetPaths) {
      if (!usedTarget.has(t)) mappings.push({ source: '', target: t, transform: 'direct' });
    }
    update({ mappings });
    setShowJsonPanel(false);
  };

  const handleSourceOnly = () => {
    if (sourcePaths.length === 0) return;
    update({ mappings: sourcePaths.map(s => ({ source: s, target: '', transform: 'direct' })) });
    setShowJsonPanel(false);
  };

  const handleTargetOnly = () => {
    if (targetPaths.length === 0) return;
    update({ mappings: targetPaths.map(t => ({ source: '', target: t, transform: 'direct' })) });
    setShowJsonPanel(false);
  };

  const sortIds = flow.mappings.map((_, i) => `${prefix}-${i}`);
  const canAutoMatch = sourcePaths.length > 0 && targetPaths.length > 0;
  const canSourceOnly = sourcePaths.length > 0 && targetPaths.length === 0;
  const canTargetOnly = targetPaths.length > 0 && sourcePaths.length === 0;

  return (
    <div>
      {/* Toast */}
      {toast && <div className="toast">{toast}</div>}

      <div className="card-header">
        <Link to="/" className="btn btn-ghost">&larr; 返回</Link>
        <div className="actions">
          <Link to={`/flows/${flow.id}/logs`} className="btn btn-secondary">
            执行日志
          </Link>
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </div>

      {error && <div className="error-msg">{error}</div>}

      {/* Basic Info */}
      <div className="card">
        <div className="card-header">
          <h3 className="section-title" style={{ margin: 0 }}>基本信息</h3>
          <div className="toggle-group" onClick={handleToggleEnabled}>
            <div className={`toggle ${flow.enabled ? 'toggle-on' : ''}`}>
              <div className="toggle-knob" />
            </div>
            <span style={{ fontSize: 13, color: flow.enabled ? 'var(--success)' : 'var(--text-tertiary)' }}>
              {flow.enabled ? '运行中' : '已停用'}
            </span>
          </div>
        </div>
        <div className="form-row">
          <div className="form-group">
            <label>流程名称</label>
            <input value={flow.name} onChange={e => update({ name: e.target.value })} />
          </div>
          <div className="form-group">
            <label>Webhook 路径</label>
            <input value={flow.webhook_path} onChange={e => update({ webhook_path: e.target.value })} />
          </div>
        </div>
        <div className="form-group">
          <label>描述</label>
          <input
            value={flow.description || ''}
            onChange={e => update({ description: e.target.value })}
            placeholder="简要描述这个流程的用途"
          />
        </div>
        <div className="webhook-url-row">
          <div className="webhook-url" style={{ flex: 1 }}>
            POST {window.location.origin}/webhook/{flow.webhook_path}
          </div>
          <button className="btn btn-secondary btn-sm" onClick={handleCopyUrl}>
            复制
          </button>
        </div>
        <div className="form-group" style={{ marginTop: 16 }}>
          <label>签名密钥（可选，用于验证请求来源）</label>
          <input
            type="password"
            value={flow.webhook_config?.secret || ''}
            onChange={e => update({ webhook_config: { ...flow.webhook_config, secret: e.target.value } })}
            placeholder="留空则不校验签名"
          />
          {flow.webhook_config?.secret && (
            <div style={{ fontSize: 12, color: 'var(--text-tertiary)', marginTop: 4 }}>
              源系统需在请求头中发送 X-Signature-256: sha256=HMAC(payload, secret)
            </div>
          )}
        </div>
      </div>

      {/* Target API */}
      <div className="card">
        <h3 className="section-title" style={{ marginBottom: 16 }}>目标 API</h3>
        <div className="form-row">
          <div className="form-group">
            <label>请求方法</label>
            <select value={flow.target.method} onChange={e => updateTarget({ method: e.target.value })}>
              <option>GET</option>
              <option>POST</option>
              <option>PUT</option>
              <option>PATCH</option>
              <option>DELETE</option>
            </select>
          </div>
          <div className="form-group">
            <label>目标 URL</label>
            <input
              value={flow.target.url}
              onChange={e => updateTarget({ url: e.target.value })}
              placeholder="https://api.example.com/endpoint"
            />
          </div>
        </div>
        <div className="form-group">
          <label>请求头</label>
          {Object.entries(flow.target.headers || {}).map(([key, value]) => (
            <div key={key} className="header-row">
              <input
                className="header-key"
                value={key}
                onChange={e => updateHeader(e.target.value, value, key)}
                placeholder="Header 名称"
              />
              <div className="header-value">
                <input
                  value={value}
                  onChange={e => updateHeader(key, e.target.value)}
                  placeholder="Header 值"
                />
                <button className="mapping-delete" onClick={() => removeHeader(key)} title="删除">
                  &times;
                </button>
              </div>
            </div>
          ))}
          <button
            className="btn btn-ghost btn-sm"
            style={{ marginTop: 4 }}
            onClick={() => updateHeader(`Header-${Date.now()}`, '')}
          >
            + 添加请求头
          </button>
        </div>
        <div className="form-row" style={{ gridTemplateColumns: '1fr 1fr 1fr' }}>
          <div className="form-group">
            <label>超时（秒）</label>
            <input
              type="number"
              min="1"
              max="300"
              value={flow.target.timeout || 30}
              onChange={e => updateTarget({ timeout: parseInt(e.target.value) || 30 })}
            />
          </div>
          <div className="form-group">
            <label>失败重试次数</label>
            <input
              type="number"
              min="0"
              max="10"
              value={flow.target.retry_count || 0}
              onChange={e => updateTarget({ retry_count: parseInt(e.target.value) || 0 })}
            />
          </div>
          <div className="form-group">
            <label>重试间隔（秒）</label>
            <input
              type="number"
              min="1"
              max="60"
              value={flow.target.retry_delay || 3}
              onChange={e => updateTarget({ retry_delay: parseInt(e.target.value) || 3 })}
            />
          </div>
        </div>
      </div>

      {/* Conditions */}
      <div className="card">
        <div className="card-header">
          <h3 className="section-title" style={{ margin: 0 }}>转发条件</h3>
          <button className="btn btn-ghost btn-sm" onClick={() => {
            const conditions = [...(flow.conditions || []), { field: '', operator: '==', value: '' }];
            update({ conditions });
          }}>
            + 添加
          </button>
        </div>

        {(!flow.conditions || flow.conditions.length === 0) ? (
          <div style={{ fontSize: 13, color: 'var(--text-tertiary)' }}>
            未设置条件，所有请求都会转发
          </div>
        ) : (
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
              <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>条件关系：</span>
              <button
                className={`btn btn-sm ${(flow.condition_logic || 'and') === 'and' ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => update({ condition_logic: 'and' })}
              >
                全部满足 (AND)
              </button>
              <button
                className={`btn btn-sm ${flow.condition_logic === 'or' ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => update({ condition_logic: 'or' })}
              >
                任一满足 (OR)
              </button>
            </div>
            {flow.conditions.map((c, i) => (
              <div key={i} className="condition-row">
                <input
                  value={c.field}
                  onChange={e => {
                    const conditions = [...flow.conditions!];
                    conditions[i] = { ...conditions[i], field: e.target.value };
                    update({ conditions });
                  }}
                  placeholder="data.status"
                />
                <select
                  value={c.operator}
                  onChange={e => {
                    const conditions = [...flow.conditions!];
                    conditions[i] = { ...conditions[i], operator: e.target.value };
                    update({ conditions });
                  }}
                >
                  <option value="==">等于</option>
                  <option value="!=">不等于</option>
                  <option value=">">大于</option>
                  <option value="<">小于</option>
                  <option value="contains">包含</option>
                  <option value="exists">存在</option>
                </select>
                {c.operator !== 'exists' && (
                  <input
                    value={c.value || ''}
                    onChange={e => {
                      const conditions = [...flow.conditions!];
                      conditions[i] = { ...conditions[i], value: e.target.value };
                      update({ conditions });
                    }}
                    placeholder="期望值"
                  />
                )}
                <button
                  className="mapping-delete"
                  onClick={() => {
                    const conditions = flow.conditions!.filter((_, j) => j !== i);
                    update({ conditions });
                  }}
                  title="删除"
                >
                  &times;
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Mappings */}
      <div className="card">
        <div className="card-header">
          <h3 className="section-title" style={{ margin: 0 }}>字段映射</h3>
          <div className="actions">
            <button
              className="btn btn-secondary btn-sm"
              onClick={() => setShowJsonPanel(!showJsonPanel)}
            >
              {showJsonPanel ? '收起' : '从 JSON 生成'}
            </button>
            <button className="btn btn-ghost btn-sm" onClick={addMapping}>
              + 添加
            </button>
          </div>
        </div>

        {/* JSON Paste Panel */}
        {showJsonPanel && (
          <div className="json-panel">
            {parseError && <div className="error-msg">{parseError}</div>}
            <div className="json-panel-row">
              <div className="json-panel-col">
                <label>源 JSON 示例</label>
                <textarea
                  value={flow.source_example || ''}
                  onChange={e => handleParseSource(e.target.value)}
                  placeholder={'粘贴 Webhook 收到的 JSON，例如:\n{\n  "data": {\n    "user_name": "张三",\n    "email": "a@b.com"\n  }\n}'}
                  rows={8}
                />
                {sourcePaths.length > 0 && (
                  <div className="json-paths">
                    <span className="json-paths-label">提取到 {sourcePaths.length} 个字段:</span>
                    {sourcePaths.map(p => <code key={p}>{p}</code>)}
                  </div>
                )}
              </div>
              <div className="json-panel-col">
                <label>目标 JSON 格式</label>
                <textarea
                  value={flow.target_example || ''}
                  onChange={e => handleParseTarget(e.target.value)}
                  placeholder={'粘贴目标 API 需要的 JSON 格式:\n{\n  "username": "",\n  "contact": {\n    "email": ""\n  }\n}'}
                  rows={8}
                />
                {targetPaths.length > 0 && (
                  <div className="json-paths">
                    <span className="json-paths-label">提取到 {targetPaths.length} 个字段:</span>
                    {targetPaths.map(p => <code key={p}>{p}</code>)}
                  </div>
                )}
              </div>
            </div>
            <div className="json-panel-actions">
              {canAutoMatch && (
                <button className="btn btn-primary" onClick={handleAutoMatch}>
                  自动匹配并生成映射
                </button>
              )}
              {canSourceOnly && (
                <button className="btn btn-primary" onClick={handleSourceOnly}>
                  用源字段生成映射
                </button>
              )}
              {canTargetOnly && (
                <button className="btn btn-primary" onClick={handleTargetOnly}>
                  用目标字段生成映射
                </button>
              )}
              {!canAutoMatch && !canSourceOnly && !canTargetOnly && (
                <span style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>
                  至少粘贴一侧的 JSON 来生成映射
                </span>
              )}
              <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
                生成后会替换现有映射
              </span>
            </div>
          </div>
        )}

        {flow.mappings.length === 0 && !showJsonPanel ? (
          <div className="empty" style={{ padding: '24px 0' }}>
            <p>暂无映射规则</p>
            <p>点击「从 JSON 生成」快速创建，或手动添加</p>
          </div>
        ) : flow.mappings.length > 0 && (
          <>
            <div className="mapping-header">
              <span></span>
              <span>源 / 固定值</span>
              <span></span>
              <span>目标字段</span>
              <span>类型</span>
              <span></span>
            </div>
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
              <SortableContext items={sortIds} strategy={verticalListSortingStrategy}>
                <div className="mapping-list">
                  {flow.mappings.map((m, i) => (
                    <SortableMapping
                      key={sortIds[i]}
                      sortId={sortIds[i]}
                      mapping={m}
                      index={i}
                      onChange={updateMapping}
                      onRemove={removeMapping}
                    />
                  ))}
                </div>
              </SortableContext>
            </DndContext>
          </>
        )}

        {/* Mapping Preview */}
        {mappingPreview && (
          <div className="preview-section">
            <div className="log-label">映射预览（基于源 JSON 示例）</div>
            <div className="log-payload">{mappingPreview}</div>
          </div>
        )}
      </div>

      {/* Webhook Test */}
      <div className="card">
        <div className="card-header">
          <h3 className="section-title" style={{ margin: 0 }}>在线测试</h3>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => setShowTestPanel(!showTestPanel)}
          >
            {showTestPanel ? '收起' : '展开'}
          </button>
        </div>

        {showTestPanel && (
          <div>
            <div className="form-group">
              <label>测试 JSON（发送到此流程的 Webhook）</label>
              <textarea
                value={testPayload}
                onChange={e => setTestPayload(e.target.value)}
                placeholder={'{\n  "data": {\n    "user_name": "测试用户",\n    "email": "test@example.com"\n  }\n}'}
                rows={6}
                style={{
                  fontFamily: "'SF Mono', 'Fira Code', monospace",
                  fontSize: 12,
                }}
              />
            </div>
            <div className="actions">
              <button
                className="btn btn-primary"
                onClick={handleTest}
                disabled={testing}
              >
                {testing ? '发送中...' : '发送测试'}
              </button>
              {flow.source_example && !testPayload && (
                <button
                  className="btn btn-ghost btn-sm"
                  onClick={() => setTestPayload(flow.source_example || '')}
                >
                  使用源 JSON 示例
                </button>
              )}
            </div>
            {testResult && (
              <div className="log-payload" style={{ marginTop: 12 }}>
                {testResult}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
