import { apiClient } from '../client'

export interface AuditLimits { daily_calls: number; daily_tokens: number; daily_amount: string }
export interface AuditPrices { input: string; cached_input: string; output: string; per_request: string }
export interface AuditPolicy {
  mode: 'legacy' | 'balanced' | 'quality'; context_mode: 'current' | 'bounded' | 'full'
  history_messages: number; history_bytes: number; review_history_messages: number; review_history_bytes: number
  review_ids: string[]; uncertainty_margin: number; request_timeout_ms: number; max_request_amount: string
}
export const defaultAuditPolicy = (): AuditPolicy => ({ mode: 'legacy', context_mode: 'bounded', history_messages: 4, history_bytes: 8192, review_history_messages: 12, review_history_bytes: 32768, review_ids: [], uncertainty_margin: 0.1, request_timeout_ms: 60000, max_request_amount: '' })
export interface AuditCoverage { scope: string; current_complete: boolean; required_context_complete: boolean; selected_messages: number; omitted_messages: number; selected_bytes: number; missing_context: boolean }
export interface AuditTrace { stage: string; provider_id: string; reason: string; decision?: string; reserved_amount: string; accounting_state: string; header_ms: number; first_text_ms?: number; total_ms: number; coverage?: AuditCoverage }
export interface AuditTestInput { text?: string; protocol?: string; body?: unknown }
export interface AuditProvider {
  purpose?: 'primary' | 'review' | 'both'; api_format?: 'chat_completions' | 'responses'; stream?: boolean
  reasoning_parameter?: 'none' | 'effort' | 'thinking' | 'enable_thinking'; reasoning_effort?: string; thinking_enabled?: boolean
  header_timeout_ms?: number; idle_timeout_ms?: number
  audit_validated?: boolean; id: string; name: string; enabled: boolean; base_url: string; model: string; proxy_id: number | null
  key_draft?: string; api_keys?: string[]; key_masks?: string[]; clear_keys?: boolean
  audit_prompt: string; payload_script: string; threshold: number; timeout_ms: number
  max_input_tokens: number; max_output_tokens: number; output_parameter: 'max_tokens' | 'max_completion_tokens'
  output_limit_verified: boolean; max_concurrent: number; prices: AuditPrices; limits: AuditLimits
}
export interface AuditConfig {
  policy?: AuditPolicy
  routing?: 'priority' | 'lowest_cost'; revision: number; enabled: boolean; currency: 'CNY' | 'USD'; unresolved_policy: '' | 'reject_temporary' | 'allow_record'
  primary_id: string; fallback_ids: string[]; max_attempts: number; cache_ttl_seconds: number
  limits: AuditLimits; providers: AuditProvider[]
}
export interface AuditUsage {
  day: string; currency: string; cache_hits: number; unresolved: number
  providers: { provider_id: string; calls: number; held_amount: string; confirmed_amount: string; unknown_calls: number; input: number; output: number; cached_input: number }[]
}
export interface AuditPreview { stage?: string; coverage?: AuditCoverage; fragments?: { id: string; role: string; kind: string; text: string; current: boolean }[]; provider_id: string; estimated_input: number; max_output: number; reserved_amount: string; fits: boolean; reason: string }
export interface AuditResult {
  coverage?: AuditCoverage; traces?: AuditTrace[]; total_ms?: number
  status: 'reviewed' | 'unresolved'; reason: string; cache_hit: boolean; attempts: number; estimated_input: number
  verdict?: { flagged: boolean; score: number; reason: string; decision_source: string; provider_id: string; model: string; threshold: number }
  usage?: { input: number; cached_input: number; output: number }
}
const root = '/admin/risk-control/v2'
export const moderationV2API = {
  config: async () => (await apiClient.get<AuditConfig>(`${root}/config`)).data,
  save: async (data: AuditConfig) => (await apiClient.put<AuditConfig>(`${root}/config`, data)).data,
  usage: async () => (await apiClient.get<AuditUsage>(`${root}/usage`)).data,
  preview: async (input: string | AuditTestInput) => (await apiClient.post<AuditPreview>(`${root}/preview`, typeof input === 'string' ? { text: input } : input)).data,
  test: async (input: string | AuditTestInput) => (await apiClient.post<AuditResult>(`${root}/test`, typeof input === 'string' ? { text: input } : input, { timeout: 270000 })).data,
}
export const emptyAuditLimits = (): AuditLimits => ({ daily_calls: 0, daily_tokens: 0, daily_amount: '' })
export function newAuditProvider(): AuditProvider {
  return { purpose: 'primary', api_format: 'chat_completions', stream: false, reasoning_parameter: 'none', reasoning_effort: 'low', thinking_enabled: false, header_timeout_ms: 0, idle_timeout_ms: 0, id: `audit-${crypto.randomUUID().slice(0, 8)}`, name: '', enabled: false, base_url: '', model: '', proxy_id: null,
    audit_prompt: '', payload_script: '', threshold: 0.85, timeout_ms: 5000, max_input_tokens: 4096,
    max_output_tokens: 512, output_parameter: 'max_tokens', output_limit_verified: false, max_concurrent: 4,
    prices: { input: '', cached_input: '', output: '', per_request: '' }, limits: emptyAuditLimits() }
}
