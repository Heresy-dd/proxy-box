export type PoolType = 'unassigned' | 'frontend' | 'landing';

export interface Subscription {
  id: string;
  name: string;
  url: string;
  update_interval: number;
  upload_bytes: number;
  download_bytes: number;
  total_bytes: number;
  expire_timestamp: number;
  last_updated_at: string;
  status: 'active' | 'error';
  error_message: string;
  node_count?: number;
}

export interface Node {
  id: string;
  subscription_id: string;
  sub_name?: string;
  original_name: string;
  name: string;
  type: string;
  server: string;
  port: number;
  raw_config: Record<string, any>;
  pool: PoolType;
  order: number;
  is_favorite: boolean;
  last_delay: number;
  last_checked_at?: string;
  active: boolean;
}

export interface Chain {
  id: string;
  name: string;
  frontend_node_id?: string | null;
  landing_node_id: string;
  listen_port: number;
  username: string;
  password: string;
  enabled: boolean;
  status: 'online' | 'offline' | 'disabled';
  last_delay: number;
  outbound_ip?: string;
  outbound_country?: string;
  order: number;
  created_at: string;
  updated_at: string;

  frontend_node?: Node;
  landing_node?: Node;
}

export interface ConfigDiff {
  total_chains: number;
  enabled_chains: number;
  active_ports: number[];
  added_ports: number[];
  removed_ports: number[];
  changes: string[];
}

export interface SpeedtestResult {
  type: 'connected' | 'front_delay' | 'landing_delay' | 'done' | 'error';
  node_id?: string;
  original_name?: string;
  delay?: number;
  frontend_delay?: number;
  timestamp?: number;
  message?: string;
}
