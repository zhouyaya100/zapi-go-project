// ========== API Response ==========
export interface ApiResponse<T = unknown> {
  success?: boolean
  message?: string
  error?: { message: string }
  data?: T
}

export interface PaginatedData<T> {
  items: T[]
  total: number
  page?: number
  page_size?: number
}

// ========== User & Auth ==========
export interface User {
  id: number
  username: string
  role: 'admin' | 'operator' | 'user'
  group_id?: number | null
  group_name?: string
  enabled?: boolean
  max_tokens?: number
  token_quota?: number
  token_quota_used?: number
  token_count?: number
  bind_mode?: 'inherit' | 'custom'
  allowed_models?: string
  authed_models?: string[]
  group_allowed_models?: string
  group_upstream_group_ids?: number[]
  group_upstream_group_names?: string[]
  effective_upstream_group_ids?: number[]
  effective_upstream_group_names?: string[]
  rate_mode?: string
  rpm?: number
  tpm?: number
  model_rate_limits?: string
  upstream_group_ids?: number[]
  upstream_group_names?: string[]
  can_create_token?: boolean
  is_super?: boolean
  created_at?: string
}

export interface LoginForm {
  username: string
  password: string
  captcha_id?: string
  captcha_code?: string
}

export interface LoginResponse {
  success: boolean
  token: string
  user: { id: number; username: string; role: string; max_tokens?: number; allowed_models?: string; token_quota?: number; token_quota_used?: number; is_super?: boolean }
}

export interface CaptchaData {
  captcha_id: string
  captcha_image: string
}

// ========== Channel ==========
export interface Channel {
  id: number
  name: string
  type: string
  base_url: string
  api_key: string
  api_key_length?: number
  models: string
  model_mapping: string
  allowed_groups: string
  weight: number
  priority: number
  enabled: boolean
  auto_ban: boolean
  fail_count: number
  test_time?: string | null
  response_time: number
  upstream_group_ids?: number[]
  created_at?: string
}

// ========== Upstream Group ==========
export interface UpstreamGroup {
  id: number
  name: string
  alias: string
  strategy: string
  allowed_groups: string
  enabled: boolean
  health_check_interval: number
  max_fails: number
  fail_timeout: number
  retry_on_fail: boolean
  channels?: UpstreamGroupChannel[]
  created_at?: string
}

export interface UpstreamGroupChannel {
  id?: number
  upstream_group_id?: number
  channel_id: number
  channel_name?: string
  name?: string
  type?: string
  weight?: number
  priority?: number
  enabled?: boolean
  fail_count?: number
  response_time?: number
}

// ========== Group ==========
export interface Group {
  id: number
  name: string
  comment: string
  rate_mode: string
  rpm: number
  tpm: number
  model_rate_limits: string
  allowed_models?: string
  upstream_group_ids?: number[]
  upstream_group_names?: string[]
  user_count?: number
  created_at?: string
}

// ========== Token ==========
export interface Token {
  id: number
  user_id: number
  username?: string
  name: string
  key: string
  models: string
  enabled: boolean
  quota_limit: number
  quota_used: number
  expires_at?: string | null
  created_at?: string
}

// ========== Log ==========
export interface LogEntry {
  id: number
  user_id?: number
  username?: string
  token_name?: string
  channel_name?: string
  model?: string
  is_stream?: boolean
  prompt_tokens?: number
  completion_tokens?: number
  cached_tokens?: number
  latency_ms?: number
  success?: boolean
  error_msg?: string
  client_ip?: string
  created_at?: string
}

// ========== Notification ==========
export interface Notification {
  id: number
  category: string
  title: string
  content: string
  sender_id?: number
  sender_name?: string
  receiver_id?: number | null
  read?: boolean
  recipient_count?: number
  created_at?: string
}

// ========== Settings ==========
export interface Settings {
  jwt_expire_hours?: number
  cors_origins?: string
  proxy_timeout?: number
  proxy_max_connections?: number
  proxy_max_keepalive?: number
  proxy_keepalive_expiry?: number
  proxy_retry_count?: number
  proxy_max_fails?: number
  proxy_fail_timeout?: number
  cache_enabled?: boolean
  cache_ttl?: number
  cache_max_entries?: number
  log_batch_size?: number
  log_batch_interval?: number
  log_retention_days?: number
  log_cleanup_interval_hours?: number
  log_cleanup_batch_size?: number
  error_log_max_entries?: number
  error_log_max_days?: number
  allow_register?: boolean
  default_max_tokens?: number
  default_token_quota?: number
  default_group?: string
  min_password_length?: number
  timezone_offset?: number
  heartbeat_enabled?: boolean
  heartbeat_interval?: number
  heartbeat_timeout?: number
  server_host?: string
  server_port?: number
  database_url?: string
  db_pool_size?: number
  db_max_overflow?: number
  groups?: { id: number; name: string; comment: string }[]
  all_models?: string[]
}

export interface PublicSettings {
  allow_register: boolean
  groups: { id: number; name: string; comment: string }[]
  all_models: string[]
}

// ========== Dashboard & Stats ==========
export interface DashboardData {
  recent_logs: { model: string; latency_ms: number; success: boolean; created_at: string }[]
  model_stats: { model: string; count: number; avg_latency: number }[]
  rpm: number
  tpm: number
  rate_mode: string
  model_limits: Record<string, { rpm: number; tpm: number }>
  token_count: number
  max_tokens: number
  token_quota: number
  token_quota_used: number
  group_name: string
  authorized_models: string[]
  total_requests: number
  success_requests: number
  total_prompt_tokens: number
  total_completion_tokens: number
  total_cached_tokens?: number
  total_uncached_tokens?: number
  platform_total_requests?: number
  platform_success_requests?: number
  platform_total_prompt_tokens?: number
  platform_total_completion_tokens?: number
  platform_total_tokens?: number
  platform_total_cached_tokens?: number
  platform_total_uncached_tokens?: number
}

export interface MyDashboardData {
  username: string
  token_count: number
  max_tokens: number
  token_quota: number
  token_quota_used: number
  group_name: string
  authorized_models: string[]
  total_requests: number
  success_requests: number
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
  total_cached_tokens?: number
  total_uncached_tokens?: number
  rate_mode: string
  rpm: number
  tpm: number
  model_limits: Record<string, { rpm: number; tpm: number }>
  recent_24h_requests: number
  recent_24h_tokens: number
  model_stats: { model: string; count: number; avg_latency: number }[]
  recent_logs: { model: string; prompt_tokens: number; completion_tokens: number; latency_ms: number; success: boolean; created_at: string }[]
}

export interface StatsData {
  total_requests: number
  success_requests: number
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
  total_cached_tokens?: number
  total_uncached_tokens?: number
  avg_latency_ms: number
  channels: number
  channels_enabled: number
  users: number
  users_enabled: number
  tokens: number
  tokens_enabled: number
  recent_24h_requests: number
  recent_24h_tokens: number
}

export interface UsageItem {
  key: string
  requests: number
  success: number
  fail?: number
  success_rate: string
  prompt_tokens: number
  completion_tokens: number
  cached_tokens?: number
  uncached_tokens?: number
  total_tokens: number
  avg_latency_ms: number
  user?: string
  model?: string
  channel?: string
}

export interface UsageResponse {
  summary: {
    total_requests: number
    success_requests: number
    total_prompt_tokens: number
    total_completion_tokens: number
    total_tokens: number
    total_cached_tokens: number
    total_uncached_tokens: number
    avg_latency_ms: number
  }
  items: UsageItem[]
  total: number
  page: number
  page_size: number
}

// ========== LB Status ==========
export interface LBChannelStatus {
  id: number
  name: string
  weight: number
  priority: number
  status: string
  active_requests: number
  total_requests: number
  global_total_requests: number
  success_rate: string
  global_success_rate: string
  avg_latency_ms: number
  fail_count: number
  circuit: string
  shared: boolean
  response_time: number
  heart_fail_count: number
}

export interface LBGroupStatus {
  id: number
  name: string
  strategy: string
  channels: LBChannelStatus[]
}

export interface VersionInfo { version: string }
export interface ErrorLogEntry { time: string; type: string; message: string }
