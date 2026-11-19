import React, { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { PoolManager } from './components/PoolManager';
import { CanvasChain } from './components/CanvasChain';
import { SubscriptionModal } from './components/SubscriptionModal';
import { ApplyDiffModal } from './components/ApplyDiffModal';
import { SettingsModal } from './components/SettingsModal';
import { SetupWizard } from './components/SetupWizard';
import { ApiClient } from './api/client';
import { Node, Chain, Subscription, ConfigDiff, PoolType } from './types';

export const App: React.FC = () => {
  // App States
  const [authNeeded, setAuthNeeded] = useState(false);
  const [isSetup, setIsSetup] = useState(false);
  const [isMock, setIsMock] = useState(false);

  // Modals
  const [subModalOpen, setSubModalOpen] = useState(false);
  const [settingsModalOpen, setSettingsModalOpen] = useState(false);
  const [applyModalOpen, setApplyModalOpen] = useState(false);

  // Core Data
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [unassignedNodes, setUnassignedNodes] = useState<Node[]>([]);
  const [frontendNodes, setFrontendNodes] = useState<Node[]>([]);
  const [landingNodes, setLandingNodes] = useState<Node[]>([]);
  const [chains, setChains] = useState<Chain[]>([]);
  const [diff, setDiff] = useState<ConfigDiff | null>(null);

  // Selection & Speedtest
  const [selectedFrontendId, setSelectedFrontendId] = useState<string | null>(null);
  const [speedtesting, setSpeedtesting] = useState(false);
  const [landingDelays, setLandingDelays] = useState<Record<string, number>>({});
  const [frontDelay, setFrontDelay] = useState<number | null>(null);

  // Initial Load
  useEffect(() => {
    initAuthAndData();

    const handleAuthReq = () => {
      setAuthNeeded(true);
      setIsSetup(false);
    };
    const handleSetupReq = () => {
      setAuthNeeded(true);
      setIsSetup(true);
    };

    window.addEventListener('auth_required', handleAuthReq);
    window.addEventListener('setup_required', handleSetupReq);

    return () => {
      window.removeEventListener('auth_required', handleAuthReq);
      window.removeEventListener('setup_required', handleSetupReq);
    };
  }, []);

  const initAuthAndData = async () => {
    try {
      const status = await ApiClient.getAuthStatus();
      if (!status.initialized) {
        setIsSetup(true);
        setAuthNeeded(true);
        return;
      }
      if (!ApiClient.getToken()) {
        setIsSetup(false);
        setAuthNeeded(true);
        return;
      }
      await refreshAllData();
    } catch {
      // Server error or unreachable, offer mock
      setIsMock(true);
      ApiClient.useMock = true;
      await refreshAllData();
    }
  };

  const refreshAllData = async () => {
    try {
      const [subs, pools, chainList, diffData] = await Promise.all([
        ApiClient.getSubscriptions(),
        ApiClient.getPools(),
        ApiClient.getChains(),
        ApiClient.getDiff(),
      ]);

      setSubscriptions(subs);
      setUnassignedNodes(pools.unassigned || []);
      setFrontendNodes(pools.frontend || []);
      setLandingNodes(pools.landing || []);
      setChains(chainList || []);
      setDiff(diffData || null);
    } catch (err) {
      console.error('refresh error', err);
    }
  };

  // Auth Handlers
  const handleAuthSubmit = async (pwd: string) => {
    if (isSetup) {
      await ApiClient.setup(pwd);
    } else {
      await ApiClient.login(pwd);
    }
    setAuthNeeded(false);
    await refreshAllData();
  };

  const handleLogout = () => {
    ApiClient.setToken(null);
    setAuthNeeded(true);
    setIsSetup(false);
  };

  const handleToggleMock = () => {
    const nextMock = !isMock;
    setIsMock(nextMock);
    ApiClient.useMock = nextMock;
    refreshAllData();
  };

  // Subscription Actions
  const handleAddSubscription = async (name: string, url: string, interval: number) => {
    await ApiClient.addSubscription(name, url, interval);
    await refreshAllData();
  };

  const handleRefreshSubscription = async (id: string) => {
    await ApiClient.refreshSubscription(id);
    await refreshAllData();
  };

  const handleDeleteSubscription = async (id: string) => {
    if (!confirm('确定删除此订阅吗？入库节点将被清理。')) return;
    await ApiClient.deleteSubscription(id);
    await refreshAllData();
  };

  // Pool Actions
  const handleMoveNodes = async (nodeIds: string[], targetPool: PoolType) => {
    await ApiClient.moveNodes(nodeIds, targetPool);
    await refreshAllData();
  };

  const handleReorderPool = async (pool: PoolType, nodeIds: string[]) => {
    await ApiClient.reorderPool(pool, nodeIds);
    await refreshAllData();
  };

  const handleToggleFavorite = async (nodeId: string, isFav: boolean) => {
    await ApiClient.toggleFavorite(nodeId, isFav);
    await refreshAllData();
  };

  // Speedtest
  const handleStartSpeedtest = (frontId: string) => {
    setSpeedtesting(true);
    setLandingDelays({});
    setFrontDelay(null);

    const closeStream = ApiClient.connectSpeedtest(
      frontId,
      (res) => {
        if (res.type === 'front_delay' && res.delay !== undefined) {
          setFrontDelay(res.delay);
        } else if (res.type === 'landing_delay' && res.node_id && res.delay !== undefined) {
          setLandingDelays((prev) => ({
            ...prev,
            [res.node_id!]: res.delay!,
          }));
        } else if (res.type === 'done' || res.type === 'error') {
          setSpeedtesting(false);
        }
      },
      (err) => {
        alert(err);
        setSpeedtesting(false);
      }
    );
  };

  // Chain Actions
  const handleAddChain = async (landingNodeId: string, frontendNodeId?: string) => {
    await ApiClient.createChain({
      landing_node_id: landingNodeId,
      frontend_node_id: frontendNodeId || null,
      enabled: true,
    });
    await refreshAllData();
  };

  const handleUpdateChain = async (id: string, data: Partial<Chain>) => {
    await ApiClient.updateChain(id, data);
    await refreshAllData();
  };

  const handleDeleteChain = async (id: string) => {
    if (!confirm('确定删除此链路映射吗？')) return;
    await ApiClient.deleteChain(id);
    await refreshAllData();
  };

  const handleDetectEgress = async (id: string) => {
    const res = await ApiClient.detectEgress(id);
    await refreshAllData();
    return res;
  };

  // Apply Config
  const handleApplyConfig = async (desc: string) => {
    await ApiClient.applyConfig(desc);
    await refreshAllData();
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-950 text-slate-100 pb-16">
      {/* Top Navbar */}
      <Navbar
        diff={diff}
        onOpenSubscriptions={() => setSubModalOpen(true)}
        onOpenSettings={() => setSettingsModalOpen(true)}
        onOpenApply={() => setApplyModalOpen(true)}
        onLogout={handleLogout}
        isMock={isMock}
        onToggleMock={handleToggleMock}
      />

      {/* Main Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-6 py-6 space-y-6">
        {/* Top: 3-Pool Manager Area */}
        <section>
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-400">
              节点归类与排序（未归类 · 前置池 · 住宅池）
            </h2>
            <span className="text-xs text-slate-500">
              支持多选框选/Ctrl批量移动 · 池内持久化拖拽排序 · 选中前置实时链式测速
            </span>
          </div>

          <PoolManager
            unassigned={unassignedNodes}
            frontend={frontendNodes}
            landing={landingNodes}
            selectedFrontendId={selectedFrontendId}
            onSelectFrontend={setSelectedFrontendId}
            onMoveNodes={handleMoveNodes}
            onReorderPool={handleReorderPool}
            onToggleFavorite={handleToggleFavorite}
            onStartSpeedtest={handleStartSpeedtest}
            speedtesting={speedtesting}
            landingDelays={landingDelays}
            frontDelay={frontDelay}
          />
        </section>

        {/* Bottom: Visual Canvas Area */}
        <section>
          <CanvasChain
            chains={chains}
            frontendNodes={frontendNodes}
            landingNodes={landingNodes}
            onAddChain={handleAddChain}
            onUpdateChain={handleUpdateChain}
            onDeleteChain={handleDeleteChain}
            onDetectEgress={handleDetectEgress}
            selectedFrontendId={selectedFrontendId}
            onSelectFrontend={setSelectedFrontendId}
          />
        </section>
      </main>

      {/* Modals */}
      <SubscriptionModal
        isOpen={subModalOpen}
        onClose={() => setSubModalOpen(false)}
        subscriptions={subscriptions}
        onAdd={handleAddSubscription}
        onRefresh={handleRefreshSubscription}
        onDelete={handleDeleteSubscription}
      />

      <ApplyDiffModal
        isOpen={applyModalOpen}
        onClose={() => setApplyModalOpen(false)}
        diff={diff}
        onApply={handleApplyConfig}
        onRefreshDiff={refreshAllData}
      />

      <SettingsModal
        isOpen={settingsModalOpen}
        onClose={() => setSettingsModalOpen(false)}
      />

      {authNeeded && (
        <SetupWizard
          isSetup={isSetup}
          onSubmit={handleAuthSubmit}
          onContinueAsMock={() => {
            setAuthNeeded(false);
            setIsMock(true);
            ApiClient.useMock = true;
            refreshAllData();
          }}
        />
      )}
    </div>
  );
};

export default App;
