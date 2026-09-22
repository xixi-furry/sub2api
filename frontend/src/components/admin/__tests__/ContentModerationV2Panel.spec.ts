import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { defineComponent } from 'vue'
import Panel from '../ContentModerationV2Panel.vue'
import { moderationV2API } from '@/api/admin/moderationV2'
import type { AuditConfig } from '@/api/admin/moderationV2'
import { moderationV2Messages } from '@/i18n/moderationV2'

vi.mock('@/api/admin/moderationV2', async importOriginal => ({
  ...await importOriginal<typeof import('@/api/admin/moderationV2')>(),
  moderationV2API: { config: vi.fn(), save: vi.fn(), usage: vi.fn(), preview: vi.fn(), test: vi.fn() },
}))
const fixture = (): AuditConfig => ({ revision: 1, enabled: false, currency: 'CNY', unresolved_policy: '', primary_id: 'existing', fallback_ids: [], max_attempts: 2, cache_ttl_seconds: 900,
  limits: { daily_calls: 0, daily_tokens: 0, daily_amount: '' }, providers: [{ id: 'existing', name: 'My provider', enabled: true, base_url: 'https://provider.example/v1', model: 'small', proxy_id: null, key_masks: ['****tail'], audit_prompt: 'Return confidence JSON', payload_script: '', threshold: 0.85, timeout_ms: 5000, max_input_tokens: 4096, max_output_tokens: 512, output_parameter: 'max_tokens', output_limit_verified: false, max_concurrent: 4, prices: { input: '', cached_input: '', output: '', per_request: '' }, limits: { daily_calls: 0, daily_tokens: 0, daily_amount: '' } }] })
async function render() {
  const wrapper = mount(Panel, { global: {
    plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh: { moderationV2: moderationV2Messages.zh }, en: { moderationV2: moderationV2Messages.en } } })],
    stubs: { BaseDialog: defineComponent({ template: '<section><slot/><slot name="footer"/></section>' }) },
  } }); await flushPromises(); return wrapper
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(moderationV2API.config).mockResolvedValue(fixture())
  vi.mocked(moderationV2API.save).mockImplementation(async c => ({ ...c, revision: c.revision + 1 }))
  vi.mocked(moderationV2API.usage).mockResolvedValue({ day: '2026-09-22', currency: 'CNY', cache_hits: 0, unresolved: 0, providers: [] })
})
describe('provider moderation console', () => {
  it('does not enable new providers and preserves blank saved keys', async () => {
    const w = await render()
    await w.get('[data-test="add-provider"]').trigger('click')
    await w.get('[data-test="name-1"]').setValue('Second provider')
    await w.get('[data-test="save-v2"]').trigger('click'); await flushPromises()
    const draft = vi.mocked(moderationV2API.save).mock.calls[0]![0]
    expect(draft.enabled).toBe(false); expect(draft.providers[1]!.enabled).toBe(false)
    expect(draft.providers[0]!.api_keys).toBeUndefined(); expect(draft.providers[0]!.key_masks).toBeUndefined()
    expect(moderationV2API.test).not.toHaveBeenCalled()
  })
  it('retains edits and key input on a revision conflict', async () => {
    vi.mocked(moderationV2API.save).mockRejectedValue({ status: 409 })
    const w = await render()
    await w.get('[data-test="name-0"]').setValue('My unsaved edit')
    await w.findAll('textarea')[0]!.setValue('new-key-a\nnew-key-b')
    await w.get('[data-test="save-v2"]').trigger('click'); await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('配置已被其他页面修改')
    expect((w.get('[data-test="name-0"]').element as HTMLInputElement).value).toBe('My unsaved edit')
    expect((w.findAll('textarea')[0]!.element as HTMLTextAreaElement).value).toContain('new-key-a')
    expect(vi.mocked(moderationV2API.save).mock.calls[0]![0].providers[0]!.api_keys).toEqual(['new-key-a', 'new-key-b'])
  })
  it('previews without making a billable call and labels unresolved tests', async () => {
    vi.mocked(moderationV2API.preview).mockResolvedValue({ provider_id: 'existing', estimated_input: 5000, max_output: 512, reserved_amount: '', fits: false, reason: 'input_budget_exceeded' })
    vi.mocked(moderationV2API.test).mockResolvedValue({ status: 'unresolved', reason: 'budget_or_concurrency_exhausted', attempts: 0, cache_hit: false, estimated_input: 100 })
    const w = await render(); await w.get('[data-test="v2-test-text"]').setValue('test')
    await w.get('[data-test="preview-v2"]').trigger('click'); await flushPromises()
    expect(moderationV2API.test).not.toHaveBeenCalled(); expect(w.text()).toContain('输入超出预算')
    await w.get('[data-test="test-v2"]').trigger('click'); await flushPromises()
    const result = w.get('[data-test="v2-result"]').text()
    expect(result).toContain('未完成审核'); expect(result).toContain('预算或并发额度不足'); expect(result).not.toContain('未命中')
  })
  it('disables paid tests until the configuration has been saved', async () => {
    vi.mocked(moderationV2API.config).mockResolvedValue({ ...fixture(), revision: 0 })
    const w = await render(); await w.get('[data-test="v2-test-text"]').setValue('test')
    expect((w.get('[data-test="test-v2"]').element as HTMLButtonElement).disabled).toBe(true)
  })
  it('has matching Chinese and English labels including failure reasons', () => {
    expect(Object.keys(moderationV2Messages.zh).sort()).toEqual(Object.keys(moderationV2Messages.en).sort())
    expect(Object.keys(moderationV2Messages.zh.reasons).sort()).toEqual(Object.keys(moderationV2Messages.en.reasons).sort())
  })
})
