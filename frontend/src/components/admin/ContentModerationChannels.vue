<template>
 <div class="audit-channels space-y-4">
  <p v-if="error" class="rounded-xl bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ error }}</p>
<div v-if="section === 'prompt'" class="space-y-5">
  <p class="hint">{{ t('moderationV2.channelPromptHint') }}</p>
  <article v-for="p in config.providers" :key="p.id" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
    <h3 class="font-medium">{{ p.name || p.id }} · {{ p.model }}</h3>
              <label class="field">{{ t('moderationV2.prompt') }}<textarea v-model="p.audit_prompt" class="input min-h-36" rows="6" spellcheck="false" /></label>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50"><summary class="cursor-pointer text-sm">{{ t('moderationV2.payload') }}</summary><p class="hint my-3">{{ t('moderationV2.payloadHint') }}</p><textarea v-model="p.payload_script" class="input font-mono text-xs" rows="7" :aria-label="t('moderationV2.payload')" spellcheck="false" /></details>
  </article>
</div>
        <div v-if="section === 'services'" class="space-y-4">
          <div class="flex justify-end"><button type="button" class="btn btn-secondary" data-test="add-provider" @click="addProvider">{{ t('moderationV2.add') }}</button></div>
          <p v-if="!config.providers.length" class="rounded-xl border border-dashed border-gray-200 p-8 text-center text-gray-500 dark:border-dark-600">{{ t('moderationV2.empty') }}</p>
          <details v-for="(p, index) in config.providers" :key="p.id" class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" :open="expanded === p.id">
            <summary class="cursor-pointer p-4 font-medium text-gray-900 dark:text-gray-100" @click.prevent="expanded = expanded === p.id ? '' : p.id">{{ p.name || `${t('moderationV2.services')} ${index + 1}` }} <span class="ml-2 text-xs font-normal text-gray-500">{{ p.model }} · {{ p.enabled ? t('moderationV2.enabled') : t('moderationV2.disabled') }}</span></summary>
            <div class="space-y-4 border-t border-gray-100 p-4 dark:border-dark-700">
              <label class="flex items-center gap-2 text-sm"><input v-model="p.enabled" type="checkbox" />{{ t('moderationV2.enabled') }}</label>
              <label class="flex items-center gap-2 text-sm"><input v-model="p.audit_validated" type="checkbox" />{{ t('moderationV2.auditValidated') }}</label>
              <p class="hint">{{ t('moderationV2.auditValidatedHint') }}</p>
              <div class="grid gap-4 md:grid-cols-2">
                <label class="field">{{ t('moderationV2.providerName') }}<input v-model="p.name" class="input" maxlength="120" :data-test="`name-${index}`" /></label>
                <label class="field">{{ t('moderationV2.model') }}<input v-model="p.model" class="input" maxlength="256" /></label>
                <label class="field md:col-span-2">{{ t('moderationV2.baseURL') }}<input v-model="p.base_url" class="input" type="url" placeholder="https://provider.example/v1" spellcheck="false" /></label>
              </div>
              <label class="field">{{ t('moderationV2.keys') }}<textarea v-model="p.key_draft" class="input font-mono" rows="2" autocomplete="off" spellcheck="false" :disabled="p.clear_keys" :aria-describedby="`key-hint-${p.id}`" /></label>
              <p :id="`key-hint-${p.id}`" class="hint">{{ t('moderationV2.keyHint') }} {{ t('moderationV2.keySaved', { count: p.key_masks?.length || 0 }) }} <span class="font-mono">{{ p.key_masks?.join(' · ') }}</span></p>
              <label class="flex items-center gap-2 text-sm"><input v-model="p.clear_keys" type="checkbox" />{{ t('moderationV2.clearKeys') }}</label>
              <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <label class="field">{{ t('moderationV2.threshold') }}<input v-model.number="p.threshold" class="input" type="number" min="0" max="1" step="0.01" /></label>
                <label class="field">{{ t('moderationV2.timeout') }}<input v-model.number="p.timeout_ms" class="input" type="number" min="500" max="30000" step="500" /></label>
                <label class="field">{{ t('moderationV2.concurrency') }}<input v-model.number="p.max_concurrent" class="input" type="number" min="1" max="100" /></label>
                <label class="field">{{ t('moderationV2.inputLimit') }}<input v-model.number="p.max_input_tokens" class="input" type="number" min="256" max="65536" /></label>
                <label class="field">{{ t('moderationV2.outputLimit') }}<input v-model.number="p.max_output_tokens" class="input" type="number" min="32" max="4096" /></label>
                <label class="field">{{ t('moderationV2.outputParameter') }}<select v-model="p.output_parameter" class="input"><option>max_tokens</option><option>max_completion_tokens</option></select></label>
                <label class="field">{{ t('moderationV2.proxy') }}<input :value="p.proxy_id ?? ''" class="input" type="number" min="1" @input="p.proxy_id = ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null" /></label>
              </div>
              <p class="hint">{{ t('moderationV2.estimateHint') }}</p>
              <label class="flex items-start gap-2 text-sm"><input v-model="p.output_limit_verified" type="checkbox" class="mt-1" />{{ t('moderationV2.verified') }}</label>
              <div class="grid gap-4 sm:grid-cols-2">
                <label v-for="f in priceFields" :key="f.key" class="field">{{ t(`moderationV2.${f.label}`) }} · {{ config.currency }}<input v-model="p.prices[f.key]" class="input" inputmode="decimal" placeholder="—" /></label>
              </div>
              <p class="hint">{{ t('moderationV2.priceHint') }}</p>
              <h3 class="font-medium">{{ t('moderationV2.providerBudget') }}</h3>
              <AuditLimitFields v-model="p.limits" :labels="limitLabels" />
              <button type="button" class="text-sm text-red-600 hover:underline dark:text-red-400" @click="removeProvider(p.id)">{{ t('moderationV2.remove') }}</button>
            </div>
          </details>
        </div>
        <div v-if="section === 'policy'" class="space-y-5">
          <label class="field">{{ t('moderationV2.routing') }}<select v-model="config.routing" class="input" data-test="channel-routing"><option value="lowest_cost">{{ t('moderationV2.cheapest') }}</option><option value="priority">{{ t('moderationV2.priority') }}</option></select></label>
          <p class="hint">{{ t('moderationV2.routingHint') }}</p>
          <div class="grid gap-4 md:grid-cols-2">
            <label v-if="config.routing !== 'lowest_cost'" class="field">{{ t('moderationV2.primary') }}<select v-model="config.primary_id" class="input"><option value="">{{ t('moderationV2.choose') }}</option><option v-for="p in config.providers" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            <label v-if="config.routing !== 'lowest_cost'" class="field">{{ t('moderationV2.fallback') }}<select :value="config.fallback_ids[0] || ''" class="input" @change="setFallback(($event.target as HTMLSelectElement).value)"><option value="">{{ t('moderationV2.noFallback') }}</option><option v-for="p in config.providers.filter(p => p.id !== config!.primary_id)" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            <label class="field">{{ t('moderationV2.attempts') }}<select v-model.number="config.max_attempts" class="input"><option :value="1">1</option><option :value="2">2</option></select></label>
            <label class="field">{{ t('moderationV2.cache') }}<input v-model.number="config.cache_ttl_seconds" class="input" type="number" min="0" max="86400" /></label>
            <label class="field md:col-span-2">{{ t('moderationV2.unresolved') }}<select v-model="config.unresolved_policy" class="input" data-test="unresolved-policy"><option value="">{{ t('moderationV2.choose') }}</option><option value="reject_temporary">{{ t('moderationV2.reject') }}</option><option value="allow_record">{{ t('moderationV2.allow') }}</option></select></label>
          </div>
          <p class="hint">{{ t('moderationV2.policyHint') }}</p>
          <p class="rounded-xl bg-primary-50 p-4 text-sm leading-relaxed text-primary-900 dark:bg-primary-950/30 dark:text-primary-200">{{ t('moderationV2.coverage') }}</p>
        </div>
        <div v-if="section === 'usage'" class="space-y-5">
          <label class="field max-w-xs">{{ t('moderationV2.currency') }}<select v-model="config.currency" class="input" :disabled="config.revision > 0"><option>CNY</option><option>USD</option></select></label>
          <h3 class="font-medium">{{ t('moderationV2.globalBudget') }}</h3><AuditLimitFields v-model="config.limits" :labels="limitLabels" />
          <p class="hint leading-relaxed">{{ t('moderationV2.usageHint') }}</p>
          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-700"><span class="text-sm">{{ t('moderationV2.day') }}: {{ usage?.day || '—' }} UTC</span><button class="btn btn-secondary" type="button" :disabled="busy" @click="refreshUsage">{{ t('moderationV2.refresh') }}</button></div>
          <div v-if="usage" class="grid gap-3 sm:grid-cols-2"><div class="metric">{{ t('moderationV2.cacheHits') }}<strong>{{ usage.cache_hits }}</strong></div><div class="metric">{{ t('moderationV2.unresolvedCount') }}<strong>{{ usage.unresolved }}</strong></div></div>
          <p v-if="usage && !usage.providers.length" class="hint">{{ t('moderationV2.noUsage') }}</p>
          <article v-for="row in usage?.providers || []" :key="row.provider_id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700"><h3 class="mb-3 font-medium">{{ config.providers.find(p => p.id === row.provider_id)?.name || row.provider_id }}</h3><dl class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><div><dt class="hint">{{ t('moderationV2.calls') }}</dt><dd>{{ row.calls }}</dd></div><div><dt class="hint">{{ t('moderationV2.confirmed') }}</dt><dd>{{ row.confirmed_amount }} {{ usage?.currency }}</dd></div><div><dt class="hint">{{ t('moderationV2.held') }}</dt><dd>{{ row.held_amount }} {{ usage?.currency }}</dd></div><div><dt class="hint">{{ t('moderationV2.unknown') }}</dt><dd>{{ row.unknown_calls }}</dd></div><div class="sm:col-span-2"><dt class="hint">{{ t('moderationV2.tokenUsage') }}</dt><dd>{{ row.input }} / {{ row.cached_input }} / {{ row.output }}</dd></div></dl></article>
        </div>
        <div v-if="section === 'trial'" class="space-y-4">
          <p class="hint">{{ t('moderationV2.testHint') }}</p><label class="field">{{ t('moderationV2.testText') }}<textarea v-model="testText" class="input" rows="6" data-test="v2-test-text" /></label>
          <div class="flex flex-wrap gap-3"><button type="button" class="btn btn-secondary" :disabled="busy || !testText.trim()" data-test="preview-v2" @click="preview">{{ t('moderationV2.preview') }}</button><button type="button" class="btn btn-primary" :disabled="busy || !testText.trim() || !config.revision" data-test="test-v2" @click="test">{{ t('moderationV2.run') }}</button></div>
          <p v-if="busy" role="status" class="hint">{{ t('moderationV2.pending') }}</p>
          <div v-if="estimate" class="space-y-2 rounded-xl bg-gray-50 p-4 text-sm dark:bg-dark-900/50" role="status"><strong>{{ estimate.fits ? t('moderationV2.fits') : t('moderationV2.tooLong') }}</strong><p>{{ t('moderationV2.selectedChannel') }}: {{ config.providers.find(p => p.id === estimate!.provider_id)?.name || estimate.provider_id || '—' }}</p><p>{{ t('moderationV2.estimatedInput') }}: {{ estimate.estimated_input }} · {{ t('moderationV2.outputLimit') }}: {{ estimate.max_output }}</p><p>{{ t('moderationV2.reserve') }}: {{ estimate.reserved_amount || t('moderationV2.unknownAmount') }} {{ config.currency }}</p><p v-if="estimate.reason">{{ reasonLabel(estimate.reason) }}</p><p class="hint">{{ t('moderationV2.estimateHint') }}</p></div>
          <div v-if="result" class="space-y-3 rounded-xl border border-primary-200 p-4 text-sm dark:border-primary-900" role="status" data-test="v2-result">
            <strong>{{ result.status === 'reviewed' ? t('moderationV2.reviewed') : t('moderationV2.unresolvedResult') }}</strong><p>{{ t('moderationV2.actualAttempts') }}: {{ result.attempts }}</p><p v-if="result.reason">{{ reasonLabel(result.reason) }}</p>
            <template v-if="result.verdict"><p class="font-semibold" :class="result.verdict.flagged ? 'text-red-600 dark:text-red-400' : 'text-emerald-700 dark:text-emerald-300'">{{ result.verdict.flagged ? t('moderationV2.hit') : t('moderationV2.pass') }}</p><p>{{ result.verdict.provider_id }} · {{ result.verdict.model }}</p><p v-if="result.verdict.decision_source === 'flagged'">{{ t('moderationV2.boolean') }}</p><p v-else>{{ t('moderationV2.score') }}: {{ result.verdict.score }} / {{ result.verdict.threshold }}</p><p v-if="result.verdict.reason">{{ t('moderationV2.reason') }}: {{ result.verdict.reason }}</p></template>
            <p v-if="result.usage">{{ t('moderationV2.tokenUsage') }}: {{ result.usage.input }} / {{ result.usage.cached_input }} / {{ result.usage.output }}</p>
          </div>
        </div>
 </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AuditLimitFields from './moderation/AuditLimitFields.vue'
import { moderationV2API, newAuditProvider } from '@/api/admin/moderationV2'
import type { AuditConfig, AuditPreview, AuditResult, AuditUsage, AuditPrices } from '@/api/admin/moderationV2'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ section: 'services' | 'policy' | 'usage' | 'trial' | 'prompt' }>()
const config = defineModel<AuditConfig>({ required: true })
const { t, te } = useI18n()
const usage = ref<AuditUsage>()
const busy = ref(false)
const error = ref('')
const expanded = ref(config.value.providers[0]?.id || '')
const testText = ref('')
const result = ref<AuditResult>()
const estimate = ref<AuditPreview>()
const priceFields: { key: keyof AuditPrices; label: string }[] = [
  { key: 'input', label: 'inputPrice' }, { key: 'cached_input', label: 'cachedPrice' },
  { key: 'output', label: 'outputPrice' }, { key: 'per_request', label: 'requestPrice' },
]
const limitLabels = computed(() => ({ calls: t('moderationV2.dailyCalls'), tokens: t('moderationV2.dailyTokens'), amount: t('moderationV2.dailyAmount') }))
function reasonLabel(reason: string) { return te(`moderationV2.reasons.${reason}`) ? t(`moderationV2.reasons.${reason}`) : reason }
function displayError(e: unknown) { error.value = extractApiErrorMessage(e, t('moderationV2.failure')) }
function addProvider() { const p = newAuditProvider(); config.value.providers.push(p); expanded.value = p.id }
function removeProvider(id: string) {
  config.value.providers = config.value.providers.filter(p => p.id !== id)
  config.value.fallback_ids = config.value.fallback_ids.filter(v => v !== id)
  if (config.value.primary_id === id) config.value.primary_id = ''
}
function setFallback(id: string) { config.value.fallback_ids = id ? [id] : [] }
async function refreshUsage() { busy.value = true; error.value = ''; try { usage.value = await moderationV2API.usage() } catch (e) { displayError(e) } finally { busy.value = false } }
async function preview() { busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined; try { estimate.value = await moderationV2API.preview(testText.value) } catch (e) { displayError(e) } finally { busy.value = false } }
async function test() { busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined; try { result.value = await moderationV2API.test(testText.value) } catch (e) { displayError(e) } finally { busy.value = false } }
onMounted(() => { if (props.section === 'usage') void refreshUsage() })
</script>
<style scoped>
.field { display: flex; flex-direction: column; gap: .5rem; font-size: .875rem; min-width: 0; }
.field .input { width: 100%; min-width: 0; }
.hint { font-size: .75rem; line-height: 1.7; color: #746d80; overflow-wrap: anywhere; }
:global(.dark) .hint { color: #aaa1b8; }
.metric { border-radius: .75rem; background: rgb(128 103 178 / .07); padding: 1rem; font-size: .875rem; }
.metric strong { display: block; margin-top: .5rem; font-size: 1.5rem; }
</style>
