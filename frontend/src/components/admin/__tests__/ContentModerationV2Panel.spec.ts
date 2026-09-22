import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import Panel from '../ContentModerationChannels.vue'
import { moderationV2API } from '@/api/admin/moderationV2'
import type { AuditConfig } from '@/api/admin/moderationV2'
import { moderationV2Messages } from '@/i18n/moderationV2'

vi.mock('@/api/admin/moderationV2', async importOriginal => ({
  ...await importOriginal<typeof import('@/api/admin/moderationV2')>(),
  moderationV2API: { config: vi.fn(), save: vi.fn(), usage: vi.fn(), preview: vi.fn(), test: vi.fn() },
}))
const fixture = (): AuditConfig => ({ revision: 1, enabled: false, currency: 'CNY', unresolved_policy: '', primary_id: 'existing', fallback_ids: [], max_attempts: 2, cache_ttl_seconds: 900,
  limits: { daily_calls: 0, daily_tokens: 0, daily_amount: '' }, providers: [{ id: 'existing', name: 'My provider', enabled: true, base_url: 'https://provider.example/v1', model: 'small', proxy_id: null, key_masks: ['****tail'], audit_prompt: 'Return confidence JSON', payload_script: '', threshold: 0.85, timeout_ms: 5000, max_input_tokens: 4096, max_output_tokens: 512, output_parameter: 'max_tokens', output_limit_verified: false, max_concurrent: 4, prices: { input: '', cached_input: '', output: '', per_request: '' }, limits: { daily_calls: 0, daily_tokens: 0, daily_amount: '' } }] })
async function render(section: 'services' | 'policy' | 'trial' | 'prompt' = 'services', model = fixture()) {
  const wrapper = mount(Panel, { props: { modelValue: model, section }, global: {
    plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh: { moderationV2: moderationV2Messages.zh }, en: { moderationV2: moderationV2Messages.en } } })],
  } }); await flushPromises(); return wrapper
}
beforeEach(() => { vi.clearAllMocks() })
describe('channels inside risk-control settings', () => {
  it('adds a disabled channel to the parent draft without saving independently', async () => {
    const w = await render()
    await w.get('[data-test="add-provider"]').trigger('click')
    await w.get('[data-test="name-1"]').setValue('OpenCode DeepSeek')
    const draft = w.props('modelValue')
    expect(draft.providers[1]!.enabled).toBe(false)
    expect(draft.providers[1]!.name).toBe('OpenCode DeepSeek')
    expect(moderationV2API.save).not.toHaveBeenCalled()
    expect(w.find('[role="dialog"]').exists()).toBe(false)
  })
  it('keeps pending key changes in the same parent draft across tabs', async () => {
    const w = await render()
    await w.findAll('textarea')[0]!.setValue('new-key-a\nnew-key-b')
    await w.setProps({ section: 'prompt' })
    await w.findAll('textarea')[0]!.setValue('Updated audit rules')
    expect(w.props('modelValue').providers[0]!.key_draft).toBe('new-key-a\nnew-key-b')
    expect(w.props('modelValue').providers[0]!.audit_prompt).toBe('Updated audit rules')
    expect(moderationV2API.save).not.toHaveBeenCalled()
  })
  it('hides fixed routes when automatic channel pricing is selected', async () => {
    const w = await render('policy', { ...fixture(), routing: 'lowest_cost' })
    expect(w.text()).not.toContain('主审核服务')
    await w.get('[data-test="channel-routing"]').setValue('priority')
    expect(w.text()).toContain('主审核服务')
  })
  it('previews without a paid request and preserves unresolved status', async () => {
    vi.mocked(moderationV2API.preview).mockResolvedValue({ provider_id: 'existing', estimated_input: 5000, max_output: 512, reserved_amount: '', fits: false, reason: 'input_budget_exceeded' })
    vi.mocked(moderationV2API.test).mockResolvedValue({ status: 'unresolved', reason: 'budget_or_concurrency_exhausted', attempts: 0, cache_hit: false, estimated_input: 100 })
    const w = await render('trial'); await w.get('[data-test="v2-test-text"]').setValue('test')
    await w.get('[data-test="preview-v2"]').trigger('click'); await flushPromises()
    expect(moderationV2API.test).not.toHaveBeenCalled(); expect(w.text()).toContain('输入超出预算')
    await w.get('[data-test="test-v2"]').trigger('click'); await flushPromises()
    expect(w.get('[data-test="v2-result"]').text()).toContain('未完成审核')
    expect(w.get('[data-test="v2-result"]').text()).not.toContain('未命中')
  })
  it('disables paid tests before the parent configuration is saved', async () => {
    const w = await render('trial', { ...fixture(), revision: 0 }); await w.get('[data-test="v2-test-text"]').setValue('test')
    expect((w.get('[data-test="test-v2"]').element as HTMLButtonElement).disabled).toBe(true)
  })
  it('keeps protocol and reasoning choices in the parent channel draft', async () => {
    const w = await render()
    await w.get('[data-test="provider-purpose"]').setValue('review')
    await w.get('[data-test="provider-protocol"]').setValue('responses')
    expect(w.props('modelValue').providers[0]!.api_format).toBe('responses')
    expect(w.props('modelValue').providers[0]!.purpose).toBe('review')
    expect(w.text()).not.toContain('enable_thinking')
    expect(w.text()).not.toContain('输出限制参数')
    await w.setProps({ section: 'prompt' })
    expect(w.props('modelValue').providers[0]!.api_format).toBe('responses')
  })
  it('exposes context and shared budgets only after opting into enhanced review', async () => {
    const w = await render('policy')
    expect(w.find('[data-test="context-mode"]').exists()).toBe(false)
    await w.get('[data-test="audit-mode"]').setValue('quality')
    await w.get('[data-test="context-mode"]').setValue('full')
    expect(w.props('modelValue').policy!.context_mode).toBe('full')
    expect(w.text()).toContain('整个审核总超时')
    expect(w.get('[data-test="unresolved-policy"]').text()).not.toContain('允许请求，记录为未完成审核')
    expect(moderationV2API.save).not.toHaveBeenCalled()
  })
  it('sends structured request previews without a model call and displays coverage', async () => {
    vi.mocked(moderationV2API.preview).mockResolvedValue({ provider_id: 'existing', estimated_input: 600, max_output: 512, reserved_amount: '0.001', fits: true, reason: '', coverage: { scope: 'bounded', current_complete: true, required_context_complete: true, selected_messages: 2, omitted_messages: 4, selected_bytes: 80, missing_context: false }, fragments: [{ id: 'e0', role: 'user', kind: 'text', text: 'original goal', current: false }] })
    const w = await render('trial')
    await w.get('select').setValue('openai_chat_completions')
    const body = { messages: [{ role: 'user', content: 'original goal' }, { role: 'user', content: 'continue' }] }
    await w.get('[data-test="v2-test-text"]').setValue(JSON.stringify(body))
    await w.get('[data-test="preview-v2"]').trigger('click'); await flushPromises()
    expect(moderationV2API.preview).toHaveBeenCalledWith({ protocol: 'openai_chat_completions', body })
    expect(moderationV2API.test).not.toHaveBeenCalled()
    expect(w.get('[data-test="context-coverage"]').text()).toContain('省略 4 个历史片段')
    expect(w.text()).toContain('original goal')
  })
  it('keeps both locales complete', () => {
    expect(Object.keys(moderationV2Messages.zh).sort()).toEqual(Object.keys(moderationV2Messages.en).sort())
    expect(Object.keys(moderationV2Messages.zh.reasons).sort()).toEqual(Object.keys(moderationV2Messages.en.reasons).sort())
  })
})
