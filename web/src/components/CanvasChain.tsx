import React, { useState } from 'react';
import {
  Link2,
  Copy,
  Check,
  Power,
  Trash2,
  Plus,
  AlertTriangle,
  Radio,
  ExternalLink,
  ShieldAlert,
  HelpCircle,
  Network,
} from 'lucide-react';
import { Chain, Node } from '../types';

interface CanvasChainProps {
  chains: Chain[];
  frontendNodes: Node[];
  landingNodes: Node[];
  onAddChain: (landingNodeId: string, frontendNodeId?: string) => Promise<void>;
  onUpdateChain: (id: string, data: Partial<Chain>) => Promise<void>;
  onDeleteChain: (id: string) => Promise<void>;
  onDetectEgress: (id: string) => Promise<{ ip: string; country: string }>;
  selectedFrontendId: string | null;
  onSelectFrontend: (id: string | null) => void;
}

export const CanvasChain: React.FC<CanvasChainProps> = ({
  chains,
  frontendNodes,
  landingNodes,
  onAddChain,
  onUpdateChain,
  onDeleteChain,
  onDetectEgress,
  selectedFrontendId,
  onSelectFrontend,
}) => {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [detectingId, setDetectingId] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedLandingToAdd, setSelectedLandingToAdd] = useState<string>('');
  const [selectedFrontToAdd, setSelectedFrontToAdd] = useState<string>('');

  const serverHost = window.location.hostname || '服务器IP';

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleEgress = async (id: string) => {
    setDetectingId(id);
    try {
      await onDetectEgress(id);
    } catch (err: any) {
      alert(err.message || '检测出口 IP 失败');
    } finally {
      setDetectingId(null);
    }
  };

  const handleCreateNew = async () => {
    if (!selectedLandingToAdd) return;
    await onAddChain(selectedLandingToAdd, selectedFrontToAdd || undefined);
    setShowAddModal(false);
    setSelectedLandingToAdd('');
    setSelectedFrontToAdd('');
  };

  // Group frontend nodes that are actively used in chains or in frontend pool
  const activeFrontends = frontendNodes.length > 0 ? frontendNodes : [];

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800/80 pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Network className="w-5 h-5 text-sky-400" />
            <h2 className="text-base font-bold text-white">链路拓扑与端口映射画布</h2>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            左列为前置中继节点（可复用于多条链路），右列为落地住宅节点与独立分配端口
          </p>
        </div>

        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-1.5 px-3.5 py-1.5 bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-600/20 transition"
        >
          <Plus className="w-4 h-4" />
          <span>新建端口链路</span>
        </button>
      </div>

      {/* Main 2-Column Canvas */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Left Column: Frontend Relays (4 cols) */}
        <div className="lg:col-span-4 space-y-3">
          <div className="flex items-center justify-between pb-1">
            <span className="text-xs font-bold uppercase tracking-wider text-sky-400 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-sky-400"></span>
              前置中继节点 (一对多)
            </span>
            <span className="text-[11px] text-slate-500">点击选中高亮连线</span>
          </div>

          {/* Direct (No Frontend) Card */}
          <div
            onClick={() => onSelectFrontend(selectedFrontendId === 'direct' ? null : 'direct')}
            className={`p-3.5 rounded-xl border text-xs cursor-pointer transition flex items-center justify-between ${
              selectedFrontendId === 'direct'
                ? 'bg-slate-800 border-sky-400 ring-2 ring-sky-500/20'
                : 'bg-slate-950/70 border-slate-800 hover:border-slate-700'
            }`}
          >
            <div>
              <p className="font-semibold text-white">直连访问 (不经前置中继)</p>
              <p className="text-[11px] text-slate-500">住宅节点直接连接目标网络</p>
            </div>
            <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">DIRECT</span>
          </div>

          {/* Active Frontend Cards */}
          {activeFrontends.length === 0 ? (
            <p className="text-xs text-slate-600 py-6 text-center border border-dashed border-slate-800 rounded-xl">
              前置池为空，请先在上方三池区将节点移入「前置池」
            </p>
          ) : (
            activeFrontends.map((front) => {
              const isSelected = selectedFrontendId === front.id;
              const connectedChains = chains.filter((c) => c.frontend_node_id === front.id);

              return (
                <div
                  key={front.id}
                  onClick={() => onSelectFrontend(isSelected ? null : front.id)}
                  className={`p-3.5 rounded-xl border text-xs cursor-pointer transition space-y-2 ${
                    isSelected
                      ? 'bg-sky-950/40 border-sky-400 ring-2 ring-sky-500/20'
                      : 'bg-slate-950/70 border-slate-800 hover:border-slate-700'
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="truncate">
                      <p className="font-semibold text-white truncate">{front.original_name}</p>
                      <p className="text-[11px] text-slate-400 font-mono truncate">{front.server}:{front.port}</p>
                    </div>
                    <span className="text-[10px] px-2 py-0.5 rounded bg-sky-950 text-sky-400 border border-sky-800 uppercase font-mono">
                      {front.type}
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1 border-t border-slate-900">
                    <span className="text-sky-400 font-medium">已挂载 {connectedChains.length} 条链路</span>
                    {front.last_delay > 0 && (
                      <span className="font-mono text-slate-400">{front.last_delay} ms</span>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Right Column: Landing Chains & Mapped Ports (8 cols) */}
        <div className="lg:col-span-8 space-y-3">
          <div className="flex items-center justify-between pb-1">
            <span className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-indigo-400"></span>
              住宅落地链路与监听端口
            </span>
            <span className="text-[11px] text-slate-500">混合协议 Mixed (HTTP + SOCKS5)</span>
          </div>

          {chains.length === 0 ? (
            <div className="text-center py-12 border border-dashed border-slate-800 rounded-2xl p-6 text-slate-500 text-xs space-y-2">
              <p>暂无链路，点击右上角「新建端口链路」分配独立监听端口</p>
            </div>
          ) : (
            chains.map((chain) => {
              const socksUrl = `socks5://${chain.username}:${chain.password}@${serverHost}:${chain.listen_port}`;
              const httpUrl = `http://${chain.username}:${chain.password}@${serverHost}:${chain.listen_port}`;
              const isFrontMatched =
                selectedFrontendId === chain.frontend_node_id ||
                (!chain.frontend_node_id && selectedFrontendId === 'direct');

              return (
                <div
                  key={chain.id}
                  className={`bg-slate-950 border rounded-2xl p-4.5 space-y-3.5 transition ${
                    isFrontMatched
                      ? 'border-indigo-500/80 ring-2 ring-indigo-500/20 bg-indigo-950/10'
                      : 'border-slate-800/90 hover:border-slate-700'
                  }`}
                >
                  {/* Top Bar: Front -> Landing & Port */}
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    {/* Route connection info */}
                    <div className="flex items-center gap-2 text-xs">
                      <div className="flex items-center gap-1.5 bg-slate-900 px-2.5 py-1 rounded-lg border border-slate-800">
                        <span className="text-slate-400 font-medium">前置:</span>
                        <select
                          value={chain.frontend_node_id || ''}
                          onChange={(e) => {
                            const val = e.target.value;
                            onUpdateChain(chain.id, { frontend_node_id: val ? val : null });
                          }}
                          className="bg-transparent text-sky-400 text-xs font-semibold focus:outline-none cursor-pointer"
                        >
                          <option value="" className="bg-slate-900 text-slate-300">直连 (DIRECT)</option>
                          {frontendNodes.map((fn) => (
                            <option key={fn.id} value={fn.id} className="bg-slate-900 text-white">
                              {fn.original_name}
                            </option>
                          ))}
                        </select>
                      </div>

                      <span className="text-slate-600 font-bold">→</span>

                      <div className="flex items-center gap-1.5 bg-slate-900 px-2.5 py-1 rounded-lg border border-slate-800">
                        <span className="text-slate-400 font-medium">落地:</span>
                        <span className="text-indigo-400 font-semibold truncate max-w-[180px]">
                          {chain.landing_node?.original_name || '未指定'}
                        </span>
                        <span className="text-[10px] px-1.5 py-0.2 rounded bg-indigo-950 text-indigo-300 uppercase font-mono">
                          {chain.landing_node?.type}
                        </span>
                      </div>
                    </div>

                    {/* Port and Toggle Switch */}
                    <div className="flex items-center gap-3">
                      <div className="flex items-center gap-1.5 bg-slate-900 px-3 py-1 rounded-lg border border-slate-700">
                        <span className="text-xs text-slate-400">端口:</span>
                        <input
                          type="number"
                          value={chain.listen_port}
                          onChange={(e) => {
                            const val = Number(e.target.value);
                            if (val > 0) onUpdateChain(chain.id, { listen_port: val });
                          }}
                          className="w-16 bg-transparent text-emerald-400 font-mono font-bold text-xs focus:outline-none"
                        />
                      </div>

                      {/* Enable Switch */}
                      <button
                        onClick={() => onUpdateChain(chain.id, { enabled: !chain.enabled })}
                        className={`p-1.5 rounded-lg border transition ${
                          chain.enabled
                            ? 'bg-emerald-950 text-emerald-400 border-emerald-800'
                            : 'bg-slate-800 text-slate-500 border-slate-700'
                        }`}
                        title={chain.enabled ? '已启用 (点击禁用)' : '已禁用 (点击启用)'}
                      >
                        <Power className="w-4 h-4" />
                      </button>

                      {/* Delete */}
                      <button
                        onClick={() => onDeleteChain(chain.id)}
                        className="p-1.5 rounded-lg text-slate-500 hover:text-red-400 hover:bg-red-950/40 transition"
                        title="删除链路"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* Middle: Status & Egress IP verification */}
                  <div className="flex flex-wrap items-center justify-between gap-3 text-xs bg-slate-900/60 p-2.5 rounded-xl border border-slate-800/80">
                    <div className="flex items-center gap-3">
                      {/* Status */}
                      <span
                        className={`flex items-center gap-1.5 font-medium ${
                          !chain.enabled
                            ? 'text-slate-500'
                            : chain.status === 'online'
                            ? 'text-emerald-400'
                            : 'text-amber-400'
                        }`}
                      >
                        <span
                          className={`w-2 h-2 rounded-full ${
                            !chain.enabled ? 'bg-slate-600' : chain.status === 'online' ? 'bg-emerald-400' : 'bg-amber-400'
                          }`}
                        />
                        {!chain.enabled ? '已禁用' : chain.status === 'online' ? '在线' : '待连接'}
                      </span>

                      {/* Egress Info */}
                      {chain.outbound_ip && (
                        <span className="text-slate-300 font-mono">
                          出口: <b>{chain.outbound_ip}</b> ({chain.outbound_country})
                        </span>
                      )}
                    </div>

                    <button
                      onClick={() => handleEgress(chain.id)}
                      disabled={detectingId === chain.id || !chain.enabled}
                      className="flex items-center gap-1 px-2.5 py-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-slate-300 hover:text-white rounded-lg text-[11px] transition"
                    >
                      <Radio className={`w-3.5 h-3.5 text-sky-400 ${detectingId === chain.id ? 'animate-pulse' : ''}`} />
                      <span>{detectingId === chain.id ? '正在探测出口...' : '探测指纹出口 IP'}</span>
                    </button>
                  </div>

                  {/* Bottom: Copy Proxy Strings for Fingerprint Browser */}
                  <div className="flex flex-wrap items-center gap-2 pt-1">
                    <span className="text-[11px] text-slate-400 font-medium">指纹浏览器代理地址:</span>

                    {/* SOCKS5 */}
                    <button
                      onClick={() => copyToClipboard(socksUrl, `socks_${chain.id}`)}
                      className="flex items-center gap-1.5 px-2.5 py-1 bg-slate-900 hover:bg-slate-800 text-slate-300 hover:text-white border border-slate-800 rounded-lg text-xs font-mono transition"
                    >
                      {copiedKey === `socks_${chain.id}` ? (
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                      ) : (
                        <Copy className="w-3.5 h-3.5 text-slate-400" />
                      )}
                      <span>复制 SOCKS5</span>
                    </button>

                    {/* HTTP */}
                    <button
                      onClick={() => copyToClipboard(httpUrl, `http_${chain.id}`)}
                      className="flex items-center gap-1.5 px-2.5 py-1 bg-slate-900 hover:bg-slate-800 text-slate-300 hover:text-white border border-slate-800 rounded-lg text-xs font-mono transition"
                    >
                      {copiedKey === `http_${chain.id}` ? (
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                      ) : (
                        <Copy className="w-3.5 h-3.5 text-slate-400" />
                      )}
                      <span>复制 HTTP</span>
                    </button>

                    <span className="text-[11px] text-slate-500 font-mono ml-auto">
                      账号: {chain.username} / 密码: {chain.password}
                    </span>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Add Chain Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-bold text-white">新建链路端口映射</h3>

            <div className="space-y-3 text-xs">
              <div>
                <label className="text-slate-400 block mb-1">选择落地住宅节点 (必须)</label>
                <select
                  value={selectedLandingToAdd}
                  onChange={(e) => setSelectedLandingToAdd(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-white"
                >
                  <option value="">-- 请选择落地节点 --</option>
                  {landingNodes.map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.original_name} ({n.type})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="text-slate-400 block mb-1">选择前置中继节点 (可选，为空表示直连落地)</label>
                <select
                  value={selectedFrontToAdd}
                  onChange={(e) => setSelectedFrontToAdd(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-white"
                >
                  <option value="">直连 (无需前置中继)</option>
                  {frontendNodes.map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.original_name} ({n.type})
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setShowAddModal(false)}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs"
              >
                取消
              </button>
              <button
                onClick={handleCreateNew}
                disabled={!selectedLandingToAdd}
                className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white rounded-xl text-xs font-semibold"
              >
                创建链路并自动分配端口
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
