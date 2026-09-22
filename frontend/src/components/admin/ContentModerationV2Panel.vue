<template>
  <BaseDialog :show="true" :title="t('title')" width="extra-wide" @close="$emit('close')">
    <div class="audit-v2 space-y-5">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('intro') }}</p>
      <p v-if="error" class="rounded-xl bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ error }}</p>
      <p v-if="notice" class="text-sm text-emerald-700 dark:text-emerald-300" role="status">{{ notice }}</p>
      <div v-if="loading" role="status">{{ t('loading') }}</div>
      <template v-else-if="config">
        <nav class="flex flex-wrap gap-2 border-b border-gray-100 pb-3 dark:border-dark-700" :aria-label="t('title')">
          <button v-for="item in tabs" :key="item" type="button" class="rounded-lg px-3 py-2 text-sm font-medium transition-colors" :class="tab === item ? 'bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-200' : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-dark-700'" :aria-pressed="tab === item" @click="tab = item">{{ t(item) }}</button>
        </nav>
        <p v-if="dirty" class="text-sm text-amber-700 dark:text-amber-300" role="status">{{ t('dirty') }}</p>
        <div v-show="tab === 'services'" class="space-y-4">
          <div class="flex justify-end"><button type="button" class="btn btn-secondary" data-test="add-provider" @click="addProvider">{{ t('add') }}</button></div>
          <p v-if="!config.providers.length" class="rounded-xl border border-dashed border-gray-200 p-8 text-center text-gray-500 dark:border-dark-600">{{ t('empty') }}</p>
          <details v-for="(p, index) in config.providers" :key="p.id" class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" :open="expanded === p.id">
            <summary class="cursor-pointer p-4 font-medium text-gray-900 dark:text-gray-100" @click.prevent="expanded = expanded === p.id ? '' : p.id">{{ p.name || `${t('services')} ${index + 1}` }} <span class="ml-2 text-xs font-normal text-gray-500">{{ p.model }} · {{ p.enabled ? t('enabled') : t('disabled') }}</span></summary>
            <div class="space-y-4 border-t border-gray-100 p-4 dark:border-dark-700">
              <label class="flex items-center gap-2 text-sm"><input v-model="p.enabled" type="checkbox" />{{ t('enabled') }}</label>
              <div class="grid gap-4 md:grid-cols-2">
                <label class="field">{{ t('providerName') }}<input v-model="p.name" class="input" maxlength="120" :data-test="`name-${index}`" /></label>
                <label class="field">{{ t('model') }}<input v-model="p.model" class="input" maxlength="256" /></label>
                <label class="field md:col-span-2">{{ t('baseURL') }}<input v-model="p.base_url" class="input" type="url" placeholder="https://provider.example/v1" spellcheck="false" /></label>
              </div>
              <label class="field">{{ t('keys') }}<textarea v-model="keyDrafts[p.id]" class="input font-mono" rows="2" autocomplete="off" spellcheck="false" :disabled="p.clear_keys" :aria-describedby="`key-hint-${p.id}`" /></label>
              <p :id="`key-hint-${p.id}`" class="hint">{{ t('keyHint') }} {{ t('keySaved', { count: p.key_masks?.length || 0 }) }} <span class="font-mono">{{ p.key_masks?.join(' · ') }}</span></p>
              <label class="flex items-center gap-2 text-sm"><input v-model="p.clear_keys" type="checkbox" />{{ t('clearKeys') }}</label>
              <label class="field">{{ t('prompt') }}<textarea v-model="p.audit_prompt" class="input min-h-36" rows="6" spellcheck="false" /></label>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50"><summary class="cursor-pointer text-sm">{{ t('payload') }}</summary><p class="hint my-3">{{ t('payloadHint') }}</p><textarea v-model="p.payload_script" class="input font-mono text-xs" rows="7" :aria-label="t('payload')" spellcheck="false" /></details>
              <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <label class="field">{{ t('threshold') }}<input v-model.number="p.threshold" class="input" type="number" min="0" max="1" step="0.01" /></label>
                <label class="field">{{ t('timeout') }}<input v-model.number="p.timeout_ms" class="input" type="number" min="500" max="30000" step="500" /></label>
                <label class="field">{{ t('concurrency') }}<input v-model.number="p.max_concurrent" class="input" type="number" min="1" max="100" /></label>
                <label class="field">{{ t('inputLimit') }}<input v-model.number="p.max_input_tokens" class="input" type="number" min="256" max="65536" /></label>
                <label class="field">{{ t('outputLimit') }}<input v-model.number="p.max_output_tokens" class="input" type="number" min="32" max="4096" /></label>
                <label class="field">{{ t('outputParameter') }}<select v-model="p.output_parameter" class="input"><option>max_tokens</option><option>max_completion_tokens</option></select></label>
                <label class="field">{{ t('proxy') }}<input :value="p.proxy_id ?? ''" class="input" type="number" min="1" @input="p.proxy_id = ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null" /></label>
              </div>
              <p class="hint">{{ t('estimateHint') }}</p>
              <label class="flex items-start gap-2 text-sm"><input v-model="p.output_limit_verified" type="checkbox" class="mt-1" />{{ t('verified') }}</label>
              <div class="grid gap-4 sm:grid-cols-2">
                <label v-for="f in priceFields" :key="f.key" class="field">{{ t(f.label) }} · {{ config.currency }}<input v-model="p.prices[f.key]" class="input" inputmode="decimal" placeholder="—" /></label>
              </div>
              <p class="hint">{{ t('priceHint') }}</p>
              <h3 class="font-medium">{{ t('providerBudget') }}</h3>
              <AuditLimitFields v-model="p.limits" :labels="limitLabels" />
              <button type="button" class="text-sm text-red-600 hover:underline dark:text-red-400" @click="removeProvider(p.id)">{{ t('remove') }}</button>
            </div>
          </details>
        </div>
        <div v-show="tab === 'policy'" class="space-y-5">
          <label class="flex items-center gap-3 font-medium"><input v-model="config.enabled" type="checkbox" data-test="enable-v2" />{{ t('activate') }}</label>
          <p class="hint">{{ t('legacy') }}</p>
          <div class="grid gap-4 md:grid-cols-2">
            <label class="field">{{ t('primary') }}<select v-model="config.primary_id" class="input"><option value="">{{ t('choose') }}</option><option v-for="p in config.providers" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            <label class="field">{{ t('fallback') }}<select :value="config.fallback_ids[0] || ''" class="input" @change="setFallback(($event.target as HTMLSelectElement).value)"><option value="">{{ t('noFallback') }}</option><option v-for="p in config.providers.filter(p => p.id !== config!.primary_id)" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            <label class="field">{{ t('attempts') }}<select v-model.number="config.max_attempts" class="input"><option :value="1">1</option><option :value="2">2</option></select></label>
            <label class="field">{{ t('cache') }}<input v-model.number="config.cache_ttl_seconds" class="input" type="number" min="0" max="86400" /></label>
            <label class="field md:col-span-2">{{ t('unresolved') }}<select v-model="config.unresolved_policy" class="input" data-test="unresolved-policy"><option value="">{{ t('choose') }}</option><option value="reject_temporary">{{ t('reject') }}</option><option value="allow_record">{{ t('allow') }}</option></select></label>
          </div>
          <p class="hint">{{ t('policyHint') }}</p>
          <p class="rounded-xl bg-primary-50 p-4 text-sm leading-relaxed text-primary-900 dark:bg-primary-950/30 dark:text-primary-200">{{ t('coverage') }}</p>
        </div>
        <div v-show="tab === 'usage'" class="space-y-5">
          <label class="field max-w-xs">{{ t('currency') }}<select v-model="config.currency" class="input" :disabled="config.revision > 0"><option>CNY</option><option>USD</option></select></label>
          <h3 class="font-medium">{{ t('globalBudget') }}</h3><AuditLimitFields v-model="config.limits" :labels="limitLabels" />
          <p class="hint leading-relaxed">{{ t('usageHint') }}</p>
          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-700"><span class="text-sm">{{ t('day') }}: {{ usage?.day || '—' }} UTC</span><button class="btn btn-secondary" type="button" :disabled="busy" @click="refreshUsage">{{ t('refresh') }}</button></div>
          <div v-if="usage" class="grid gap-3 sm:grid-cols-2"><div class="metric">{{ t('cacheHits') }}<strong>{{ usage.cache_hits }}</strong></div><div class="metric">{{ t('unresolvedCount') }}<strong>{{ usage.unresolved }}</strong></div></div>
          <p v-if="usage && !usage.providers.length" class="hint">{{ t('noUsage') }}</p>
          <article v-for="row in usage?.providers || []" :key="row.provider_id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700"><h3 class="mb-3 font-medium">{{ config.providers.find(p => p.id === row.provider_id)?.name || row.provider_id }}</h3><dl class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"><div><dt class="hint">{{ t('calls') }}</dt><dd>{{ row.calls }}</dd></div><div><dt class="hint">{{ t('confirmed') }}</dt><dd>{{ row.confirmed_amount }} {{ usage?.currency }}</dd></div><div><dt class="hint">{{ t('held') }}</dt><dd>{{ row.held_amount }} {{ usage?.currency }}</dd></div><div><dt class="hint">{{ t('unknown') }}</dt><dd>{{ row.unknown_calls }}</dd></div><div class="sm:col-span-2"><dt class="hint">{{ t('tokenUsage') }}</dt><dd>{{ row.input }} / {{ row.cached_input }} / {{ row.output }}</dd></div></dl></article>
        </div>
        <div v-show="tab === 'trial'" class="space-y-4">
          <p class="hint">{{ t('testHint') }}</p><label class="field">{{ t('testText') }}<textarea v-model="testText" class="input" rows="6" data-test="v2-test-text" /></label>
          <div class="flex flex-wrap gap-3"><button type="button" class="btn btn-secondary" :disabled="busy || !testText.trim()" data-test="preview-v2" @click="preview">{{ t('preview') }}</button><button type="button" class="btn btn-primary" :disabled="busy || !testText.trim() || !config.revision" data-test="test-v2" @click="test">{{ t('run') }}</button></div>
          <p v-if="busy" role="status" class="hint">{{ t('pending') }}</p>
          <div v-if="estimate" class="space-y-2 rounded-xl bg-gray-50 p-4 text-sm dark:bg-dark-900/50" role="status"><strong>{{ estimate.fits ? t('fits') : t('tooLong') }}</strong><p>{{ t('estimatedInput') }}: {{ estimate.estimated_input }} · {{ t('outputLimit') }}: {{ estimate.max_output }}</p><p>{{ t('reserve') }}: {{ estimate.reserved_amount || t('unknownAmount') }} {{ config.currency }}</p><p v-if="estimate.reason">{{ reasonLabel(estimate.reason) }}</p><p class="hint">{{ t('estimateHint') }}</p></div>
          <div v-if="result" class="space-y-3 rounded-xl border border-primary-200 p-4 text-sm dark:border-primary-900" role="status" data-test="v2-result">
            <strong>{{ result.status === 'reviewed' ? t('reviewed') : t('unresolvedResult') }}</strong><p>{{ t('actualAttempts') }}: {{ result.attempts }}</p><p v-if="result.reason">{{ reasonLabel(result.reason) }}</p>
            <template v-if="result.verdict"><p class="font-semibold" :class="result.verdict.flagged ? 'text-red-600 dark:text-red-400' : 'text-emerald-700 dark:text-emerald-300'">{{ result.verdict.flagged ? t('hit') : t('pass') }}</p><p>{{ result.verdict.provider_id }} · {{ result.verdict.model }}</p><p v-if="result.verdict.decision_source === 'flagged'">{{ t('boolean') }}</p><p v-else>{{ t('score') }}: {{ result.verdict.score }} / {{ result.verdict.threshold }}</p><p v-if="result.verdict.reason">{{ t('reason') }}: {{ result.verdict.reason }}</p></template>
            <p v-if="result.usage">{{ t('tokenUsage') }}: {{ result.usage.input }} / {{ result.usage.cached_input }} / {{ result.usage.output }}</p>
          </div>
        </div>
      </template>
    </div>
    <template #footer><div class="flex justify-between gap-3"><button class="btn btn-secondary" type="button" :disabled="busy || loading" @click="reload">{{ t('reload') }}</button><button class="btn btn-primary" type="button" data-test="save-v2" :disabled="busy || loading || !config" @click="save">{{ t('save') }}</button></div></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AuditLimitFields from './moderation/AuditLimitFields.vue'
import { moderationV2API, newAuditProvider } from '@/api/admin/moderationV2'
import type { AuditConfig, AuditPreview, AuditResult, AuditUsage, AuditPrices } from '@/api/admin/moderationV2'
import { moderationV2Messages } from '@/i18n/moderationV2'
import { extractApiErrorMessage } from '@/utils/apiError'

const emit = defineEmits<{ close: []; saved: [enabled: boolean] }>()
const { t, te } = useI18n({ useScope: 'local', messages: moderationV2Messages })
const tabs = ['services', 'policy', 'usage', 'trial'] as const
const tab = ref<typeof tabs[number]>('services')
const config = ref<AuditConfig>()
const usage = ref<AuditUsage>()
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const savedJSON = ref('')
const expanded = ref('')
const keyDrafts = ref<Record<string, string>>({})
const testText = ref('')
const result = ref<AuditResult>()
const estimate = ref<AuditPreview>()
const priceFields: { key: keyof AuditPrices; label: string }[] = [
  { key: 'input', label: 'inputPrice' }, { key: 'cached_input', label: 'cachedPrice' },
  { key: 'output', label: 'outputPrice' }, { key: 'per_request', label: 'requestPrice' },
]
const dirty = computed(() => !!config.value && (JSON.stringify(config.value) !== savedJSON.value || Object.values(keyDrafts.value).some(v => v.trim())))
const limitLabels = computed(() => ({ calls: t('dailyCalls'), tokens: t('dailyTokens'), amount: t('dailyAmount') }))
function reasonLabel(reason: string) { return te(`reasons.${reason}`) ? t(`reasons.${reason}`) : reason }
function displayError(e: unknown) {
  const err = e as { status?: number; response?: { status?: number } }
  error.value = err?.status === 409 || err?.response?.status === 409 ? t('conflict') : extractApiErrorMessage(e, t('failure'))
}
async function reload() {
  loading.value = true; error.value = ''; notice.value = ''
  try {
    config.value = await moderationV2API.config(); savedJSON.value = JSON.stringify(config.value)
    keyDrafts.value = {}; expanded.value = config.value.providers[0]?.id || ''; result.value = undefined; estimate.value = undefined
    try { usage.value = await moderationV2API.usage() } catch (e) { displayError(e) }
  } catch (e) { displayError(e) } finally { loading.value = false }
}
function addProvider() { if (!config.value) return; const p = newAuditProvider(); config.value.providers.push(p); expanded.value = p.id }
function removeProvider(id: string) {
  if (!config.value) return
  config.value.providers = config.value.providers.filter(p => p.id !== id)
  config.value.fallback_ids = config.value.fallback_ids.filter(v => v !== id)
  if (config.value.primary_id === id) config.value.primary_id = ''
  delete keyDrafts.value[id]
}
function setFallback(id: string) { if (config.value) config.value.fallback_ids = id ? [id] : [] }
async function save() {
  if (!config.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const draft: AuditConfig = JSON.parse(JSON.stringify(config.value))
    for (const p of draft.providers) {
      delete p.key_masks; delete p.api_keys
      const keys = keyDrafts.value[p.id]?.split(/\r?\n/).map(v => v.trim()).filter(Boolean)
      if (keys?.length && !p.clear_keys) p.api_keys = keys
    }
    config.value = await moderationV2API.save(draft); savedJSON.value = JSON.stringify(config.value)
    keyDrafts.value = {}; notice.value = t('saved'); result.value = undefined; estimate.value = undefined
    emit('saved', config.value.enabled)
  } catch (e) { displayError(e) } finally { busy.value = false }
}
async function refreshUsage() { busy.value = true; error.value = ''; try { usage.value = await moderationV2API.usage() } catch (e) { displayError(e) } finally { busy.value = false } }
async function preview() { busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined; try { estimate.value = await moderationV2API.preview(testText.value) } catch (e) { displayError(e) } finally { busy.value = false } }
async function test() { busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined; try { result.value = await moderationV2API.test(testText.value); usage.value = await moderationV2API.usage() } catch (e) { displayError(e) } finally { busy.value = false } }
onMounted(reload)
</script>

<style scoped>
.field { display: flex; flex-direction: column; gap: .5rem; font-size: .875rem; min-width: 0; }
.field .input { width: 100%; min-width: 0; }
.hint { font-size: .75rem; line-height: 1.7; color: #746d80; overflow-wrap: anywhere; }
:global(.dark) .hint { color: #aaa1b8; }
.metric { border-radius: .75rem; background: rgb(128 103 178 / .07); padding: 1rem; font-size: .875rem; }
.metric strong { display: block; margin-top: .5rem; font-size: 1.5rem; }
</style>
