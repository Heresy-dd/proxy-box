import React, { useState, useEffect } from 'react';
import { X, CheckCircle, AlertTriangle, Send, History, RotateCcw, ShieldCheck } from 'lucide-react';
import { ConfigDiff } from '../types';
import { ApiClient } from '../api/client';

interface ApplyDiffModalProps {
  isOpen: boolean;
  onClose: () => void;
  diff: ConfigDiff | null;
  onApply: (desc: string) => Promise<void>;
  onRefreshDiff: () => Promise<void>;
}

export const ApplyDiffModal: React.FC<ApplyDiffModalProps> = ({
  isOpen,
  onClose,
  diff,
  onApply,
  onRefreshDiff,
}) => {
  const [tab, setTab] = useState<'diff' | 'history'>('diff');
  const [description, setDescription] = useState('');
  const [applying, setApplying] = useState(false);
  const [versions, setVersions] = useState<any[]>([]);
  const [rollingBack, setRollingBack] = useState<number | null>(null);

  useEffect(() => {
    if (isOpen) {
      onRefreshDiff();
      ApiClient.getVersions().then(setVersions).catch(() => {});
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const handleApply = async () => {
    setApplying(true);
    try {
      await onApply(description || '应用控制台配置变更');
      setDescription('');
      onClose();
    } catch (err: any) {
      alert(err.message || '应用配置失败');
    } finally {
      setApplying(false);
    }
  };

  const handleRollback = async (version: number) => {
    if (!confirm(`确定回滚至版本 v${version} 吗？`)) return;
    setRollingBack(version);
    try {
      await ApiClient.rollback(version);
      alert(`已成功回滚至版本 v${version}`);
      onClose();
    } catch (err: any) {
      alert(err.message || '回滚失败');
    } finally {
      setRollingBack(null);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-3">
            <h2 className="text-base font-bold text-white">生产配置应用与版本回滚</h2>
            <div className="flex bg-slate-950 p-1 rounded-lg border border-slate-800 text-xs">
              <button
                onClick={() => setTab('diff')}
                className={`px-3 py-1 rounded-md font-medium transition ${
                  tab === 'diff' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white'
                }`}
              >
                变更 Diff 预览
              </button>
              <button
                onClick={() => setTab('history')}
                className={`px-3 py-1 rounded-md font-medium transition ${
                  tab === 'history' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white'
                }`}
              >
                历史版本 ({versions.length})
              </button>
            </div>
          </div>

          <button onClick={onClose} className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {tab === 'diff' ? (
            <div className="space-y-4">
              <div className="p-4 bg-slate-950 rounded-xl border border-slate-800 space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-400">活跃启用链路数:</span>
                  <span className="text-white font-mono font-bold">{diff?.enabled_chains ?? 0}</span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-400">监听端口分配:</span>
                  <span className="text-emerald-400 font-mono font-bold">
                    {diff?.active_ports && diff.active_ports.length > 0 ? diff.active_ports.join(', ') : '无'}
                  </span>
                </div>
              </div>

              <div>
                <label className="text-xs text-slate-400 block mb-1.5">配置变更描述 (可选)</label>
                <input
                  type="text"
                  placeholder="例：新增 9001 澳门落地链路"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-sky-500"
                />
              </div>

              <div className="bg-sky-950/30 border border-sky-900/40 p-3 rounded-xl flex items-start gap-2.5 text-xs text-sky-300">
                <ShieldCheck className="w-4 h-4 text-sky-400 shrink-0 mt-0.5" />
                <span>
                  点击应用后，系统自动通过 <code>mihomo -t</code> 校验配置语法；通过 External-Controller API
                  热重载；若热重载遇到不可逆异常将自动安全回滚至上一可用版本。
                </span>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {versions.length === 0 ? (
                <p className="text-xs text-slate-500 text-center py-6">暂无历史配置版本记录</p>
              ) : (
                versions.map((ver) => (
                  <div
                    key={ver.version}
                    className="p-3.5 bg-slate-950 border border-slate-800 rounded-xl flex items-center justify-between gap-3 text-xs"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-sky-400">v{ver.version}</span>
                        <span className="text-slate-300 font-medium">{ver.description || '配置应用'}</span>
                      </div>
                      <p className="text-[11px] text-slate-500 mt-0.5">
                        {new Date(ver.applied_at).toLocaleString('zh-CN')}
                      </p>
                    </div>

                    <button
                      onClick={() => handleRollback(ver.version)}
                      disabled={rollingBack === ver.version}
                      className="flex items-center gap-1 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg transition"
                    >
                      <RotateCcw className={`w-3.5 h-3.5 ${rollingBack === ver.version ? 'animate-spin' : ''}`} />
                      <span>{rollingBack === ver.version ? '回滚中...' : '回滚此版'}</span>
                    </button>
                  </div>
                ))
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        {tab === 'diff' && (
          <div className="p-4 border-t border-slate-800 flex justify-end gap-2 bg-slate-950/50">
            <button onClick={onClose} className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs">
              取消
            </button>
            <button
              onClick={handleApply}
              disabled={applying}
              className="px-5 py-2 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded-xl text-xs font-semibold flex items-center gap-1.5 shadow-lg shadow-sky-600/20"
            >
              <Send className="w-4 h-4" />
              <span>{applying ? '预检并热重载中...' : '立即应用生产配置'}</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
