import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PriceLookup from '../moderation/AuditPriceLookup.vue'
import { moderationV2API, type AuditModelPrices } from '@/api/admin/moderationV2'
import { moderationV2Messages } from '@/i18n/moderationV2'
vi.mock('@/api/admin/moderationV2', () => ({ moderationV2API: { modelPrices: vi.fn() } }))
const response = (): AuditModelPrices => ({ source: 'https://models.dev/api.json', currency: 'USD', fetched_at: '2026-09-23T00:00:00Z', stale: false, matched_provider_id: 'opencode', total: 1, items: [{ provider_id: 'opencode', provider_name: 'OpenCode Zen', model_id: 'deepseek-flash', model_name: 'DeepSeek Flash', matches_channel: true, prices: { input: '0.15', output: '0.6', cached_input: '0.003', per_request: '' }, importable: true, context: 1000000 }] })
const render = (currency: 'USD' | 'CNY' = 'USD') => mount(PriceLookup, { props: { model: 'deepseek-flash', baseUrl: 'https://opencode.ai/zen/v1', currency }, global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh: { moderationV2: moderationV2Messages.zh } } })] } })
beforeEach(() => { vi.mocked(moderationV2API.modelPrices).mockReset(); vi.mocked(moderationV2API.modelPrices).mockResolvedValue(response()) })
async function open(w: ReturnType<typeof render>) { await w.get('[data-test="lookup-prices"]').trigger('click'); await flushPromises() }
describe('public model price lookup', () => {
  it('looks up on demand and imports only prices, not model quality or credentials', async () => {
    const w=render(); expect(moderationV2API.modelPrices).not.toHaveBeenCalled()
    await open(w)
    expect(moderationV2API.modelPrices).toHaveBeenCalledWith('deepseek-flash','https://opencode.ai/zen/v1')
    expect(w.emitted('apply')).toBeUndefined()
    await w.get('[data-test="apply-prices"]').trigger('click')
    expect(w.emitted('apply')).toEqual([[{ input:'0.15', output:'0.6', cached_input:'0.003' }]])
  })
  it('requires explicit conversion for CNY and preserves the ledger currency', async () => {
    const w=render('CNY');await open(w)
    expect(w.get('[data-test="apply-prices"]').attributes('disabled')).toBeDefined()
    await w.get('[data-test="usd-rate"]').setValue('7.2')
    await w.get('[data-test="apply-prices"]').trigger('click')
    expect(w.emitted('apply')).toEqual([[{ input:'1.08', output:'4.32', cached_input:'0.0216' }]])
    expect(w.props('currency')).toBe('CNY')
  })
  it('does not automatically select a different provider with the same model name', async () => {
    const data=response();data.items[0]!.matches_channel=false;data.matched_provider_id=''
    vi.mocked(moderationV2API.modelPrices).mockResolvedValue(data)
    const w=render();await open(w)
    expect(w.find('[data-test="apply-prices"]').exists()).toBe(false)
    await w.get('[data-test="price-entry"]').setValue(JSON.stringify(['opencode','deepseek-flash']))
    expect(w.get('[data-test="apply-prices"]').attributes('disabled')).toBeDefined()
    await w.get('[data-test="confirm-reference"]').setValue(true)
    await w.get('[data-test="apply-prices"]').trigger('click')
    expect(w.emitted('apply')).toHaveLength(1)
  })
  it('keeps tiered prices read-only and does not import a base rate as the full tariff', async () => {
    const data=response();data.items[0]!.importable=false;data.items[0]!.manual_reason='tiered_price'
    vi.mocked(moderationV2API.modelPrices).mockResolvedValue(data)
    const w=render();await open(w)
    expect(w.text()).toContain('阶梯')
    expect(w.find('[data-test="apply-prices"]').exists()).toBe(false)
  })
  it('ignores a response for a channel whose URL changed while fetching', async () => {
    let resolve!: (value:AuditModelPrices)=>void
    vi.mocked(moderationV2API.modelPrices).mockReturnValue(new Promise(done=>{resolve=done}))
    const w=render();await w.get('[data-test="lookup-prices"]').trigger('click')
    await w.setProps({baseUrl:'https://another.example/v1'})
    resolve(response());await flushPromises()
    expect(w.find('[data-test="price-entry"]').exists()).toBe(false)
    expect(w.emitted('apply')).toBeUndefined()
  })
  it('keeps catalog failure separate from existing saved prices', async () => {
    vi.mocked(moderationV2API.modelPrices).mockRejectedValue(new Error('offline'))
    const w=render();await open(w)
    expect(w.get('[role="alert"]').text()).toContain('已有价格不变')
    expect(w.emitted('apply')).toBeUndefined()
  })
})
