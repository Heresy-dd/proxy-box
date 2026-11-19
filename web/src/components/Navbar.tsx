import React from 'react';
import { Layers, Rss, Settings, RefreshCw, Send, ShieldCheck, LogOut, Activity } from 'lucide-react';
import { ConfigDiff } from '../types';

interface NavbarProps {
  diff: ConfigDiff | null;
  onOpenSubscriptions: () => void;
  onOpenSettings: () => void;
  onOpenApply: () => void;
  onLogout: () => void;
  isMock: boolean;
  onToggleMock: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  diff,
  onOpenSubscriptions,
  onOpenSettings,
  onOpenApply,
  onLogout,
  isMock,
  onToggleMock,
}) => {
  const hasChanges = diff && (diff.added_ports?.length > 0 || diff.removed_ports?.length > 0);

  return (
    <header className="sticky top-0 z-30 bg-slate-900/90 backdrop-blur-md border-b border-slate-800 px-6 py-3">
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-600 flex items-center justify-center shadow-lg shadow-sky-500/20">
            <Layers className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-lg text-white tracking-wide">ProxyBox</span>
              <span className="text-xs px-2 py-0.5 rounded bg-sky-950 text-sky-400 border border-sky-800 font-mono">
                Mihomo Core
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
            <p className="text-xs text-slate-400">订阅合并 · 前置/住宅链式代理 · 独立端口映射</p>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-3">
          {/* Subscriptions */}
          <button
            onClick={onOpenSubscriptions}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-medium border border-slate-700 transition"
          >
            <Rss className="w-4 h-4 text-sky-400" />
            <span>订阅管理</span>
          </button>

          {/* Settings */}
          <button
            onClick={onOpenSettings}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-medium border border-slate-700 transition"
          >
            <Settings className="w-4 h-4 text-slate-400" />
            <span>系统设置</span>
          </button>

          {/* Apply Config Button */}
          <button
            onClick={onOpenApply}
            className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition shadow-md ${
              hasChanges
                ? 'bg-sky-500 hover:bg-sky-400 text-white shadow-sky-500/25 animate-pulse'
                : 'bg-indigo-600 hover:bg-indigo-500 text-white shadow-indigo-600/20'
            }`}
          >
            <Send className="w-4 h-4" />
            <span>应用配置</span>
            {hasChanges && (
              <span className="w-2 h-2 rounded-full bg-amber-300"></span>
            )}
          </button>

          {/* Logout */}
          <button
            onClick={onLogout}
            title="退出登录"
            className="p-2 rounded-lg bg-slate-800 hover:bg-red-950/50 hover:text-red-400 text-slate-400 border border-slate-700 transition"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>
  );
};
