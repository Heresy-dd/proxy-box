import React, { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { UnassignedPool } from './components/UnassignedPool';
import { TriColumnCanvas } from './components/TriColumnCanvas';
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
  const [landingStatusLabels, setLandingStatusLabels] = useState<Record<string, string>>({});
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
      // Server error or unreachable, fallback to mock preview
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

  // Speedtest using Job runner with polling & SSE fallback
  const handleStartSpeedtest = (frontId: string) => {
    setSpeedtesting(true);
    setLandingDelays({});
    setLandingStatusLabels({});
    setFrontDelay(null);

    const cancelFn = ApiClient.runSpeedtestJob(
      frontId,
      undefined,
      (res) => {
        if (res.type === 'front_delay' && res.delay !== undefined) {
          setFrontDelay(res.delay);
        } else if (res.type === 'landing_delay' && res.node_id && res.delay !== undefined) {
          setLandingDelays((prev) => ({
            ...prev,
            [res.node_id!]: res.delay!,
          }));
          if (res.status_label) {
            setLandingStatusLabels((prev) => ({
              ...prev,
              [res.node_id!]: res.status_label!,
            }));
          }
        }
      },
      () => {
        setSpeedtesting(false);
      },
      (err) => {
        console.warn('Speedtest error:', err);
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
    <div className="min-h-screen flex flex-col bg-slate-950 text-slate-100 pb-16 font-sans">
      {/* Top Sticky Navbar */}
      <Navbar
        diff={diff}
        subscriptions={subscriptions}
        onOpenSubscriptions={() => setSubModalOpen(true)}
        onOpenSettings={() => setSettingsModalOpen(true)}
        onOpenApply={() => setApplyModalOpen(true)}
        onLogout={handleLogout}
        isMock={isMock}
        onToggleMock={handleToggleMock}
      />

      {/* Main Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-6 space-y-6">
        {/* Top: Collapsible Unassigned Pool */}
        <section>
          <UnassignedPool
            nodes={unassignedNodes}
            onMoveNodes={handleMoveNodes}
          />
        </section>

        {/* Center: 3 Parallel Columns on the Same Horizon */}
        <section>
          <TriColumnCanvas
            chains={chains}
            frontendNodes={frontendNodes}
            landingNodes={landingNodes}
            selectedFrontendId={selectedFrontendId}
            onSelectFrontend={setSelectedFrontendId}
            onAddChain={handleAddChain}
            onUpdateChain={handleUpdateChain}
            onDeleteChain={handleDeleteChain}
            onDetectEgress={handleDetectEgress}
            onToggleFavorite={handleToggleFavorite}
            onStartSpeedtest={handleStartSpeedtest}
            speedtesting={speedtesting}
            landingDelays={landingDelays}
            landingStatusLabels={landingStatusLabels}
            frontDelay={frontDelay}
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
