import React, { useState, useMemo } from 'react';
import {
  ChevronDown,
  ChevronUp,
  Search,
  CheckSquare,
  Square,
  ArrowRight,
  Layers,
  Sparkles,
} from 'lucide-react';
import { Node, PoolType } from '../types';

interface UnassignedPoolProps {
  nodes: Node[];
  onMoveNodes: (nodeIds: string[], targetPool: PoolType) => Promise<void>;
}

export const UnassignedPool: React.FC<UnassignedPoolProps> = ({ nodes, onMoveNodes }) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [moving, setMoving] = useState(false);

  const filteredNodes = useMemo(() => {
    if (!searchTerm) return nodes;
    const lower = searchTerm.toLowerCase();
    return nodes.filter(
      (n) =>
        n.original_name.toLowerCase().includes(lower) ||
        n.type.toLowerCase().includes(lower) ||
        (n.sub_name && n.sub_name.toLowerCase().includes(lower))
    );
  }, [nodes, searchTerm]);

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const selectAll = () => {
    setSelectedIds(new Set(filteredNodes.map((n) => n.id)));
  };

  const clearSelection = () => {
    setSelectedIds(new Set());
  };

  const handleBatchMove = async (targetPool: PoolType) => {
    if (selectedIds.size === 0) return;
    setMoving(true);
    try {
      await onMoveNodes(Array.from(selectedIds), targetPool);
      setSelectedIds(new Set());
    } finally {
      setMoving(false);
    }
  };

  const handleSingleMove = async (id: string, targetPool: PoolType) => {
    await onMoveNodes([id], targetPool);
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden shadow-lg transition-all duration-200">
      {/* Collapsed / Summary Bar */}
      <div className="p-3.5 flex flex-wrap items-center justify-between gap-3 bg-slate-950/40">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-xl bg-slate-800/80 border border-slate-700/60 flex items-center justify-center text-slate-300">
            <Layers className="w-4 h-4" />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-bold text-white uppercase tracking-wider">
              未归类节点
            </span>
            <span
              className={`text-[11px] font-mono px-2 py-0.5 rounded-full border ${
                nodes.length > 0
                  ? 'bg-amber-950/60 border-amber-800/80 text-amber-300 font-semibold'
                  : 'bg-slate-800 border-slate-700 text-slate-400'
              }`}
            >
              {nodes.length}
            </span>
          </div>
          <span className="text-[11px] text-slate-400 hidden sm:inline">
            新订阅或未分配角色的节点汇聚在此，可批量归入「前置中继池」或「住宅落地池」
          </span>
        </div>

        <div className="flex items-center gap-2">
          {nodes.length > 0 && !isExpanded && (
            <div className="flex items-center gap-1.5 text-xs text-slate-400">
              <span className="text-[11px] bg-slate-800/80 px-2.5 py-1 rounded-lg border border-slate-700">
                {nodes.slice(0, 3).map((n) => n.original_name).join(' · ')}
                {nodes.length > 3 ? ` 等共 ${nodes.length} 个` : ''}
              </span>
            </div>
          )}

          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="flex items-center gap-1 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-xl text-xs font-medium border border-slate-700 transition"
          >
            <span>{isExpanded ? '收起面板' : '展开管理'}</span>
            {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
          </button>
        </div>
      </div>

      {/* Expanded Area */}
      {isExpanded && (
        <div className="p-4 border-t border-slate-800 space-y-3.5 bg-slate-900/40">
          {/* Controls Bar */}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="搜索未归类节点 / 协议..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="bg-slate-950 border border-slate-800 rounded-xl pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-sky-500 w-48 sm:w-64"
                />
              </div>

              {filteredNodes.length > 0 && (
                <div className="flex items-center gap-1.5 text-xs">
                  <button
                    onClick={selectAll}
                    className="px-2.5 py-1 rounded-lg bg-slate-800 text-slate-300 hover:text-white transition flex items-center gap-1"
                  >
                    <CheckSquare className="w-3 h-3" /> 全选 ({filteredNodes.length})
                  </button>
                  {selectedIds.size > 0 && (
                    <button
                      onClick={clearSelection}
                      className="px-2.5 py-1 rounded-lg bg-slate-800 text-slate-400 hover:text-white transition flex items-center gap-1"
                    >
                      <Square className="w-3 h-3" /> 清空
                    </button>
                  )}
                </div>
              )}
            </div>

            {/* Batch Action Buttons */}
            {selectedIds.size > 0 && (
              <div className="flex items-center gap-2">
                <span className="text-[11px] text-slate-400 font-medium">
                  已选 <span className="text-white font-bold">{selectedIds.size}</span> 个节点:
                </span>
                <button
                  onClick={() => handleBatchMove('frontend')}
                  disabled={moving}
                  className="flex items-center gap-1 px-3 py-1 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-xs font-medium transition shadow-md shadow-sky-600/20 disabled:opacity-50"
                >
                  <span>移入前置池</span>
                </button>
                <button
                  onClick={() => handleBatchMove('landing')}
                  disabled={moving}
                  className="flex items-center gap-1 px-3 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-medium transition shadow-md shadow-indigo-600/20 disabled:opacity-50"
                >
                  <span>移入住宅池</span>
                </button>
              </div>
            )}
          </div>

          {/* Node Tag Grid */}
          {filteredNodes.length === 0 ? (
            <div className="py-8 text-center text-slate-500 text-xs">
              {searchTerm ? '无匹配节点' : '未归类节点池为空，所有节点已分配完毕'}
            </div>
          ) : (
            <div className="flex flex-wrap gap-2 max-h-60 overflow-y-auto pr-1">
              {filteredNodes.map((node) => {
                const isSelected = selectedIds.has(node.id);

                return (
                  <div
                    key={node.id}
                    className={`group flex items-center gap-2 pl-3 pr-2 py-1.5 rounded-xl border text-xs transition cursor-pointer ${
                      isSelected
                        ? 'bg-sky-950/70 border-sky-500/80 text-white ring-1 ring-sky-500/30'
                        : 'bg-slate-950/80 border-slate-800 text-slate-300 hover:border-slate-700 hover:text-white'
                    }`}
                    onClick={() => toggleSelect(node.id)}
                  >
                    <span className="font-medium truncate max-w-[200px]">{node.original_name}</span>
                    <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700/60">
                      {node.type}
                    </span>

                    {/* Quick Move Mini Buttons on Hover */}
                    <div
                      className="flex items-center gap-1 ml-1 opacity-60 group-hover:opacity-100 transition"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <button
                        title="移至前置池"
                        onClick={() => handleSingleMove(node.id, 'frontend')}
                        className="px-1.5 py-0.5 rounded bg-sky-950/80 text-sky-400 border border-sky-800 hover:bg-sky-900 text-[10px]"
                      >
                        前置
                      </button>
                      <button
                        title="移至住宅池"
                        onClick={() => handleSingleMove(node.id, 'landing')}
                        className="px-1.5 py-0.5 rounded bg-indigo-950/80 text-indigo-400 border border-indigo-800 hover:bg-indigo-900 text-[10px]"
                      >
                        住宅
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
