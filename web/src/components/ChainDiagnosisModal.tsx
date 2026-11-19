import React from 'react';
import {
  X,
  Activity,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Server,
  Radio,
  Network,
  Globe2,
  ArrowRight,
} from 'lucide-react';
import { ChainDiagnosis } from '../types';

interface ChainDiagnosisModalProps {
  isOpen: boolean;
  onClose: () => void;
  chainName: string;
  listenPort: number;
  loading: boolean;
  diagnosis: ChainDiagnosis | null;
  onReDiagnose: () => void;
}

export const ChainDiagnosisModal: React.FC<ChainDiagnosisModalProps> = ({
  isOpen,
  onClose,
  chainName,
  listenPort,
  loading,
  diagnosis,
  onReDiagnose,
}) => {
  if (!isOpen) return null;

  const isHealthy =
    diagnosis &&
    diagnosis.frontend_ok &&
    diagnosis.landing_ok &&
    diagnosis.chain_ok &&
    diagnosis.port_ok &&
    Boolean(diagnosis.egress_ip);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="p-5 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-2xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                链路多层诊断 · 端口 {listenPort}
              </h3>
              <p className="text-xs text-slate-400 truncate max-w-sm">{chainName}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white rounded-xl hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Body */}
        <div className="p-6 space-y-5 overflow-y-auto flex-1 text-xs">
          {loading ? (
            <div className="py-12 flex flex-col items-center justify-center gap-3 text-slate-400">
              <RefreshCw className="w-7 h-7 animate-spin text-sky-400" />
              <p className="text-sm font-medium">正在逐层测试前置、落地、链式代理及出口 IP...</p>
              <p className="text-[11px] text-slate-500">此过程约需 2~5 秒，请稍候</p>
            </div>
          ) : !diagnosis ? (
            <div className="py-8 text-center text-slate-500">暂无诊断数据</div>
          ) : (
            <>
              {/* Summary Banner */}
              <div
                className={`p-4 rounded-2xl border flex items-start gap-3.5 ${
                  isHealthy
                    ? 'bg-emerald-950/40 border-emerald-800/80 text-emerald-300'
                    : 'bg-amber-950/40 border-amber-800/80 text-amber-300'
                }`}
              >
                {isHealthy ? (
                  <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
                )}
                <div className="space-y-1">
                  <p className="font-semibold text-sm">{diagnosis.diagnosis_summary}</p>
                  <p className="text-[11px] opacity-90">建议操作：{diagnosis.suggested_action}</p>
                </div>
              </div>

              {/* 5-Layer Inspection List */}
              <div className="space-y-2.5">
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  分层连通性检测详情
                </p>

                {/* Layer 1: Frontend */}
                <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <Server className="w-4 h-4 text-sky-400" />
                    <div>
                      <p className="font-semibold text-slate-200">1. 前置中继节点</p>
                      <p className="text-[11px] text-slate-400">
                        {diagnosis.frontend_status === 'direct'
                          ? '直连模式 (不使用前置)'
                          : diagnosis.frontend_status === 'missing'
                          ? '⚠️ 节点已缺失，未找到配置'
                          : diagnosis.frontend_ok
                          ? `直连测速正常 (${diagnosis.frontend_delay}ms)`
                          : '直连测速超时/失败'}
                      </p>
                    </div>
                  </div>
                  <div>
                    {diagnosis.frontend_ok ? (
                      <span className="px-2 py-0.5 bg-emerald-950 border border-emerald-800 text-emerald-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> 正常
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-rose-950 border border-rose-800 text-rose-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" /> 异常
                      </span>
                    )}
                  </div>
                </div>

                {/* Layer 2: Landing */}
                <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <Radio className="w-4 h-4 text-indigo-400" />
                    <div>
                      <p className="font-semibold text-slate-200">2. 住宅落地节点</p>
                      <p className="text-[11px] text-slate-400">
                        {diagnosis.landing_status === 'missing'
                          ? '⚠️ 节点已缺失，未找到配置'
                          : diagnosis.landing_ok
                          ? `直连测速正常 (${diagnosis.landing_delay}ms)`
                          : '直连超时（原生专线需经前置）'}
                      </p>
                    </div>
                  </div>
                  <div>
                    {diagnosis.landing_ok ? (
                      <span className="px-2 py-0.5 bg-emerald-950 border border-emerald-800 text-emerald-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> 正常
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-amber-950 border border-amber-800 text-amber-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <AlertTriangle className="w-3.5 h-3.5" /> 专线依赖
                      </span>
                    )}
                  </div>
                </div>

                {/* Layer 3: Chained Link */}
                <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <Network className="w-4 h-4 text-purple-400" />
                    <div>
                      <p className="font-semibold text-slate-200">3. 链式代理握手 (前置 → 落地)</p>
                      <p className="text-[11px] text-slate-400">
                        {diagnosis.chain_ok
                          ? `链式拨号成功 (${diagnosis.chain_delay}ms)`
                          : '前置无法握手落地节点'}
                      </p>
                    </div>
                  </div>
                  <div>
                    {diagnosis.chain_ok ? (
                      <span className="px-2 py-0.5 bg-emerald-950 border border-emerald-800 text-emerald-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> 可链式
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-rose-950 border border-rose-800 text-rose-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" /> 握手失败
                      </span>
                    )}
                  </div>
                </div>

                {/* Layer 4: Production Port */}
                <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <Radio className="w-4 h-4 text-emerald-400" />
                    <div>
                      <p className="font-semibold text-slate-200">4. 内核本地监听器 (端口 {listenPort})</p>
                      <p className="text-[11px] text-slate-400">
                        {diagnosis.port_ok
                          ? '端口监听活跃，响应正常'
                          : `端口未响应: ${diagnosis.port_error || '连接被拒'}`}
                      </p>
                    </div>
                  </div>
                  <div>
                    {diagnosis.port_ok ? (
                      <span className="px-2 py-0.5 bg-emerald-950 border border-emerald-800 text-emerald-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> 监听正常
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-rose-950 border border-rose-800 text-rose-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" /> 未就绪
                      </span>
                    )}
                  </div>
                </div>

                {/* Layer 5: Egress IP Detection */}
                <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <Globe2 className="w-4 h-4 text-amber-400" />
                    <div>
                      <p className="font-semibold text-slate-200">5. 真实出口 IP 探测</p>
                      <p className="text-[11px] text-slate-400">
                        {diagnosis.egress_ip
                          ? `${diagnosis.egress_ip} (${diagnosis.egress_country})`
                          : diagnosis.egress_error || '出口探测超时'}
                      </p>
                    </div>
                  </div>
                  <div>
                    {diagnosis.egress_ip ? (
                      <span className="px-2 py-0.5 bg-emerald-950 border border-emerald-800 text-emerald-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" /> 出口正常
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-rose-950 border border-rose-800 text-rose-400 rounded-lg text-[11px] font-medium flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" /> 探测失败
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 bg-slate-950 border-t border-slate-800 flex items-center justify-between">
          <button
            onClick={onReDiagnose}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-white rounded-xl text-xs font-medium transition disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>重新诊断</span>
          </button>

          <button
            onClick={onClose}
            className="px-4 py-1.5 bg-sky-600 hover:bg-sky-500 text-white rounded-xl text-xs font-semibold transition"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  );
};
