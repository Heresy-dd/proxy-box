import React, { useState, useEffect } from 'react';
import { X, Shield, Terminal, Save, RefreshCw } from 'lucide-react';
import { ApiClient } from '../api/client';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
  const [tab, setTab] = useState<'security' | 'speedtest' | 'logs'>('security');
  const [whitelist, setWhitelist] = useState('');
  const [speedtestUrl, setSpeedtestUrl] = useState('https://cp.cloudflare.com/generate_204');
  const [timeout, setTimeoutVal] = useState('5');
  const [concurrency, setConcurrency] = useState('10');
  const [logs, setLogs] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [fetchingLogs, setFetchingLogs] = useState(false);

  useEffect(() => {
    if (isOpen) {
      ApiClient.getSettings().then((s) => {
        if (s.ip_whitelist !== undefined) setWhitelist(s.ip_whitelist);
        if (s.speedtest_url) setSpeedtestUrl(s.speedtest_url);
        if (s.speedtest_timeout) setTimeoutVal(s.speedtest_timeout);
        if (s.speedtest_concurrency) setConcurrency(s.speedtest_concurrency);
      }).catch(() => {});

      fetchLogs();
    }
  }, [isOpen]);

  const fetchLogs = async () => {
    setFetchingLogs(true);
    try {
      const res = await ApiClient.getLogs();
      setLogs(res.logs || []);
    } catch {
      setLogs(['暂无日志']);
    } finally {
      setFetchingLogs(false);
    }
  };

  if (!isOpen) return null;

  const handleSave = async () => {
    setSaving(true);
    try {
      await ApiClient.saveSettings({
        ip_whitelist: whitelist,
        speedtest_url: speedtestUrl,
        speedtest_timeout: timeout,
        speedtest_concurrency: concurrency,
      });
      alert('设置已成功保存！若修改了来源 IP 白名单，请在主界面点击「应用配置」生效。');
      onClose();
    } catch (err: any) {
      alert(err.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
          <div className="flex items-center gap-3">
            <h2 className="text-base font-bold text-white">系统设置与日志</h2>
            <div className="flex bg-slate-950 p-1 rounded-lg border border-slate-800 text-xs">
              <button
                onClick={() => setTab('security')}
                className={`px-3 py-1 rounded-md font-medium transition ${
                  tab === 'security' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white'
                }`}
              >
                访问安全白名单
              </button>
              <button
                onClick={() => setTab('speedtest')}
                className={`px-3 py-1 rounded-md font-medium transition ${
                  tab === 'speedtest' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white'
                }`}
              >
                链式测速参数
              </button>
              <button
                onClick={() => setTab('logs')}
                className={`px-3 py-1 rounded-md font-medium transition ${
                  tab === 'logs' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-white'
                }`}
              >
                Mihomo 日志
              </button>
            </div>
          </div>

          <button onClick={onClose} className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {tab === 'security' && (
            <div className="space-y-4 text-xs">
              <div className="p-3.5 bg-amber-950/30 border border-amber-900/40 rounded-xl text-amber-300">
                <p className="font-semibold mb-1">为防止服务器被恶意扫描盗用，建议设置来源 IP 白名单：</p>
                <p className="text-[11px] text-amber-400/80">
                  仅允许指定的客户端公网 IP（如指纹浏览器所在的公网出口 IP）连接监听端口；多个 IP 请用半角逗号分隔。
                </p>
              </div>

              <div>
                <label className="text-slate-300 block mb-1.5 font-medium">允许访问的客户端 IP 白名单</label>
                <textarea
                  rows={3}
                  placeholder="例如：124.78.112.5, 219.140.22.88/32（留空表示允许全部 IP）"
                  value={whitelist}
                  onChange={(e) => setWhitelist(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-white font-mono focus:outline-none focus:border-sky-500"
                />
              </div>
            </div>
          )}

          {tab === 'speedtest' && (
            <div className="space-y-4 text-xs">
              <div>
                <label className="text-slate-300 block mb-1 font-medium">测速验证 URL</label>
                <input
                  type="text"
                  value={speedtestUrl}
                  onChange={(e) => setSpeedtestUrl(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-white font-mono focus:outline-none focus:border-sky-500"
                />
                <p className="text-[11px] text-slate-500 mt-1">推荐使用 Cloudflare 204 无内容端点</p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-slate-300 block mb-1 font-medium">单节点测速超时 (秒)</label>
                  <input
                    type="number"
                    value={timeout}
                    onChange={(e) => setTimeoutVal(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-white font-mono focus:outline-none focus:border-sky-500"
                  />
                </div>
                <div>
                  <label className="text-slate-300 block mb-1 font-medium">最大并发测速数</label>
                  <input
                    type="number"
                    value={concurrency}
                    onChange={(e) => setConcurrency(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-white font-mono focus:outline-none focus:border-sky-500"
                  />
                </div>
              </div>
            </div>
          )}

          {tab === 'logs' && (
            <div className="space-y-3">
              <div className="flex items-center justify-between text-xs">
                <span className="text-slate-400">运行日志 (最近 500 行内存环形缓冲)</span>
                <button
                  onClick={fetchLogs}
                  disabled={fetchingLogs}
                  className="flex items-center gap-1 text-sky-400 hover:text-sky-300"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${fetchingLogs ? 'animate-spin' : ''}`} />
                  <span>刷新日志</span>
                </button>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 font-mono text-[11px] text-slate-300 max-h-80 overflow-y-auto space-y-1">
                {logs.length === 0 ? (
                  <p className="text-slate-600">暂无日志</p>
                ) : (
                  logs.map((line, idx) => (
                    <div key={idx} className="break-all whitespace-pre-wrap">
                      {line}
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        {tab !== 'logs' && (
          <div className="p-4 border-t border-slate-800 flex justify-end gap-2 bg-slate-950/50">
            <button onClick={onClose} className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs">
              取消
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-5 py-2 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded-xl text-xs font-semibold flex items-center gap-1.5 shadow-lg shadow-sky-600/20"
            >
              <Save className="w-4 h-4" />
              <span>{saving ? '保存中...' : '保存设置'}</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
