import React, { useState } from 'react';
import {
  Star,
  GripVertical,
  Search,
  Filter,
  ArrowRightLeft,
  Zap,
  Gauge,
  CheckSquare,
  Square,
  ArrowUp,
  ArrowDown,
  Info,
} from 'lucide-react';
import { Node, PoolType } from '../types';

interface PoolManagerProps {
  unassigned: Node[];
  frontend: Node[];
  landing: Node[];
  selectedFrontendId: string | null;
  onSelectFrontend: (id: string | null) => void;
  onMoveNodes: (nodeIds: string[], targetPool: PoolType) => Promise<void>;
  onReorderPool: (pool: PoolType, nodeIds: string[]) => Promise<void>;
  onToggleFavorite: (nodeId: string, isFav: boolean) => Promise<void>;
  onStartSpeedtest: (frontId: string) => void;
  speedtesting: boolean;
  landingDelays: Record<string, number>;
  frontDelay: number | null;
}

export const PoolManager: React.FC<PoolManagerProps> = ({
  unassigned,
  frontend,
  landing,
  selectedFrontendId,
  onSelectFrontend,
  onMoveNodes,
  onReorderPool,
  onToggleFavorite,
  onStartSpeedtest,
  speedtesting,
  landingDelays,
  frontDelay,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedProtocol, setSelectedProtocol] = useState<string>('all');
  const [selectedRegion, setSelectedRegion] = useState<string>('all');
  const [selectedNodeIds, setSelectedNodeIds] = useState<Set<string>>(new Set());

  // Filter helper
  const filterNodes = (nodes: Node[]) => {
    return nodes.filter((n) => {
      if (searchTerm) {
        const term = searchTerm.toLowerCase();
        const matchesName = n.name.toLowerCase().includes(term);
        const matchesServer = n.server.toLowerCase().includes(term);
        const matchesOriginal = n.original_name.toLowerCase().includes(term);
        if (!matchesName && !matchesServer && !matchesOriginal) return false;
      }
      if (selectedProtocol !== 'all' && n.type.toLowerCase() !== selectedProtocol.toLowerCase()) {
        return false;
      }
      if (selectedRegion !== 'all') {
        const name = (n.name + n.original_name).toLowerCase();
        if (selectedRegion === 'hk' && !name.includes('香港') && !name.includes('hk')) return false;
        if (selectedRegion === 'sg' && !name.includes('新加坡') && !name.includes('sg') && !name.includes('狮城')) return false;
        if (selectedRegion === 'jp' && !name.includes('日本') && !name.includes('jp') && !name.includes('东京')) return false;
        if (selectedRegion === 'us' && !name.includes('美国') && !name.includes('us') && !name.includes('美')) return false;
        if (selectedRegion === 'tw' && !name.includes('台湾') && !name.includes('tw')) return false;
      }
      return true;
    });
  };

  const filteredUnassigned = filterNodes(unassigned);
  const filteredFrontend = filterNodes(frontend);
  const filteredLanding = filterNodes(landing);

  // Selection toggle
  const toggleSelect = (id: string) => {
    const next = new Set(selectedNodeIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedNodeIds(next);
  };

  const selectAllInList = (nodes: Node[]) => {
    const next = new Set(selectedNodeIds);
    nodes.forEach((n) => next.add(n.id));
    setSelectedNodeIds(next);
  };

  const clearSelection = () => {
    setSelectedNodeIds(new Set());
  };

  const handleBatchMove = async (target: PoolType) => {
    if (selectedNodeIds.size === 0) return;
    await onMoveNodes(Array.from(selectedNodeIds), target);
    clearSelection();
  };

  const handleMoveUp = async (pool: PoolType, list: Node[], idx: number) => {
    if (idx <= 0) return;
    const ids = list.map((n) => n.id);
    const temp = ids[idx - 1];
    ids[idx - 1] = ids[idx];
    ids[idx] = temp;
    await onReorderPool(pool, ids);
  };

  const handleMoveDown = async (pool: PoolType, list: Node[], idx: number) => {
    if (idx >= list.length - 1) return;
    const ids = list.map((n) => n.id);
    const temp = ids[idx + 1];
    ids[idx + 1] = ids[idx];
    ids[idx] = temp;
    await onReorderPool(pool, ids);
  };

  const getDelayBadge = (delay?: number) => {
    if (delay === undefined || delay === 0) {
      return <span className="text-[11px] text-slate-500 font-mono">未测</span>;
    }
    if (delay < 0) {
      return <span className="text-[11px] text-slate-500 font-mono bg-slate-800 px-1.5 py-0.5 rounded">超时</span>;
    }
    let colorClass = 'text-emerald-400 bg-emerald-950/80 border-emerald-800';
    if (delay > 800) {
      colorClass = 'text-red-400 bg-red-950/80 border-red-800';
    } else if (delay > 300) {
      colorClass = 'text-amber-400 bg-amber-950/80 border-amber-800';
    }
    return (
      <span className={`text-[11px] font-mono px-1.5 py-0.5 rounded border ${colorClass}`}>
        {delay} ms
      </span>
    );
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 shadow-xl space-y-4">
      {/* Search & Filters */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2 flex-1 min-w-[260px]">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="搜索节点名称、服务器地址、端口..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-9 pr-4 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-sky-500"
            />
          </div>
        </div>

        {/* Region & Protocol Tags */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Protocol filter */}
          <select
            value={selectedProtocol}
            onChange={(e) => setSelectedProtocol(e.target.value)}
            className="bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-300 focus:outline-none focus:border-sky-500"
          >
            <option value="all">全部协议</option>
            <option value="hysteria2">Hysteria 2</option>
            <option value="tuic">TUIC</option>
            <option value="vless">VLESS</option>
            <option value="trojan">Trojan</option>
            <option value="ss">Shadowsocks</option>
          </select>

          {/* Region filter */}
          <select
            value={selectedRegion}
            onChange={(e) => setSelectedRegion(e.target.value)}
            className="bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-300 focus:outline-none focus:border-sky-500"
          >
            <option value="all">全部地区</option>
            <option value="hk">🇭🇰 香港</option>
            <option value="sg">🇸🇬 新加坡</option>
            <option value="jp">🇯🇵 日本</option>
            <option value="us">🇺🇸 美国</option>
            <option value="tw">🇹🇼 台湾</option>
          </select>

          {/* Batch Actions */}
          {selectedNodeIds.size > 0 && (
            <div className="flex items-center gap-1.5 bg-slate-800 p-1 rounded-xl border border-slate-700 animate-fadeIn">
              <span className="text-xs text-sky-400 px-2 font-medium">已选 {selectedNodeIds.size}</span>
              <button
                onClick={() => handleBatchMove('frontend')}
                className="px-2.5 py-1 text-xs bg-sky-600 hover:bg-sky-500 text-white rounded-lg transition"
              >
                至前置池
              </button>
              <button
                onClick={() => handleBatchMove('landing')}
                className="px-2.5 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition"
              >
                至住宅池
              </button>
              <button
                onClick={() => handleBatchMove('unassigned')}
                className="px-2.5 py-1 text-xs bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg transition"
              >
                移出
              </button>
              <button
                onClick={clearSelection}
                className="px-2 py-1 text-xs text-slate-400 hover:text-white transition"
              >
                取消
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Three Pools Columns */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* 1. 未归类池 */}
        <div className="bg-slate-950/80 border border-slate-800/80 rounded-2xl flex flex-col h-[480px]">
          <div className="p-3.5 border-b border-slate-800/80 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-slate-500"></span>
              <h3 className="font-semibold text-sm text-slate-200">未归类节点</h3>
              <span className="text-xs text-slate-500 font-mono">({filteredUnassigned.length})</span>
            </div>
            <button
              onClick={() => selectAllInList(filteredUnassigned)}
              className="text-xs text-slate-400 hover:text-sky-400"
            >
              全选
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-2.5 space-y-2">
            {filteredUnassigned.length === 0 ? (
              <div className="text-center py-12 text-slate-600 text-xs">暂无未归类节点</div>
            ) : (
              filteredUnassigned.map((node) => {
                const isSelected = selectedNodeIds.has(node.id);
                return (
                  <div
                    key={node.id}
                    onClick={() => toggleSelect(node.id)}
                    className={`p-2.5 rounded-xl border text-xs transition cursor-pointer flex items-center justify-between gap-2 ${
                      isSelected
                        ? 'bg-sky-950/40 border-sky-600 text-sky-200'
                        : 'bg-slate-900/80 border-slate-800 hover:border-slate-700 text-slate-300'
                    }`}
                  >
                    <div className="flex items-center gap-2 truncate">
                      <div className="text-slate-400">
                        {isSelected ? <CheckSquare className="w-4 h-4 text-sky-400" /> : <Square className="w-4 h-4" />}
                      </div>
                      <div className="truncate">
                        <p className="font-medium truncate text-white">{node.original_name}</p>
                        <p className="text-[11px] text-slate-500 truncate font-mono">{node.sub_name} · {node.type}</p>
                      </div>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleFavorite(node.id, !node.is_favorite);
                      }}
                      className={`p-1 rounded hover:bg-slate-800 ${
                        node.is_favorite ? 'text-amber-400' : 'text-slate-600'
                      }`}
                    >
                      <Star className="w-3.5 h-3.5 fill-current" />
                    </button>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* 2. 前置池 (一对多复用) */}
        <div className="bg-slate-950/80 border border-sky-900/40 rounded-2xl flex flex-col h-[480px]">
          <div className="p-3.5 border-b border-slate-800/80 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-sky-400 animate-pulse"></span>
              <h3 className="font-semibold text-sm text-sky-300">前置池 (复用节点)</h3>
              <span className="text-xs text-sky-500 font-mono">({filteredFrontend.length})</span>
            </div>
            <button
              onClick={() => selectAllInList(filteredFrontend)}
              className="text-xs text-slate-400 hover:text-sky-400"
            >
              全选
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-2.5 space-y-2">
            {filteredFrontend.length === 0 ? (
              <div className="text-center py-12 text-slate-600 text-xs">
                从左侧或订阅中将机场节点勾选并移入前置池
              </div>
            ) : (
              filteredFrontend.map((node, idx) => {
                const isSelected = selectedNodeIds.has(node.id);
                const isActiveFront = selectedFrontendId === node.id;

                return (
                  <div
                    key={node.id}
                    className={`p-2.5 rounded-xl border text-xs transition cursor-pointer space-y-1.5 ${
                      isActiveFront
                        ? 'bg-sky-950/60 border-sky-400 ring-2 ring-sky-500/30'
                        : isSelected
                        ? 'bg-slate-800 border-sky-600'
                        : 'bg-slate-900/80 border-slate-800 hover:border-slate-700'
                    }`}
                    onClick={() => {
                      // Click to select this frontend node for chain latency preview
                      onSelectFrontend(isActiveFront ? null : node.id);
                    }}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 truncate">
                        <div
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleSelect(node.id);
                          }}
                          className="text-slate-400 hover:text-white"
                        >
                          {isSelected ? <CheckSquare className="w-4 h-4 text-sky-400" /> : <Square className="w-4 h-4" />}
                        </div>
                        <div className="truncate">
                          <p className="font-medium truncate text-white">{node.original_name}</p>
                          <p className="text-[11px] text-slate-400 font-mono">
                            {node.sub_name} · <span className="uppercase text-sky-400">{node.type}</span>
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-1">
                        {/* Delay badge */}
                        {getDelayBadge(isActiveFront && frontDelay !== null ? frontDelay : node.last_delay)}

                        {/* Star */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onToggleFavorite(node.id, !node.is_favorite);
                          }}
                          className={`p-1 rounded hover:bg-slate-800 ${
                            node.is_favorite ? 'text-amber-400' : 'text-slate-600'
                          }`}
                        >
                          <Star className="w-3.5 h-3.5 fill-current" />
                        </button>
                      </div>
                    </div>

                    {/* Order handles & selection indicator */}
                    <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1 border-t border-slate-900">
                      <span className={isActiveFront ? 'text-sky-400 font-medium' : 'text-slate-500'}>
                        {isActiveFront ? '★ 当前选中前置 (可点击连线)' : '点击选中为当前测速前置'}
                      </span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleMoveUp('frontend', filteredFrontend, idx);
                          }}
                          className="hover:text-slate-200 p-0.5"
                          title="上移"
                        >
                          <ArrowUp className="w-3 h-3" />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleMoveDown('frontend', filteredFrontend, idx);
                          }}
                          className="hover:text-slate-200 p-0.5"
                          title="下移"
                        >
                          <ArrowDown className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* 3. 住宅落地池 (链式延迟显示) */}
        <div className="bg-slate-950/80 border border-indigo-900/40 rounded-2xl flex flex-col h-[480px]">
          <div className="p-3.5 border-b border-slate-800/80 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-indigo-400"></span>
              <h3 className="font-semibold text-sm text-indigo-300">住宅落地池</h3>
              <span className="text-xs text-indigo-500 font-mono">({filteredLanding.length})</span>
            </div>

            <div className="flex items-center gap-2">
              {selectedFrontendId && (
                <button
                  onClick={() => onStartSpeedtest(selectedFrontendId)}
                  disabled={speedtesting}
                  className="flex items-center gap-1.5 px-2.5 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-lg transition"
                >
                  <Gauge className={`w-3.5 h-3.5 ${speedtesting ? 'animate-spin' : ''}`} />
                  <span>{speedtesting ? '测速中...' : '链式测速'}</span>
                </button>
              )}
              <button
                onClick={() => selectAllInList(filteredLanding)}
                className="text-xs text-slate-400 hover:text-sky-400"
              >
                全选
              </button>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-2.5 space-y-2">
            {filteredLanding.length === 0 ? (
              <div className="text-center py-12 text-slate-600 text-xs">
                从左侧或订阅中将住宅代理节点勾选并移入住宅落地池
              </div>
            ) : (
              filteredLanding.map((node, idx) => {
                const isSelected = selectedNodeIds.has(node.id);
                // Latency via selected frontend or default
                const delay = landingDelays[node.id] !== undefined ? landingDelays[node.id] : node.last_delay;

                return (
                  <div
                    key={node.id}
                    onClick={() => toggleSelect(node.id)}
                    className={`p-2.5 rounded-xl border text-xs transition cursor-pointer space-y-1.5 ${
                      isSelected
                        ? 'bg-indigo-950/40 border-indigo-500 text-indigo-200'
                        : 'bg-slate-900/80 border-slate-800 hover:border-slate-700 text-slate-300'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 truncate">
                        <div className="text-slate-400">
                          {isSelected ? <CheckSquare className="w-4 h-4 text-indigo-400" /> : <Square className="w-4 h-4" />}
                        </div>
                        <div className="truncate">
                          <p className="font-medium truncate text-white">{node.original_name}</p>
                          <p className="text-[11px] text-slate-400 font-mono">
                            {node.sub_name} · <span className="uppercase text-indigo-400">{node.type}</span>
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5">
                        {/* Delay */}
                        {getDelayBadge(delay)}

                        {/* Star */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onToggleFavorite(node.id, !node.is_favorite);
                          }}
                          className={`p-1 rounded hover:bg-slate-800 ${
                            node.is_favorite ? 'text-amber-400' : 'text-slate-600'
                          }`}
                        >
                          <Star className="w-3.5 h-3.5 fill-current" />
                        </button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1 border-t border-slate-900">
                      <span>{selectedFrontendId ? '已计算链式延迟' : '选择前置可测链式延迟'}</span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleMoveUp('landing', filteredLanding, idx);
                          }}
                          className="hover:text-slate-200 p-0.5"
                          title="上移"
                        >
                          <ArrowUp className="w-3 h-3" />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleMoveDown('landing', filteredLanding, idx);
                          }}
                          className="hover:text-slate-200 p-0.5"
                          title="下移"
                        >
                          <ArrowDown className="w-3 h-3" />
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
    </div>
  );
};
