import { Subscription, Node, Chain, ConfigDiff, SpeedtestResult, PoolType } from '../types';
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
