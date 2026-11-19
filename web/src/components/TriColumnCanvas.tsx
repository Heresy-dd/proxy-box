import React, { useState, useEffect, useRef, useCallback } from 'react';
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
  Activity,
  Eye,
  EyeOff,
  Server,
  Zap,
  Star,
  GripVertical,
  RefreshCw,
  Search,
  Sparkles,
} from 'lucide-react';
import { Chain, Node, ChainDiagnosis } from '../types';
import { ChainDiagnosisModal } from './ChainDiagnosisModal';
import { ApiClient } from '../api/client';

interface TriColumnCanvasProps {
  chains: Chain[];
  frontendNodes: Node[];
  landingNodes: Node[];
  selectedFrontendId: string | null;
  onSelectFrontend: (id: string | null) => void;
  onAddChain: (landingNodeId: string, frontendNodeId?: string) => Promise<void>;
  onUpdateChain: (id: string, data: Partial<Chain>) => Promise<void>;
  onDeleteChain: (id: string) => Promise<void>;
  onDetectEgress: (id: string) => Promise<{ ip: string; country: string }>;
  onToggleFavorite: (nodeId: string, isFav: boolean) => Promise<void>;
  onStartSpeedtest: (frontId: string) => void;
  speedtesting: boolean;
  landingDelays: Record<string, number>;
  landingStatusLabels?: Record<string, string>;
  frontDelay: number | null;
}

interface AnchorCoord {
  x: number;
  y: number;
}

interface DraftSource {
  type: 'frontend' | 'landing';
  id: string; // node ID or 'direct'
}

export const TriColumnCanvas: React.FC<TriColumnCanvasProps> = ({
  chains,
  frontendNodes,
  landingNodes,
  selectedFrontendId,
  onSelectFrontend,
  onAddChain,
  onUpdateChain,
  onDeleteChain,
  onDetectEgress,
  onToggleFavorite,
  onStartSpeedtest,
  speedtesting,
  landingDelays,
  landingStatusLabels = {},
  frontDelay,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [connections, setConnections] = useState<
    Array<{
      id: string;
      d: string;
      highlighted: boolean;
      type: 'front-landing' | 'landing-port';
      chainId: string;
    }>
  >([]);

  // Draft anchor state
  const [draftSource, setDraftSource] = useState<DraftSource | null>(null);

  // Masking toggles for port passwords
  const [revealedPasswords, setRevealedPasswords] = useState<Record<string, boolean>>({});

  // Diagnosis modal state
  const [diagnosisModalOpen, setDiagnosisModalOpen] = useState(false);
  const [diagnosingChain, setDiagnosingChain] = useState<Chain | null>(null);
  const [diagnosisData, setDiagnosisData] = useState<ChainDiagnosis | null>(null);
  const [diagnosisLoading, setDiagnosisLoading] = useState(false);

  // Search & Copy feedback
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [detectingId, setDetectingId] = useState<string | null>(null);
  const [frontSearch, setFrontSearch] = useState('');
  const [landingSearch, setLandingSearch] = useState('');
  const [portSearch, setPortSearch] = useState('');

  const serverHost = window.location.hostname || '127.0.0.1';

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const togglePasswordVisibility = (chainId: string) => {
    setRevealedPasswords((prev) => ({
      ...prev,
      [chainId]: !prev[chainId],
    }));
  };

  // Run layered diagnosis
  const handleOpenDiagnosis = async (chain: Chain) => {
    setDiagnosingChain(chain);
    setDiagnosisModalOpen(true);
    setDiagnosisLoading(true);
    setDiagnosisData(null);
    try {
      const res = await ApiClient.diagnoseChain(chain.id);
      setDiagnosisData(res);
    } catch (err: any) {
      alert(err.message || '诊断请求失败');
    } finally {
      setDiagnosisLoading(false);
    }
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

  // Recalculate Bézier curves for lines
  const updateSvgCurves = useCallback(() => {
    if (!containerRef.current) return;
    const containerRect = containerRef.current.getBoundingClientRect();

    const getCenter = (el: HTMLElement | null): AnchorCoord | null => {
      if (!el) return null;
      const rect = el.getBoundingClientRect();
      return {
        x: rect.left + rect.width / 2 - containerRect.left,
        y: rect.top + rect.height / 2 - containerRect.top,
      };
    };

    const newConnections: typeof connections = [];

    chains.forEach((chain) => {
      // 1. Frontend -> Landing curve
      const frontAnchorId = chain.frontend_node_id
        ? `anchor-front-${chain.frontend_node_id}`
        : 'anchor-front-direct';
      const landingInAnchorId = `anchor-landing-in-${chain.landing_node_id}`;

      const fEl = document.getElementById(frontAnchorId);
      const lInEl = document.getElementById(landingInAnchorId);

      const fCoord = getCenter(fEl);
      const lInCoord = getCenter(lInEl);

      if (fCoord && lInCoord) {
        const dx = Math.max(30, (lInCoord.x - fCoord.x) / 2);
        const d = `M ${fCoord.x} ${fCoord.y} C ${fCoord.x + dx} ${fCoord.y}, ${lInCoord.x - dx} ${lInCoord.y}, ${lInCoord.x} ${lInCoord.y}`;
        const isHighlighted =
          selectedFrontendId === chain.frontend_node_id ||
          (!chain.frontend_node_id && selectedFrontendId === 'direct');

        newConnections.push({
          id: `f-${chain.id}`,
          d,
          highlighted: Boolean(isHighlighted),
          type: 'front-landing',
          chainId: chain.id,
        });
      }

      // 2. Landing -> Port curve
      const landingOutAnchorId = `anchor-landing-out-${chain.landing_node_id}`;
      const portInAnchorId = `anchor-port-in-${chain.id}`;

      const lOutEl = document.getElementById(landingOutAnchorId);
      const pInEl = document.getElementById(portInAnchorId);

      const lOutCoord = getCenter(lOutEl);
      const pInCoord = getCenter(pInEl);

      if (lOutCoord && pInCoord) {
        const dx = Math.max(30, (pInCoord.x - lOutCoord.x) / 2);
        const d = `M ${lOutCoord.x} ${lOutCoord.y} C ${lOutCoord.x + dx} ${lOutCoord.y}, ${pInCoord.x - dx} ${pInCoord.y}, ${pInCoord.x} ${pInCoord.y}`;
        const isHighlighted =
          selectedFrontendId === chain.frontend_node_id ||
          (!chain.frontend_node_id && selectedFrontendId === 'direct');

        newConnections.push({
          id: `l-${chain.id}`,
          d,
          highlighted: Boolean(isHighlighted),
          type: 'landing-port',
          chainId: chain.id,
        });
      }
    });

    setConnections(newConnections);
  }, [chains, selectedFrontendId]);

  useEffect(() => {
    updateSvgCurves();
    const timer = setTimeout(updateSvgCurves, 100);

    const observer = new ResizeObserver(() => updateSvgCurves());
    if (containerRef.current) observer.observe(containerRef.current);

    window.addEventListener('resize', updateSvgCurves);
    window.addEventListener('scroll', updateSvgCurves);

    return () => {
      clearTimeout(timer);
      observer.disconnect();
      window.removeEventListener('resize', updateSvgCurves);
      window.removeEventListener('scroll', updateSvgCurves);
    };
  }, [updateSvgCurves, frontendNodes, landingNodes, chains]);

  // Handle anchor clicks for draft connections
  const handleAnchorClick = async (type: 'frontend' | 'landing' | 'port', id: string) => {
    if (type === 'frontend') {
      if (draftSource && draftSource.type === 'frontend' && draftSource.id === id) {
        setDraftSource(null);
      } else {
        setDraftSource({ type: 'frontend', id });
      }
      return;
    }

    if (type === 'landing') {
      if (draftSource && draftSource.type === 'frontend') {
        // Create new chain: frontend -> landing
        const fid = draftSource.id === 'direct' ? undefined : draftSource.id;
        await onAddChain(id, fid);
        setDraftSource(null);
      } else {
        setDraftSource({ type: 'landing', id });
      }
      return;
    }

    if (type === 'port') {
      if (draftSource && draftSource.type === 'landing') {
        // Re-bind chain's landing node to draft landing node
        await onUpdateChain(id, { landing_node_id: draftSource.id });
        setDraftSource(null);
      }
    }
  };

  // Filtered lists
  const filteredFrontends = frontendNodes.filter((n) =>
    n.original_name.toLowerCase().includes(frontSearch.toLowerCase()) ||
    n.type.toLowerCase().includes(frontSearch.toLowerCase())
  );

  const filteredLandings = landingNodes.filter((n) =>
    n.original_name.toLowerCase().includes(landingSearch.toLowerCase()) ||
    n.type.toLowerCase().includes(landingSearch.toLowerCase())
  );

  const filteredChains = chains.filter((c) =>
    c.name.toLowerCase().includes(portSearch.toLowerCase()) ||
    c.listen_port.toString().includes(portSearch) ||
    (c.landing_node?.original_name && c.landing_node.original_name.toLowerCase().includes(portSearch.toLowerCase()))
  );

  return (
    <div
      ref={containerRef}
      className="relative bg-slate-900/60 border border-slate-800 rounded-3xl p-6 shadow-2xl space-y-6 overflow-hidden"
    >
      {/* SVG Connection Layer */}
      <svg className="absolute inset-0 w-full h-full pointer-events-none z-10">
        <defs>
          <linearGradient id="gradient-front" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#38bdf8" />
            <stop offset="100%" stopColor="#818cf8" />
          </linearGradient>
          <linearGradient id="gradient-landing" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#818cf8" />
            <stop offset="100%" stopColor="#a855f7" />
          </linearGradient>
        </defs>

        {connections.map((c) => (
          <path
            key={c.id}
            d={c.d}
            fill="none"
            stroke={
              c.highlighted
                ? c.type === 'front-landing'
                  ? '#38bdf8'
                  : '#a855f7'
                : '#475569'
            }
            strokeWidth={c.highlighted ? 3 : 1.5}
            strokeOpacity={c.highlighted ? 0.95 : 0.4}
            strokeDasharray={c.highlighted ? 'none' : '4,3'}
            className="transition-all duration-300"
          />
        ))}
      </svg>

      {/* Top Banner / Canvas Header */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800/80 pb-4 relative z-20">
        <div>
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400">
              <Network className="w-4 h-4" />
            </div>
            <h2 className="text-base font-bold text-white tracking-wide">
              ProxyBox 链式代理拓扑画布 (3 列同屏交互)
            </h2>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            左列前置中继（点击聚焦+测速） · 中列住宅落地（实时链式延迟） · 右列独立映射端口（指纹浏览器出口）
          </p>
        </div>

        {/* Draft Notice if anchor selected */}
        {draftSource && (
          <div className="flex items-center gap-2 px-3 py-1.5 bg-sky-950/80 border border-sky-500 rounded-xl text-xs text-sky-300 animate-pulse">
            <Sparkles className="w-3.5 h-3.5 text-sky-400" />
            <span>
              已选锚点：{draftSource.type === 'frontend' ? '前置节点' : '住宅落地'}，点击下一列锚点完成连线
            </span>
            <button
              onClick={() => setDraftSource(null)}
              className="ml-1 text-slate-400 hover:text-white font-bold"
            >
              取消
            </button>
          </div>
        )}
      </div>

      {/* 3 Parallel Columns on the Same Horizon */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 relative z-20">
        {/* =========================================
            COLUMN 1: 前置中继池 (Frontend Relay)
            ========================================= */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-2xl p-4 flex flex-col space-y-3.5">
          <div className="flex items-center justify-between pb-1 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-sky-400 shadow-sm shadow-sky-400/50"></span>
              <h3 className="text-xs font-bold text-sky-400 uppercase tracking-wider">
                1. 前置中继池 ({frontendNodes.length})
              </h3>
            </div>
            <span className="text-[11px] text-slate-500">点击聚焦测速</span>
          </div>

          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索前置节点..."
              value={frontSearch}
              onChange={(e) => setFrontSearch(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-sky-500"
            />
          </div>

          <div className="space-y-2.5 flex-1 overflow-y-auto max-h-[720px] pr-1">
            {/* Direct Card */}
            <div
              onClick={() => onSelectFrontend(selectedFrontendId === 'direct' ? null : 'direct')}
              className={`relative p-3.5 rounded-xl border text-xs cursor-pointer transition flex items-center justify-between group ${
                selectedFrontendId === 'direct'
                  ? 'bg-sky-950/40 border-sky-400 ring-2 ring-sky-500/20'
                  : 'bg-slate-900/70 border-slate-800 hover:border-slate-700'
              }`}
            >
              <div>
                <p className="font-semibold text-white">直连访问 (DIRECT)</p>
                <p className="text-[11px] text-slate-500">不经前置中继，落地节点直出</p>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
                  DIRECT
                </span>
                <button
                  id="anchor-front-direct"
                  title="连线此节点"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleAnchorClick('frontend', 'direct');
                  }}
                  className={`w-6 h-6 rounded-full flex items-center justify-center transition border ${
                    draftSource?.type === 'frontend' && draftSource.id === 'direct'
                      ? 'bg-sky-500 border-white text-white shadow-lg'
                      : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-sky-600 hover:text-white'
                  }`}
                >
                  <Plus className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            {/* Frontend Cards */}
            {filteredFrontends.length === 0 ? (
              <div className="py-10 text-center text-slate-500 text-xs border border-dashed border-slate-800 rounded-xl">
                前置池为空，请在上方将机场节点移入前置池
              </div>
            ) : (
              filteredFrontends.map((front) => {
                const isSelected = selectedFrontendId === front.id;
                const isDrafting = draftSource?.type === 'frontend' && draftSource.id === front.id;
                const connectedCount = chains.filter((c) => c.frontend_node_id === front.id).length;

                return (
                  <div
                    key={front.id}
                    onClick={() => {
                      onSelectFrontend(isSelected ? null : front.id);
                      if (!isSelected) {
                        onStartSpeedtest(front.id);
                      }
                    }}
                    className={`relative p-3.5 rounded-xl border text-xs cursor-pointer transition space-y-2 group ${
                      isSelected
                        ? 'bg-sky-950/40 border-sky-400 ring-2 ring-sky-500/20'
                        : 'bg-slate-900/70 border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="truncate flex-1">
                        <div className="flex items-center gap-1.5">
                          <p className="font-semibold text-white truncate">{front.original_name}</p>
                          {front.is_favorite && (
                            <Star className="w-3.5 h-3.5 text-amber-400 fill-amber-400 shrink-0" />
                          )}
                        </div>
                        <p className="text-[11px] text-slate-400 font-mono truncate">
                          {front.server}:{front.port}
                        </p>
                      </div>

                      <div className="flex items-center gap-1.5 shrink-0">
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-sky-950 border border-sky-800 text-sky-400 font-mono uppercase">
                          {front.type}
                        </span>

                        {/* Anchor Button */}
                        <button
                          id={`anchor-front-${front.id}`}
                          title="连线此节点"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleAnchorClick('frontend', front.id);
                          }}
                          className={`w-6 h-6 rounded-full flex items-center justify-center transition border ${
                            isDrafting
                              ? 'bg-sky-500 border-white text-white shadow-lg'
                              : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-sky-600 hover:text-white'
                          }`}
                        >
                          <Plus className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1.5 border-t border-slate-800/60">
                      <span className="text-sky-400 font-medium">已挂载 {connectedCount} 条链路</span>
                      <div className="flex items-center gap-2">
                        {front.last_delay > 0 && (
                          <span className="font-mono text-slate-300 font-semibold">
                            {front.last_delay} ms
                          </span>
                        )}
                        <button
                          title="触发链式测速"
                          onClick={(e) => {
                            e.stopPropagation();
                            onSelectFrontend(front.id);
                            onStartSpeedtest(front.id);
                          }}
                          className="px-2 py-0.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded text-[10px] flex items-center gap-1"
                        >
                          <Zap className="w-2.5 h-2.5 text-amber-400" />
                          <span>测速</span>
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* =========================================
            COLUMN 2: 住宅落地池 (Residential Landing)
            ========================================= */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-2xl p-4 flex flex-col space-y-3.5">
          <div className="flex items-center justify-between pb-1 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-indigo-400 shadow-sm shadow-indigo-400/50"></span>
              <h3 className="text-xs font-bold text-indigo-400 uppercase tracking-wider">
                2. 住宅落地池 ({landingNodes.length})
              </h3>
            </div>
            {speedtesting ? (
              <span className="text-[11px] text-sky-400 flex items-center gap-1 animate-pulse">
                <RefreshCw className="w-3 h-3 animate-spin" />
                正在链式测速...
              </span>
            ) : (
              <span className="text-[11px] text-slate-500">双向锚点</span>
            )}
          </div>

          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索落地节点..."
              value={landingSearch}
              onChange={(e) => setLandingSearch(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div className="space-y-2.5 flex-1 overflow-y-auto max-h-[720px] pr-1">
            {filteredLandings.length === 0 ? (
              <div className="py-10 text-center text-slate-500 text-xs border border-dashed border-slate-800 rounded-xl">
                住宅池为空，请在上方将住宅专线节点移入住宅池
              </div>
            ) : (
              filteredLandings.map((landing) => {
                const chainDelay = landingDelays[landing.id];
                const statusLabel = landingStatusLabels[landing.id];
                const isDrafting = draftSource?.type === 'landing' && draftSource.id === landing.id;
                const isAssigned = chains.some((c) => c.landing_node_id === landing.id);

                return (
                  <div
                    key={landing.id}
                    className={`relative p-3.5 rounded-xl border text-xs transition space-y-2 group ${
                      isDrafting
                        ? 'bg-indigo-950/40 border-indigo-500 ring-2 ring-indigo-500/20'
                        : isAssigned
                        ? 'bg-slate-900/90 border-slate-700/80'
                        : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      {/* Left Anchor (Incoming from Frontend) */}
                      <button
                        id={`anchor-landing-in-${landing.id}`}
                        title="接收前置节点连线"
                        onClick={() => handleAnchorClick('landing', landing.id)}
                        className="w-6 h-6 rounded-full bg-slate-800 border border-slate-700 hover:bg-sky-600 hover:text-white text-slate-300 flex items-center justify-center shrink-0 transition"
                      >
                        <Plus className="w-3.5 h-3.5" />
                      </button>

                      <div className="truncate flex-1">
                        <div className="flex items-center gap-1.5">
                          <p className="font-semibold text-white truncate">{landing.original_name}</p>
                          {landing.is_favorite && (
                            <Star className="w-3.5 h-3.5 text-amber-400 fill-amber-400 shrink-0" />
                          )}
                        </div>
                        <p className="text-[11px] text-slate-400 font-mono truncate">
                          {landing.server}:{landing.port}
                        </p>
                      </div>

                      {/* Right Anchor (Outgoing to Port) */}
                      <div className="flex items-center gap-1.5 shrink-0">
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-950 border border-indigo-800 text-indigo-400 font-mono uppercase">
                          {landing.type}
                        </span>

                        <button
                          id={`anchor-landing-out-${landing.id}`}
                          title="连线到端口映射"
                          onClick={() => handleAnchorClick('landing', landing.id)}
                          className={`w-6 h-6 rounded-full flex items-center justify-center transition border ${
                            isDrafting
                              ? 'bg-indigo-500 border-white text-white shadow-lg'
                              : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-indigo-600 hover:text-white'
                          }`}
                        >
                          <Plus className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>

                    {/* Bottom Status / Latency Row */}
                    <div className="flex items-center justify-between text-[11px] pt-1.5 border-t border-slate-800/60">
                      <div>
                        {chainDelay !== undefined ? (
                          chainDelay > 0 ? (
                            <span className="text-emerald-400 font-medium font-mono">
                              {chainDelay} ms (链式通)
                            </span>
                          ) : (
                            <span className="text-rose-400 font-medium">
                              {statusLabel || '连接超时'}
                            </span>
                          )
                        ) : landing.last_delay > 0 ? (
                          <span className="text-slate-400 font-mono">
                            直连 {landing.last_delay} ms
                          </span>
                        ) : (
                          <span className="text-slate-500">未测速</span>
                        )}
                      </div>

                      <button
                        onClick={() => onAddChain(landing.id, selectedFrontendId || undefined)}
                        className="px-2 py-0.5 bg-indigo-600/20 hover:bg-indigo-600 text-indigo-300 hover:text-white rounded border border-indigo-500/30 text-[10px] transition"
                      >
                        + 映射新端口
                      </button>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* =========================================
            COLUMN 3: 端口映射池 (Port Mapping)
            ========================================= */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-2xl p-4 flex flex-col space-y-3.5">
          <div className="flex items-center justify-between pb-1 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-purple-400 shadow-sm shadow-purple-400/50"></span>
              <h3 className="text-xs font-bold text-purple-400 uppercase tracking-wider">
                3. 端口映射池 ({chains.length})
              </h3>
            </div>
            <span className="text-[11px] text-slate-500">混合协议 Mixed</span>
          </div>

          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索端口 / 链路名称..."
              value={portSearch}
              onChange={(e) => setPortSearch(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-purple-500"
            />
          </div>

          <div className="space-y-3 flex-1 overflow-y-auto max-h-[720px] pr-1">
            {filteredChains.length === 0 ? (
              <div className="py-10 text-center text-slate-500 text-xs border border-dashed border-slate-800 rounded-xl">
                暂无端口映射，点击中列落地节点的「+ 映射新端口」快速创建
              </div>
            ) : (
              filteredChains.map((chain) => {
                const socksUrl = `socks5://${chain.username}:${chain.password}@${serverHost}:${chain.listen_port}`;
                const httpUrl = `http://${chain.username}:${chain.password}@${serverHost}:${chain.listen_port}`;
                const fingerprintFmt = `${serverHost}:${chain.listen_port}:${chain.username}:${chain.password}`;
                const isRevealed = revealedPasswords[chain.id] || false;

                const hasMissingFront = chain.frontend_node_status === 'missing';
                const hasMissingLanding = chain.landing_node_status === 'missing';

                return (
                  <div
                    key={chain.id}
                    className="relative bg-slate-900/80 border border-slate-800 hover:border-slate-700 rounded-2xl p-4 space-y-3 transition"
                  >
                    {/* Header Row */}
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        {/* Anchor point on left edge */}
                        <button
                          id={`anchor-port-in-${chain.id}`}
                          title="连线此端口"
                          onClick={() => handleAnchorClick('port', chain.id)}
                          className="w-6 h-6 rounded-full bg-slate-800 border border-slate-700 hover:bg-purple-600 hover:text-white text-slate-300 flex items-center justify-center shrink-0 transition"
                        >
                          <Plus className="w-3.5 h-3.5" />
                        </button>

                        <div className="flex items-center gap-1.5">
                          <span className="font-mono font-bold text-sm text-white">
                            :{chain.listen_port}
                          </span>
                          <span
                            className={`px-1.5 py-0.5 rounded text-[10px] font-semibold border ${
                              chain.status === 'online'
                                ? 'bg-emerald-950/80 border-emerald-800 text-emerald-400'
                                : 'bg-slate-800 border-slate-700 text-slate-400'
                            }`}
                          >
                            {chain.status === 'online' ? 'ONLINE' : 'OFFLINE'}
                          </span>

                          {/* UDP capability badge from verification */}
                          <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-slate-800/80 border border-slate-700 text-amber-300/80">
                            UDP 不可用
                          </span>
                        </div>
                      </div>

                      {/* Enable toggle & Delete */}
                      <div className="flex items-center gap-2">
                        <button
                          title={chain.enabled ? '停用此端口' : '启用此端口'}
                          onClick={() => onUpdateChain(chain.id, { enabled: !chain.enabled })}
                          className={`p-1.5 rounded-lg border transition ${
                            chain.enabled
                              ? 'bg-emerald-950/60 border-emerald-800 text-emerald-400'
                              : 'bg-slate-800 border-slate-700 text-slate-500'
                          }`}
                        >
                          <Power className="w-3.5 h-3.5" />
                        </button>

                        <button
                          title="删除链路"
                          onClick={() => onDeleteChain(chain.id)}
                          className="p-1.5 rounded-lg bg-slate-800 hover:bg-red-950/60 text-slate-400 hover:text-red-400 border border-slate-700 transition"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>

                    {/* Missing node warning alert if any */}
                    {(hasMissingFront || hasMissingLanding) && (
                      <div className="p-2.5 bg-rose-950/40 border border-rose-800/80 rounded-xl flex items-center gap-2 text-rose-300 text-xs">
                        <AlertTriangle className="w-4 h-4 shrink-0 text-rose-400" />
                        <div className="truncate">
                          {hasMissingFront && (
                            <p className="truncate">⚠️ 前置节点已缺失: {chain.frontend_node_id}</p>
                          )}
                          {hasMissingLanding && (
                            <p className="truncate">⚠️ 落地节点已缺失: {chain.landing_node_id}</p>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Path summary */}
                    <div className="text-[11px] text-slate-400 bg-slate-950/50 p-2 rounded-xl border border-slate-800/60 flex items-center justify-between">
                      <div className="truncate">
                        <span className="text-slate-500">路径: </span>
                        <span className="text-sky-300">
                          {chain.frontend_node?.original_name || '直连 DIRECT'}
                        </span>
                        <span className="text-slate-500 mx-1">→</span>
                        <span className="text-indigo-300">
                          {chain.landing_node?.original_name || `[缺失: ${chain.landing_node_id}]`}
                        </span>
                      </div>
                    </div>

                    {/* Auth & Fingerprint browser format */}
                    <div className="grid grid-cols-2 gap-2 text-[11px] font-mono">
                      <div className="bg-slate-950 p-2 rounded-lg border border-slate-800 flex items-center justify-between">
                        <span className="text-slate-500">User:</span>
                        <div className="flex items-center gap-1 truncate">
                          <span className="text-white truncate">{chain.username}</span>
                          <button
                            onClick={() => copyToClipboard(chain.username, `user-${chain.id}`)}
                            className="text-slate-500 hover:text-white"
                          >
                            {copiedKey === `user-${chain.id}` ? (
                              <Check className="w-3 h-3 text-emerald-400" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                      </div>

                      <div className="bg-slate-950 p-2 rounded-lg border border-slate-800 flex items-center justify-between">
                        <span className="text-slate-500">Pass:</span>
                        <div className="flex items-center gap-1 truncate">
                          <span className="text-white truncate">
                            {isRevealed ? chain.password : '••••••••'}
                          </span>
                          <button
                            title="切换显示密码"
                            onClick={() => togglePasswordVisibility(chain.id)}
                            className="text-slate-500 hover:text-white"
                          >
                            {isRevealed ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                          </button>
                          <button
                            onClick={() => copyToClipboard(chain.password, `pass-${chain.id}`)}
                            className="text-slate-500 hover:text-white"
                          >
                            {copiedKey === `pass-${chain.id}` ? (
                              <Check className="w-3 h-3 text-emerald-400" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Quick copy row */}
                    <div className="flex flex-wrap items-center gap-1.5 text-[10px]">
                      <button
                        onClick={() => copyToClipboard(fingerprintFmt, `fmt-${chain.id}`)}
                        className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center gap-1 transition"
                      >
                        {copiedKey === `fmt-${chain.id}` ? (
                          <Check className="w-3 h-3 text-emerald-400" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                        <span>IP:端口:账号:密码</span>
                      </button>

                      <button
                        onClick={() => copyToClipboard(socksUrl, `socks-${chain.id}`)}
                        className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center gap-1 transition"
                      >
                        {copiedKey === `socks-${chain.id}` ? (
                          <Check className="w-3 h-3 text-emerald-400" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                        <span>SOCKS5 URL</span>
                      </button>

                      <button
                        onClick={() => copyToClipboard(httpUrl, `http-${chain.id}`)}
                        className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center gap-1 transition"
                      >
                        {copiedKey === `http-${chain.id}` ? (
                          <Check className="w-3 h-3 text-emerald-400" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                        <span>HTTP URL</span>
                      </button>
                    </div>

                    {/* Egress IP & Actions */}
                    <div className="pt-2 border-t border-slate-800/60 flex items-center justify-between text-xs">
                      <div>
                        {chain.outbound_ip ? (
                          <p className="text-[11px] font-mono text-emerald-400">
                            出口: {chain.outbound_ip} ({chain.outbound_country})
                          </p>
                        ) : (
                          <p className="text-[11px] text-slate-500">未探测出口 IP</p>
                        )}
                      </div>

                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={() => handleEgress(chain.id)}
                          disabled={detectingId === chain.id}
                          className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white rounded-lg text-[11px] flex items-center gap-1 transition disabled:opacity-50"
                        >
                          <RefreshCw
                            className={`w-3 h-3 ${detectingId === chain.id ? 'animate-spin' : ''}`}
                          />
                          <span>{detectingId === chain.id ? '探测中...' : '探测出口'}</span>
                        </button>

                        <button
                          onClick={() => handleOpenDiagnosis(chain)}
                          className="px-2.5 py-1 bg-indigo-600/30 hover:bg-indigo-600 text-indigo-300 hover:text-white rounded-lg text-[11px] border border-indigo-500/40 font-medium flex items-center gap-1 transition"
                        >
                          <Activity className="w-3 h-3" />
                          <span>链路诊断</span>
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>

      {/* Layered Diagnosis Modal */}
      {diagnosingChain && (
        <ChainDiagnosisModal
          isOpen={diagnosisModalOpen}
          onClose={() => setDiagnosisModalOpen(false)}
          chainName={diagnosingChain.name}
          listenPort={diagnosingChain.listen_port}
          loading={diagnosisLoading}
          diagnosis={diagnosisData}
          onReDiagnose={() => handleOpenDiagnosis(diagnosingChain)}
        />
      )}
    </div>
  );
};
