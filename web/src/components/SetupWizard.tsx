import React, { useState } from 'react';
import { ShieldCheck, Lock, ArrowRight, Layers } from 'lucide-react';

interface SetupWizardProps {
  isSetup: boolean; // true = first time setup, false = regular login
  onSubmit: (password: string) => Promise<void>;
  onContinueAsMock: () => void;
}

export const SetupWizard: React.FC<SetupWizardProps> = ({ isSetup, onSubmit, onContinueAsMock }) => {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) return;
    if (isSetup && password !== confirmPassword) {
      setError('两次输入的密码不一致');
      return;
    }
    if (isSetup && password.length < 6) {
      setError('密码长度至少为 6 个字符');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await onSubmit(password);
    } catch (err: any) {
      setError(err.message || '操作失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950 p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-md p-8 shadow-2xl space-y-6">
        <div className="text-center space-y-2">
          <div className="w-14 h-14 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-600 flex items-center justify-center mx-auto shadow-xl shadow-sky-500/25">
            <Layers className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-xl font-bold text-white tracking-wide">ProxyBox 管理面板</h1>
          <p className="text-xs text-slate-400">
            {isSetup ? '欢迎使用！首次启动请初始化设置管理员密码' : '请输入管理员密码以进入控制台'}
          </p>
        </div>

        {error && (
          <div className="p-3 bg-red-950/60 border border-red-800 text-red-300 text-xs rounded-xl text-center">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-xs text-slate-400 block mb-1">
              {isSetup ? '设置管理密码 (至少 6 位)' : '管理员密码'}
            </label>
            <div className="relative">
              <Lock className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-9 pr-4 py-2.5 text-sm text-white focus:outline-none focus:border-sky-500"
                required
              />
            </div>
          </div>

          {isSetup && (
            <div>
              <label className="text-xs text-slate-400 block mb-1">再次确认密码</label>
              <div className="relative">
                <Lock className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="password"
                  placeholder="••••••••"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-9 pr-4 py-2.5 text-sm text-white focus:outline-none focus:border-sky-500"
                  required
                />
              </div>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 disabled:opacity-50 text-white font-semibold rounded-xl text-sm transition flex items-center justify-center gap-2 shadow-lg shadow-sky-600/25"
          >
            <span>{loading ? '验证中...' : isSetup ? '完成初始化并进入' : '立即登录'}</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>

        <div className="text-center pt-2 border-t border-slate-800/80">
          <button
            onClick={onContinueAsMock}
            className="text-xs text-slate-500 hover:text-sky-400 transition"
          >
            跳过并以「离线 Mock 演示模式」预览界面 →
          </button>
        </div>
      </div>
    </div>
  );
};
