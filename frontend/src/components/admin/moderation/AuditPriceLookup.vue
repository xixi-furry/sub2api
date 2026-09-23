<template>
  <div class="space-y-3" data-test="price-lookup">
    <button type="button" class="btn btn-secondary" :aria-expanded="opened" data-test="lookup-prices" @click="openLookup">{{ t('moderationV2.catalog.lookup') }}</button>
    <div v-if="opened" class="space-y-4 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-end gap-2">
        <label class="min-w-0 flex-1 text-sm">{{ t('moderationV2.catalog.modelSearch') }}<input v-model="query" class="input mt-2 w-full" maxlength="256" data-test="price-query" @keydown.enter.prevent="search" /></label>
        <button type="button" class="btn btn-secondary" :disabled="loading || query.trim().length < 2" data-test="search-prices" @click="search">{{ t(loading ? 'moderationV2.catalog.loading' : 'moderationV2.catalog.search') }}</button>
      </div>
      <p v-if="error" class="text-sm text-red-700 dark:text-red-300" role="alert">{{ error }}</p>
      <template v-if="result">
        <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.source') }} <a href="https://models.dev" target="_blank" rel="noopener noreferrer" class="underline">models.dev</a> · {{ t('moderationV2.catalog.updated', { time: new Date(result.fetched_at).toLocaleString() }) }}</p>
        <p v-if="result.stale" role="status" class="text-sm text-amber-700 dark:text-amber-300">{{ t('moderationV2.catalog.stale') }}</p>
        <p v-if="!result.items.length" class="text-sm text-gray-600 dark:text-gray-300">{{ t('moderationV2.catalog.empty') }}</p>
        <template v-else>
          <label class="block text-sm">{{ t('moderationV2.catalog.entry') }}<select v-model="selectedID" class="input mt-2 w-full" data-test="price-entry"><option value="">{{ t('moderationV2.choose') }}</option><option v-for="item in result.items" :key="entryKey(item)" :value="entryKey(item)">{{ item.matches_channel ? '✓ ' : '' }}{{ item.provider_name }} / {{ item.model_id }}</option></select></label>
          <p v-if="result.total > result.items.length" class="text-xs text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.truncated', { count: result.items.length, total: result.total }) }}</p>
          <div v-if="selected" class="space-y-3">
            <p class="break-words text-sm font-medium">{{ selected.provider_name }} · {{ selected.model_id }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.unit') }}</p>
            <dl class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
              <div><dt class="text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.input') }}</dt><dd class="tabular-nums">{{ selected.prices.input || '—' }}</dd></div>
              <div><dt class="text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.output') }}</dt><dd class="tabular-nums">{{ selected.prices.output || '—' }}</dd></div>
              <div><dt class="text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.cached') }}</dt><dd class="tabular-nums">{{ selected.prices.cached_input || '—' }}</dd></div>
            </dl>
            <p class="text-xs leading-5 text-gray-600 dark:text-gray-300">{{ t('moderationV2.catalog.qualityHint') }}</p>
            <p v-if="!selected.importable" class="text-sm text-amber-700 dark:text-amber-300">{{ t(`moderationV2.catalog.reasons.${selected.manual_reason || 'special_price'}`) }}</p>
            <template v-else>
              <p v-if="selected.prices.input === '0' && selected.prices.output === '0'" class="text-xs text-amber-700 dark:text-amber-300">{{ t('moderationV2.catalog.zeroHint') }}</p>
              <label v-if="needsConfirmation" class="flex items-start gap-2 text-sm"><input v-model="confirmedReference" type="checkbox" class="mt-1" data-test="confirm-reference" />{{ t('moderationV2.catalog.referenceHint') }}</label>
              <label v-if="currency === 'CNY'" class="block text-sm">{{ t('moderationV2.catalog.exchange') }}<input v-model="usdRate" class="input mt-2 w-full" inputmode="decimal" placeholder="1 USD = ? CNY" data-test="usd-rate" /><span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('moderationV2.catalog.exchangeHint') }}</span></label>
              <p v-if="converted" class="text-sm tabular-nums" data-test="converted-prices">{{ t('moderationV2.catalog.converted', { currency, input: converted.input, output: converted.output }) }}</p>
              <button type="button" class="btn btn-secondary" :disabled="!converted || (needsConfirmation && !confirmedReference)" data-test="apply-prices" @click="apply">{{ t('moderationV2.catalog.apply') }}</button>
              <p v-if="applied" role="status" class="text-sm text-primary-700 dark:text-primary-300">{{ t('moderationV2.catalog.applied') }}</p>
            </template>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { moderationV2API, type AuditModelPrice, type AuditModelPrices, type AuditPrices } from '@/api/admin/moderationV2'
import { convertAuditPrice } from '@/utils/moderationPrices'

const props = defineProps<{ model: string; baseUrl: string; currency: 'USD' | 'CNY' }>()
const usdRate = defineModel<string>('usdRate', { default: '' })
const emit = defineEmits<{ apply: [prices: Pick<AuditPrices, 'input' | 'output' | 'cached_input'>] }>()
const { t } = useI18n()
const opened = ref(false), loading = ref(false), applied = ref(false), confirmedReference = ref(false)
const query = ref(props.model), error = ref(''), selectedID = ref('')
const result = ref<AuditModelPrices>()
let generation = 0
const entryKey = (entry: AuditModelPrice) => JSON.stringify([entry.provider_id, entry.model_id])
const selected = computed(() => result.value?.items.find(item => entryKey(item) === selectedID.value))
const needsConfirmation = computed(() => selected.value && (!selected.value.matches_channel || selected.value.model_id.toLowerCase() !== props.model.trim().toLowerCase()))
const converted = computed(() => {
  if (!selected.value?.importable) return undefined
  const rate = props.currency === 'USD' ? '1' : usdRate.value.trim()
  try {
    return {
      input: convertAuditPrice(selected.value.prices.input, rate),
      output: convertAuditPrice(selected.value.prices.output, rate),
      cached_input: selected.value.prices.cached_input === '' ? '' : convertAuditPrice(selected.value.prices.cached_input, rate),
    }
  } catch { return undefined }
})
async function openLookup() {
  opened.value = !opened.value
  if (opened.value && !result.value && query.value.trim().length >= 2) await search()
}
async function search() {
  if (loading.value || query.value.trim().length < 2) return
  const request = ++generation
  loading.value = true; error.value = ''; result.value = undefined; applied.value = false; selectedID.value = ''
  try {
    const response = await moderationV2API.modelPrices(query.value.trim(), props.baseUrl)
    if (request !== generation) return
    result.value = response
    const exact = response.items.filter(item => item.matches_channel && item.model_id.toLowerCase() === props.model.trim().toLowerCase())
    if (exact.length === 1) selectedID.value = entryKey(exact[0]!)
  } catch { if (request === generation) error.value = t('moderationV2.catalog.failure') }
  finally { if (request === generation) loading.value = false }
}
function apply() {
  if (!converted.value || (needsConfirmation.value && !confirmedReference.value)) return
  emit('apply', converted.value)
  applied.value = true
}
watch(selectedID, () => { confirmedReference.value = false; applied.value = false })
watch(() => [props.currency, usdRate.value], () => { applied.value = false })
watch(() => [props.model, props.baseUrl], () => {
  generation++; result.value = undefined; selectedID.value = ''; loading.value = false; applied.value = false; query.value = props.model
})
</script>
