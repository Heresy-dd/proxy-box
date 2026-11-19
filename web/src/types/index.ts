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
  pending_confirmation?: boolean;
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
  frontend_node_status?: 'ok' | 'missing' | 'invalid_pool' | 'direct';
  landing_node_status?: 'ok' | 'missing' | 'invalid_pool';
}

export interface ChainDiagnosis {
  chain_id: string;
  frontend_ok: boolean;
  frontend_delay: number;
  frontend_status: string;
  landing_ok: boolean;
  landing_delay: number;
  landing_status: string;
  chain_ok: boolean;
  chain_delay: number;
  port_ok: boolean;
  port_error?: string;
  egress_ip?: string;
  egress_country?: string;
  egress_error?: string;
  diagnosis_summary: string;
  suggested_action: string;
}

export interface ConfigDiff {
  total_chains: number;
  enabled_chains: number;
  active_ports: number[];
  added_ports: number[];
  removed_ports: number[];
  changes: string[];
}

export interface SpeedtestItemResult {
  seq?: number;
  type: 'connected' | 'front_delay' | 'landing_delay' | 'done' | 'error';
  node_id?: string;
  original_name?: string;
  delay?: number;
  frontend_delay?: number;
  status?: 'SUCCESS' | 'TIMEOUT' | 'HANDSHAKE_FAILED' | 'REFUSED' | 'TARGET_UNREACHABLE' | 'UNVERIFIED_COMBO' | 'ERROR';
  status_label?: string;
  timestamp?: number;
  message?: string;
}

export type SpeedtestResult = SpeedtestItemResult;

export interface SpeedtestJobResponse {
  job_id: string;
  status: 'running' | 'finished' | 'cancelled' | 'failed';
  total: number;
  completed: number;
  last_seq: number;
  results: SpeedtestItemResult[];
  error?: string;
}

export interface MigrationReport {
  total_parsed: number;
  inherited: number;
  added: number;
  deprecated: number;
  conflicts: number;
  details?: string[];
}
