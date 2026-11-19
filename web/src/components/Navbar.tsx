import React from 'react';
import { Layers, Rss, Settings, RefreshCw, Send, ShieldCheck, LogOut, Activity, ChevronDown } from 'lucide-react';
import { ConfigDiff, Subscription } from '../types';

interface NavbarProps {
  diff: ConfigDiff | null;
  subscriptions?: Subscription[];
  onOpenSubscriptions: () => void;
  onOpenSettings: () => void;
  onOpenApply: () => void;
  onLogout: () => void;
  isMock: boolean;
  onToggleMock: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  diff,
  subscriptions = [],
  onOpenSubscriptions,
  onOpenSettings,
  onOpenApply,
  onLogout,
  isMock,
  onToggleMock,
}) => {
  const hasChanges = diff && (diff.added_ports?.length > 0 || diff.removed_ports?.length > 0);

  const totalUsedBytes = subscriptions.reduce((acc, s) => acc + s.upload_bytes + s.download_bytes, 0);
  const totalQuotaBytes = subscriptions.reduce((acc, s) => acc + s.total_bytes, 0);
  const totalNodesCount = subscriptions.reduce((acc, s) => acc + (s.node_count || 0), 0);

  const formatGb = (b: number) => (b / (1024 * 1024 * 1024)).toFixed(1);

  return (
    <header className="sticky top-0 z-30 bg-slate-900/90 backdrop-blur-md border-b border-slate-800 px-6 py-3">
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-600 flex items-center justify-center shadow-lg shadow-sky-500/20">
            <Layers className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-lg text-white tracking-wide">ProxyBox</span>
              <span className="text-xs px-2 py-0.5 rounded bg-sky-950 text-sky-400 border border-sky-800 font-mono">
                Mihomo Core v1.19
              </span>
              <button
                onClick={onToggleMock}
                className={`text-[11px] px-2 py-0.5 rounded font-medium transition cursor-pointer border ${
                  isMock
                    ? 'bg-amber-950/80 text-amber-400 border-amber-800'
                    : 'bg-emerald-950/80 text-emerald-400 border-emerald-800'
                }`}
                title="点击切换在线 / 离线预览模式"
              >
                {isMock ? '离线 Mock 预览' : '服务在线'}
              </button>
            </div>
            <p className="text-xs text-slate-400">
              前置中继 × 住宅落地链式代理 · 独立混合端口映射
            </p>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-3">
          {/* Subscriptions Pill Button */}
          <button
            onClick={onOpenSubscriptions}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium border border-slate-700 transition group"
          >
            <Rss className="w-4 h-4 text-sky-400 group-hover:scale-110 transition" />
            <div className="flex items-center gap-1.5">
              <span>订阅源 ({subscriptions.length})</span>
              {totalQuotaBytes > 0 && (
                <span className="text-[10px] px-1.5 py-0.2 rounded bg-slate-900 text-slate-400 font-mono">
                  {formatGb(totalUsedBytes)}/{formatGb(totalQuotaBytes)} GB
                </span>
              )}
            </div>
          </button>

          {/* Settings */}
          <button
            onClick={onOpenSettings}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium border border-slate-700 transition"
          >
            <Settings className="w-4 h-4 text-slate-400" />
            <span>系统设置</span>
          </button>

          {/* Apply Config Button */}
          <button
            onClick={onOpenApply}
            className={`flex items-center gap-2 px-4 py-1.5 rounded-xl text-xs font-semibold transition shadow-md ${
              hasChanges
                ? 'bg-sky-500 hover:bg-sky-400 text-white shadow-sky-500/25 animate-pulse'
                : 'bg-indigo-600 hover:bg-indigo-500 text-white shadow-indigo-600/20'
            }`}
          >
            <Send className="w-3.5 h-3.5" />
            <span>应用到生产</span>
            {hasChanges && (
              <span className="w-2 h-2 rounded-full bg-amber-300"></span>
            )}
          </button>

          {/* Logout */}
          <button
            onClick={onLogout}
            title="退出登录"
            className="p-2 rounded-xl bg-slate-800 hover:bg-red-950/50 hover:text-red-400 text-slate-400 border border-slate-700 transition"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>
  );
};
