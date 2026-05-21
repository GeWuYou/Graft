import type { MonitorTrendRange } from '../contract/trend';

export interface ServerStatusDependency {
  status: string;
  detail: string;
  latency_ms: number | null;
}

export interface ServerStatusPlugin {
  name: string;
  status: string;
  status_detail: string;
  version: string;
  depends_on: string[];
  missing_dependencies?: string[];
}

export interface ServerStatusServer {
  version: string;
  started_at: string;
  uptime_seconds: number;
  go_version: string;
  app_name: string;
  app_env: string;
}

export interface ServerStatusRuntime {
  go_version: string;
  host_name: string;
  operating_system: string;
  architecture: string;
  cpu_cores: number;
  load_average: {
    one_minute: number;
    five_minutes: number;
    fifteen_minutes: number;
  };
  disk_usage: {
    path: string;
    total_bytes: number;
    used_bytes: number;
    free_bytes: number;
    used_percent: number;
  };
  host_memory_total_bytes: number;
  host_memory_used_bytes: number;
  host_memory_free_bytes: number;
  host_memory_used_percent: number;
  goroutines: number;
  runtime_alloc_bytes: number;
  runtime_heap_in_use_bytes: number;
  runtime_sys_bytes: number;
  runtime_gc_cycles: number;
}

export interface ServerStatusDependencies {
  database: ServerStatusDependency;
  redis: ServerStatusDependency;
}

export interface ServerStatusSummary {
  total_dependencies: number;
  healthy_dependencies: number;
  degraded_dependencies: number;
  unknown_dependencies: number;
  disabled_dependencies: number;
  total_plugins: number;
  healthy_plugins: number;
}

export interface ServerStatusTrendPoint {
  observed_at: string;
  cpu_percent: number;
  host_memory_used_percent: number;
  load_average_one_minute: number;
  load_average_five_minutes: number;
  load_average_fifteen_minutes: number;
  goroutines: number;
  runtime_alloc_bytes: number;
  runtime_heap_in_use_bytes: number;
  runtime_sys_bytes: number;
}

export interface ServerStatusTrend {
  range: MonitorTrendRange;
  retention_seconds: number;
  sample_interval_seconds: number;
  points: ServerStatusTrendPoint[];
}

export interface ServerStatusResponse {
  status: string;
  observed_at: string;
  server: ServerStatusServer;
  runtime: ServerStatusRuntime;
  dependencies: ServerStatusDependencies;
  summary: ServerStatusSummary;
  trend: ServerStatusTrend;
  plugins: ServerStatusPlugin[];
}
