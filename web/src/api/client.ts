import { Subscription, Node, Chain, ConfigDiff, SpeedtestResult, SpeedtestItemResult, SpeedtestJobResponse, ChainDiagnosis, PoolType } from '../types';
import { mockSubscriptions, mockNodes, mockChains, mockDiff } from '../mock/mockData';

const BASE_URL = '/api';

export class ApiClient {
  private static token: string | null = localStorage.getItem('proxybox_token');
  public static useMock: boolean = false;

  public static setToken(t: string | null) {
    this.token = t;
    if (t) {
      localStorage.setItem('proxybox_token', t);
    } else {
      localStorage.removeItem('proxybox_token');
    }
  }

  public static getToken(): string | null {
    return this.token;
  }

  private static async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    if (this.useMock) {
      return this.handleMock<T>(endpoint, options);
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const resp = await fetch(`${BASE_URL}${endpoint}`, {
        ...options,
        headers,
      });

      if (resp.status === 401) {
        this.setToken(null);
        window.dispatchEvent(new Event('auth_required'));
        throw new Error('未授权或登录已过期');
      }

      if (resp.status === 428) {
        window.dispatchEvent(new Event('setup_required'));
        throw new Error('系统尚未初始化管理员密码');
      }

      if (!resp.ok) {
        const errJson = await resp.json().catch(() => ({}));
        throw new Error(errJson.error || `请求失败 HTTP ${resp.status}`);
      }

      return await resp.json();
    } catch (err: any) {
      // If server unreachable, prompt or fallback to mock
      if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
        console.warn('API connection failed, falling back to mock mode');
        this.useMock = true;
        return this.handleMock<T>(endpoint, options);
      }
      throw err;
    }
  }

  private static handleMock<T>(endpoint: string, options: RequestInit): Promise<T> {
    if (endpoint === '/auth/status') {
      return Promise.resolve({ initialized: true } as any);
    }
    if (endpoint === '/subscriptions') {
      return Promise.resolve([...mockSubscriptions] as any);
    }
    if (endpoint === '/pools') {
      return Promise.resolve({
        unassigned: mockNodes.filter(n => n.pool === 'unassigned'),
        frontend: mockNodes.filter(n => n.pool === 'frontend'),
        landing: mockNodes.filter(n => n.pool === 'landing'),
      } as any);
    }
    if (endpoint === '/chains') {
      return Promise.resolve([...mockChains] as any);
    }
    if (endpoint === '/chains/next-port') {
      return Promise.resolve({ port: 9004 } as any);
    }
    if (endpoint === '/apply/diff') {
      return Promise.resolve({ ...mockDiff } as any);
    }
    if (endpoint.includes('/diagnose')) {
      return Promise.resolve({
        chain_id: 'mock_chain',
        frontend_ok: true,
        frontend_delay: 850,
        frontend_status: 'ok',
        landing_ok: true,
        landing_delay: 950,
        landing_status: 'ok',
        chain_ok: true,
        chain_delay: 1100,
        port_ok: true,
        egress_ip: '45.144.136.21',
        egress_country: 'Macao',
        diagnosis_summary: '链路健康可用，出口 IP: 45.144.136.21 (Macao)',
        suggested_action: '链路状态良好，可在指纹浏览器中正常使用',
      } as any);
    }
    if (endpoint === '/speedtest/jobs') {
      return Promise.resolve({ job_id: 'mock_job_1', total: mockNodes.filter(n => n.pool === 'landing').length, status: 'running' } as any);
    }
    if (endpoint.startsWith('/speedtest/jobs/')) {
      const landingNodes = mockNodes.filter(n => n.pool === 'landing');
      return Promise.resolve({
        job_id: 'mock_job_1',
        status: 'finished',
        total: landingNodes.length,
        completed: landingNodes.length,
        last_seq: landingNodes.length,
        results: landingNodes.map((n, idx) => ({
          seq: idx + 1,
          type: 'landing_delay',
          node_id: n.id,
          original_name: n.original_name,
          delay: Math.floor(Math.random() * 300) + 150,
          frontend_delay: 850,
          status: 'SUCCESS',
          status_label: '可链式',
          timestamp: Date.now(),
        })),
      } as any);
    }
    if (endpoint === '/settings') {
      return Promise.resolve({
        ip_whitelist: '',
        speedtest_url: 'https://cp.cloudflare.com/generate_204',
        speedtest_timeout: '5',
        speedtest_concurrency: '10',
      } as any);
    }
    return Promise.resolve({ status: 'ok' } as any);
  }

  // --- Auth ---
  static async getAuthStatus(): Promise<{ initialized: boolean }> {
    return this.request<{ initialized: boolean }>('/auth/status');
  }

  static async setup(password: string): Promise<{ token: string }> {
    const res = await this.request<{ token: string }>('/auth/setup', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
    this.setToken(res.token);
    return res;
  }

  static async login(password: string): Promise<{ token: string }> {
    const res = await this.request<{ token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
    this.setToken(res.token);
    return res;
  }

  // --- Subscriptions ---
  static async getSubscriptions(): Promise<Subscription[]> {
    return this.request<Subscription[]>('/subscriptions');
  }

  static async addSubscription(name: string, url: string, interval: number): Promise<Subscription> {
    return this.request<Subscription>('/subscriptions', {
      method: 'POST',
      body: JSON.stringify({ name, url, update_interval: interval }),
    });
  }

  static async refreshSubscription(id: string): Promise<Subscription> {
    return this.request<Subscription>(`/subscriptions/${id}/refresh`, { method: 'POST' });
  }

  static async deleteSubscription(id: string): Promise<void> {
    return this.request<void>(`/subscriptions/${id}`, { method: 'DELETE' });
  }

  // --- Pools & Nodes ---
  static async getPools(search?: string, subID?: string): Promise<{ unassigned: Node[]; frontend: Node[]; landing: Node[] }> {
    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (subID) params.append('sub_id', subID);
    return this.request<{ unassigned: Node[]; frontend: Node[]; landing: Node[] }>(`/pools?${params.toString()}`);
  }

  static async moveNodes(nodeIDs: string[], targetPool: PoolType): Promise<void> {
    return this.request<void>('/nodes/move', {
      method: 'POST',
      body: JSON.stringify({ node_ids: nodeIDs, target_pool: targetPool }),
    });
  }

  static async reorderPool(pool: PoolType, nodeIDs: string[]): Promise<void> {
    return this.request<void>('/nodes/reorder', {
      method: 'POST',
      body: JSON.stringify({ pool, node_ids: nodeIDs }),
    });
  }

  static async toggleFavorite(nodeID: string, isFavorite: boolean): Promise<void> {
    return this.request<void>(`/nodes/${nodeID}/favorite`, {
      method: 'POST',
      body: JSON.stringify({ is_favorite: isFavorite }),
    });
  }

  // --- Chains ---
  static async getChains(): Promise<Chain[]> {
    return this.request<Chain[]>('/chains');
  }

  static async getNextPort(): Promise<number> {
    const res = await this.request<{ port: number }>('/chains/next-port');
    return res.port;
  }

  static async createChain(data: Partial<Chain>): Promise<Chain> {
    return this.request<Chain>('/chains', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  static async updateChain(id: string, data: Partial<Chain>): Promise<Chain> {
    return this.request<Chain>(`/chains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  static async deleteChain(id: string): Promise<void> {
    return this.request<void>(`/chains/${id}`, { method: 'DELETE' });
  }

  static async detectEgress(id: string): Promise<{ ip: string; country: string }> {
    return this.request<{ ip: string; country: string }>(`/chains/${id}/egress-check`, { method: 'POST' });
  }

  static async diagnoseChain(id: string): Promise<ChainDiagnosis> {
    return this.request<ChainDiagnosis>(`/chains/${id}/diagnose`, { method: 'POST' });
  }

  // --- Speedtest Job & Polling ---
  static async createSpeedtestJob(
    frontID: string,
    landingIDs?: string[],
    url?: string,
    timeout?: number,
    concurrency?: number,
  ): Promise<{ job_id: string; total: number; status: string }> {
    return this.request<{ job_id: string; total: number; status: string }>('/speedtest/jobs', {
      method: 'POST',
      body: JSON.stringify({
        front_id: frontID,
        landing_ids: landingIDs || [],
        url: url || '',
        timeout: timeout || 5,
        concurrency: concurrency || 10,
      }),
    });
  }

  static async getSpeedtestJob(jobID: string, sinceSeq: number = 0): Promise<SpeedtestJobResponse> {
    return this.request<SpeedtestJobResponse>(`/speedtest/jobs/${jobID}?since=${sinceSeq}`);
  }

  static runSpeedtestJob(
    frontID: string,
    landingIDs: string[] | undefined,
    onResult: (res: SpeedtestItemResult) => void,
    onComplete: () => void,
    onError: (err: string) => void,
  ): () => void {
    let active = true;
    let pollTimer: any = null;

    (async () => {
      try {
        const initRes = await this.createSpeedtestJob(frontID, landingIDs);
        onResult({ type: 'connected' });
        let lastSeq = 0;

        const poll = async () => {
          if (!active) return;
          try {
            const data = await this.getSpeedtestJob(initRes.job_id, lastSeq);
            if (!active) return;

            if (data.results && data.results.length > 0) {
              for (const r of data.results) {
                if (r.seq && r.seq > lastSeq) {
                  lastSeq = r.seq;
                }
                onResult(r);
              }
            }

            if (data.status === 'finished' || data.status === 'failed' || data.status === 'cancelled') {
              onComplete();
              return;
            }

            pollTimer = setTimeout(poll, 400);
          } catch (err: any) {
            if (!active) return;
            console.error('Speedtest poll error:', err);
            onError(err.message || '测速轮询失败');
            onComplete();
          }
        };

        pollTimer = setTimeout(poll, 200);
      } catch (err: any) {
        if (!active) return;
        onError(err.message || '启动测速任务失败');
        onComplete();
      }
    })();

    return () => {
      active = false;
      if (pollTimer) clearTimeout(pollTimer);
    };
  }

  // --- Speedtest SSE ---
  static connectSpeedtest(
    frontID: string,
    onResult: (res: SpeedtestResult) => void,
    onError: (err: string) => void,
  ): () => void {
    if (this.useMock) {
      // Mock streaming speedtest
      let idx = 0;
      const landingNodes = mockNodes.filter(n => n.pool === 'landing');
      onResult({ type: 'connected' });
      onResult({ type: 'front_delay', node_id: frontID, delay: 95, timestamp: Date.now() });

      const timer = setInterval(() => {
        if (idx < landingNodes.length) {
          const node = landingNodes[idx];
          const delay = Math.floor(Math.random() * 350) + 120;
          onResult({
            type: 'landing_delay',
            node_id: node.id,
            original_name: node.original_name,
            delay: delay,
            frontend_delay: 95,
            timestamp: Date.now(),
          });
          idx++;
        } else {
          clearInterval(timer);
          onResult({ type: 'done', timestamp: Date.now() });
        }
      }, 400);

      return () => clearInterval(timer);
    }

    const token = this.token ? `&token=${encodeURIComponent(this.token)}` : '';
    const eventSource = new EventSource(`/api/speedtest/stream?front_id=${encodeURIComponent(frontID)}${token}`);

    eventSource.onmessage = (e) => {
      try {
        const data: SpeedtestResult = JSON.parse(e.data);
        onResult(data);
        if (data.type === 'done' || data.type === 'error') {
          eventSource.close();
        }
      } catch (err) {
        console.error('SSE JSON parse error', err);
      }
    };

    eventSource.onerror = (e) => {
      onError('测速流中断或连接超时');
      eventSource.close();
    };

    return () => eventSource.close();
  }

  // --- Apply & Rollback ---
  static async getDiff(): Promise<ConfigDiff> {
    return this.request<ConfigDiff>('/apply/diff');
  }

  static async applyConfig(description?: string): Promise<{ status: string; version: number }> {
    return this.request<{ status: string; version: number }>('/apply', {
      method: 'POST',
      body: JSON.stringify({ description: description || '' }),
    });
  }

  static async getVersions(): Promise<any[]> {
    return this.request<any[]>('/apply/versions');
  }

  static async rollback(version: number): Promise<void> {
    return this.request<void>('/apply/rollback', {
      method: 'POST',
      body: JSON.stringify({ version }),
    });
  }

  // --- Settings & Logs ---
  static async getSettings(): Promise<Record<string, string>> {
    return this.request<Record<string, string>>('/settings');
  }

  static async saveSettings(settings: Record<string, string>): Promise<void> {
    return this.request<void>('/settings', {
      method: 'POST',
      body: JSON.stringify(settings),
    });
  }

  static async getLogs(): Promise<{ logs: string[] }> {
    return this.request<{ logs: string[] }>('/logs');
  }
}
