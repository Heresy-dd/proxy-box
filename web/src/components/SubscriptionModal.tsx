import React, { useState } from 'react';
import { X, Plus, RefreshCw, Trash2, Globe, Clock, HardDrive, AlertCircle, CheckCircle2 } from 'lucide-react';
import { Subscription } from '../types';

interface SubscriptionModalProps {
  isOpen: boolean;
  onClose: () => void;
  subscriptions: Subscription[];
  onAdd: (name: string, url: string, interval: number) => Promise<void>;
  onRefresh: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export const SubscriptionModal: React.FC<SubscriptionModalProps> = ({
  isOpen,
  onClose,
  subscriptions,
  onAdd,
  onRefresh,
  onDelete,
}) => {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [interval, setInterval] = useState(60);
  const [submitting, setSubmitting] = useState(false);
  const [refreshingId, setRefreshingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !url.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await onAdd(name.trim(), url.trim(), interval);
      setName('');
      setUrl('');
    } catch (err: any) {
      setError(err.message || '添加订阅失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRefresh = async (id: string) => {
    setRefreshingId(id);
    setError(null);
    try {
      await onRefresh(id);
    } catch (err: any) {
      setError(err.message || '刷新订阅失败');
    } finally {
      setRefreshingId(null);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes <= 0) return '0 B';
    const gb = bytes / (1024 * 1024 * 1024);
    if (gb >= 1) return `${gb.toFixed(2)} GB`;
    const mb = bytes / (1024 * 1024);
    return `${mb.toFixed(1)} MB`;
  };

  const formatExpire = (ts: number) => {
    if (!ts || ts <= 0) return '长期有效 / 未提供';
    return new Date(ts * 1000).toLocaleDateString('zh-CN');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-3xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div>
            <h2 className="text-lg font-bold text-white">订阅源管理</h2>
            <p className="text-xs text-slate-400">支持 Clash/Mihomo YAML 及 Base64 URI 列表，自动提取流量头并过滤伪节点</p>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {error && (
            <div className="p-3 bg-red-950/60 border border-red-800 text-red-200 text-xs rounded-xl flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Add Subscription Form */}
          <form onSubmit={handleSubmit} className="bg-slate-950/70 border border-slate-800 rounded-xl p-4 space-y-3">
            <h3 className="text-sm font-semibold text-sky-400 flex items-center gap-2">
              <Plus className="w-4 h-4" /> 添加新订阅
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div>
                <label className="text-xs text-slate-400 block mb-1">订阅名称</label>
                <input
                  type="text"
                  placeholder="例：机场A-前置 / 住宅专线"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-white focus:outline-none focus:border-sky-500"
                  required
                />
              </div>
              <div className="md:col-span-2">
                <label className="text-xs text-slate-400 block mb-1">订阅 URL</label>
                <input
                  type="url"
                  placeholder="https://..."
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-white focus:outline-none focus:border-sky-500 font-mono text-xs"
                  required
                />
              </div>
            </div>
            <div className="flex items-center justify-between pt-1">
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <span>定时刷新间隔:</span>
                <select
                  value={interval}
                  onChange={(e) => setInterval(Number(e.target.value))}
                  className="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-slate-200 text-xs"
                >
                  <option value={30}>30 分钟</option>
                  <option value={60}>60 分钟 (1小时)</option>
                  <option value={120}>2 小时</option>
                  <option value={360}>6 小时</option>
                  <option value={1440}>24 小时</option>
                </select>
              </div>
              <button
                type="submit"
                disabled={submitting}
                className="px-4 py-1.5 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition"
              >
                {submitting ? '拉取解析中...' : '添加并拉取'}
              </button>
            </div>
          </form>

          {/* Subscriptions List */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-slate-300">已配置订阅 ({subscriptions.length})</h3>
            {subscriptions.length === 0 ? (
              <p className="text-xs text-slate-500 text-center py-6">暂无订阅，请在上方添加您的前置机场或住宅代理订阅</p>
            ) : (
              subscriptions.map((sub) => {
                const usedBytes = sub.upload_bytes + sub.download_bytes;
                const percent = sub.total_bytes > 0 ? Math.min(100, (usedBytes / sub.total_bytes) * 100) : 0;

                return (
                  <div key={sub.id} className="bg-slate-950 border border-slate-800 rounded-xl p-4 space-y-3">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-white text-sm">{sub.name}</span>
                          <span
                            className={`text-[10px] px-2 py-0.5 rounded font-mono ${
                              sub.status === 'active'
                                ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                                : 'bg-red-950 text-red-400 border border-red-800'
                            }`}
                          >
                            {sub.status === 'active' ? '正常' : '拉取异常'}
                          </span>
                          <span className="text-xs text-slate-500">
                            已入库节点: <b className="text-slate-300">{sub.node_count ?? 0}</b>
                          </span>
                        </div>
                        <p className="text-xs font-mono text-slate-500 truncate max-w-md mt-0.5">{sub.url}</p>
                      </div>

                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleRefresh(sub.id)}
                          disabled={refreshingId === sub.id}
                          className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition disabled:opacity-50"
                          title="立即刷新订阅"
                        >
                          <RefreshCw className={`w-4 h-4 ${refreshingId === sub.id ? 'animate-spin text-sky-400' : ''}`} />
                        </button>
                        <button
                          onClick={() => onDelete(sub.id)}
                          className="p-1.5 rounded-lg bg-slate-800 hover:bg-red-950/60 text-slate-400 hover:text-red-400 transition"
                          title="删除订阅"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>

                    {/* Traffic & Info */}
                    {sub.total_bytes > 0 && (
                      <div className="space-y-1.5">
                        <div className="flex justify-between text-xs text-slate-400">
                          <span>流量使用: {formatBytes(usedBytes)} / {formatBytes(sub.total_bytes)}</span>
                          <span>{percent.toFixed(1)}%</span>
                        </div>
                        <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all ${
                              percent > 85 ? 'bg-red-500' : percent > 60 ? 'bg-amber-400' : 'bg-sky-500'
                            }`}
                            style={{ width: `${percent}%` }}
                          />
                        </div>
                      </div>
                    )}

                    <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1 border-t border-slate-900">
                      <div className="flex items-center gap-3">
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          更新于: {sub.last_updated_at ? new Date(sub.last_updated_at).toLocaleTimeString() : '未更新'}
                        </span>
                        <span>到期: {formatExpire(sub.expire_timestamp)}</span>
                      </div>
                      <span>定时间隔: {sub.update_interval} 分钟</span>
                    </div>

                    {sub.error_message && (
                      <p className="text-xs text-red-400 bg-red-950/40 p-2 rounded-lg border border-red-900/50">
                        {sub.error_message}
                      </p>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
