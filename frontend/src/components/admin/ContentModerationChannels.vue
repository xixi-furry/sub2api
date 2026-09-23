<template>
 <div class="audit-channels space-y-4">
  <p v-if="error" class="rounded-xl bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ error }}</p>
<div v-if="section === 'prompt'" class="space-y-5">
  <p class="hint">{{ t('moderationV2.channelPromptHint') }}</p><p v-if="policy.mode !== 'legacy'" class="hint">{{ t('moderationV2.contractHint') }}</p>
  <article v-for="p in config.providers" :key="p.id" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
    <h3 class="font-medium">{{ p.name || p.id }} · {{ p.model }}</h3>
              <label class="field">{{ t('moderationV2.prompt') }}<textarea v-model="p.audit_prompt" class="input min-h-36" rows="6" spellcheck="false" /></label>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50"><summary class="min-h-8 cursor-pointer text-sm">{{ t('moderationV2.payload') }}</summary><p class="hint my-3">{{ t('moderationV2.payloadHint') }}</p><textarea v-model="p.payload_script" class="input font-mono text-xs" rows="7" :aria-label="t('moderationV2.payload')" spellcheck="false" /></details>
  </article>
</div>
        <div v-if="section === 'services'" class="space-y-4">
          <div class="flex justify-end"><button type="button" class="btn btn-secondary" data-test="add-provider" @click="addProvider">{{ t('moderationV2.add') }}</button></div>
          <p v-if="!config.providers.length" class="rounded-xl border border-dashed border-gray-200 p-8 text-center text-gray-500 dark:border-dark-600">{{ t('moderationV2.empty') }}</p>
          <details v-for="(p, index) in config.providers" :key="p.id" class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" :open="expanded === p.id">
            <summary class="min-h-8 cursor-pointer p-4 font-medium text-gray-900 dark:text-gray-100" @click.prevent="expanded = expanded === p.id ? '' : p.id">{{ p.name || `${t('moderationV2.services')} ${index + 1}` }} <span class="ml-2 text-xs font-normal text-gray-500">{{ p.model }} · {{ p.enabled ? t('moderationV2.enabled') : t('moderationV2.disabled') }}</span></summary>
            <div class="space-y-4 border-t border-gray-100 p-4 dark:border-dark-700">
              <label class="flex items-center gap-2 text-sm"><input v-model="p.enabled" type="checkbox" />{{ t('moderationV2.enabled') }}</label>
              <div class="grid gap-4 md:grid-cols-2">
                <label class="field">{{ t('moderationV2.providerName') }}<input v-model="p.name" class="input" maxlength="120" :data-test="`name-${index}`" /></label>
                <label class="field">{{ t('moderationV2.model') }}<input v-model="p.model" class="input" maxlength="256" /></label>
                <label class="field md:col-span-2">{{ t('moderationV2.baseURL') }}<input v-model="p.base_url" class="input" type="url" placeholder="https://provider.example/v1" spellcheck="false" /></label>
              </div>
              <label class="field">{{ t('moderationV2.keys') }}<textarea v-model="p.key_draft" class="input font-mono" rows="2" autocomplete="off" spellcheck="false" :disabled="p.clear_keys" :aria-describedby="`key-hint-${p.id}`" /></label>
              <p :id="`key-hint-${p.id}`" class="hint">{{ t('moderationV2.keyHint') }} {{ t('moderationV2.keySaved', { count: p.key_masks?.length || 0 }) }} <span class="font-mono">{{ p.key_masks?.join(' · ') }}</span></p>
              <div class="grid gap-4 md:grid-cols-2">
                <label class="field">{{ t('moderationV2.purpose') }}<select v-model="p.purpose" class="input" data-test="provider-purpose"><option value="primary">{{ t('moderationV2.primaryPurpose') }}</option><option value="review">{{ t('moderationV2.reviewPurpose') }}</option><option value="both">{{ t('moderationV2.bothPurpose') }}</option></select></label>
                <label class="field">{{ t('moderationV2.threshold') }}<input v-model.number="p.threshold" class="input" type="number" min="0" max="1" step="0.01" /></label>
              </div>
              <label class="field">{{ t('moderationV2.prompt') }}<textarea v-model="p.audit_prompt" class="input min-h-28" rows="4" spellcheck="false" data-test="channel-prompt" /></label>
              <p class="hint">{{ t('moderationV2.promptSimpleHint') }}</p>
              <label class="flex items-center gap-2 text-sm"><input v-model="p.audit_validated" type="checkbox" />{{ t('moderationV2.auditValidated') }}</label>
              <p class="hint">{{ t('moderationV2.auditValidatedHint') }}</p>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50" :open="config.routing === 'lowest_cost'" data-test="provider-pricing">
                <summary class="min-h-8 cursor-pointer text-sm font-medium">{{ t('moderationV2.pricingSettings') }}</summary>
                <div class="mt-4 space-y-4">
              <AuditPriceLookup v-model:usd-rate="usdRate" :model="p.model" :base-url="p.base_url" :currency="config.currency" @apply="prices => Object.assign(p.prices, prices)" />
              <div class="grid gap-4 sm:grid-cols-2">
                <label v-for="f in priceFields" :key="f.key" class="field">{{ t(`moderationV2.${f.label}`) }} · {{ config.currency }}<input v-model="p.prices[f.key]" class="input" inputmode="decimal" placeholder="—" /></label>
              </div>
              <p class="hint">{{ t('moderationV2.priceHint') }}</p>
              <label class="flex items-start gap-2 text-sm"><input v-model="p.output_limit_verified" type="checkbox" class="mt-1" />{{ t('moderationV2.verified') }}</label>
                </div>
              </details>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50" data-test="provider-advanced">
                <summary class="min-h-8 cursor-pointer text-sm font-medium">{{ t('moderationV2.providerAdvanced') }} · {{ p.api_format === 'responses' ? 'Responses' : 'Chat Completions' }}</summary>
                <div class="mt-4 space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
                <label class="field">{{ t('moderationV2.protocol') }}<select v-model="p.api_format" class="input" data-test="provider-protocol" @change="p.reasoning_parameter = 'none'"><option value="chat_completions">Chat Completions</option><option value="responses">Responses</option></select></label>
              </div>
              <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50">
                <summary class="min-h-8 cursor-pointer text-sm">{{ t('moderationV2.generationSettings') }}</summary>
                <div class="mt-4 grid gap-4 md:grid-cols-2">
                  <label class="flex items-center gap-2 text-sm md:col-span-2"><input v-model="p.stream" type="checkbox" />{{ t('moderationV2.stream') }}</label>
                  <label class="field">{{ t('moderationV2.reasoningParameter') }}<select v-model="p.reasoning_parameter" class="input"><option value="none">{{ t('moderationV2.providerDefault') }}</option><option value="effort">reasoning effort</option><option v-if="p.api_format !== 'responses'" value="thinking">thinking.type</option><option v-if="p.api_format !== 'responses'" value="enable_thinking">enable_thinking</option></select></label>
                  <label v-if="p.reasoning_parameter === 'effort'" class="field">{{ t('moderationV2.reasoningEffort') }}<select v-model="p.reasoning_effort" class="input"><option v-for="value in ['none', 'minimal', 'low', 'medium', 'high', 'xhigh']" :key="value">{{ value }}</option></select></label>
                  <label v-if="p.reasoning_parameter === 'thinking' || p.reasoning_parameter === 'enable_thinking'" class="flex items-center gap-2 text-sm"><input v-model="p.thinking_enabled" type="checkbox" />{{ t('moderationV2.thinking') }}</label>
                  <label class="field">{{ t('moderationV2.headerTimeout') }}<input v-model.number="p.header_timeout_ms" class="input" type="number" min="0" :max="p.timeout_ms" step="1000" /></label>
                  <label class="field">{{ t('moderationV2.idleTimeout') }}<input v-model.number="p.idle_timeout_ms" class="input" type="number" min="0" :max="p.timeout_ms" step="1000" /></label>
                </div><p class="hint mt-3">{{ t('moderationV2.generationHint') }}</p>
              </details>
              <label class="flex items-center gap-2 text-sm"><input v-model="p.clear_keys" type="checkbox" />{{ t('moderationV2.clearKeys') }}</label>
              <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <label class="field">{{ t('moderationV2.timeout') }}<input v-model.number="p.timeout_ms" class="input" type="number" min="500" max="180000" step="500" /></label>
                <label class="field">{{ t('moderationV2.concurrency') }}<input v-model.number="p.max_concurrent" class="input" type="number" min="1" max="100" /></label>
                <label class="field">{{ t('moderationV2.inputLimit') }}<input v-model.number="p.max_input_tokens" class="input" type="number" min="256" max="262144" /></label>
                <label class="field">{{ t('moderationV2.outputLimit') }}<input v-model.number="p.max_output_tokens" class="input" type="number" min="32" max="32768" /></label>
                <label v-if="p.api_format !== 'responses'" class="field">{{ t('moderationV2.outputParameter') }}<select v-model="p.output_parameter" class="input"><option>max_tokens</option><option>max_completion_tokens</option></select></label>
                <label class="field">{{ t('moderationV2.proxy') }}<input :value="p.proxy_id ?? ''" class="input" type="number" min="1" @input="p.proxy_id = ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null" /></label>
              </div>
              <p class="hint">{{ t('moderationV2.estimateHint') }}</p>
              <h3 class="font-medium">{{ t('moderationV2.providerBudget') }}</h3>
              <AuditLimitFields v-model="p.limits" :labels="limitLabels" />
                  <details><summary class="min-h-8 cursor-pointer text-sm">{{ t('moderationV2.payload') }}</summary><p class="hint my-3">{{ t('moderationV2.payloadHint') }}</p><textarea v-model="p.payload_script" class="input font-mono text-xs" rows="7" :aria-label="t('moderationV2.payload')" spellcheck="false" /></details>
                </div>
              </details>
              <button type="button" class="text-sm text-red-600 hover:underline dark:text-red-400" @click="removeProvider(p.id)">{{ t('moderationV2.remove') }}</button>
            </div>
          </details>
        </div>
        <div v-if="section === 'policy'" class="space-y-4" data-test="audit-policy">
          <div class="grid gap-4 md:grid-cols-2">
            <label class="field">{{ t('moderationV2.auditMode') }}<select v-model="policy.mode" class="input" data-test="audit-mode" @change="onAuditModeChange"><option value="legacy">{{ t('moderationV2.legacyMode') }}</option><option value="balanced">{{ t('moderationV2.balancedMode') }}</option><option value="quality">{{ t('moderationV2.qualityMode') }}</option></select></label>
            <label v-if="policy.mode !== 'legacy'" class="field">{{ t('moderationV2.contextMode') }}<select v-model="policy.context_mode" class="input" data-test="context-mode"><option value="bounded">{{ t('moderationV2.boundedContext') }}</option><option value="full">{{ t('moderationV2.fullContext') }}</option><option value="current">{{ t('moderationV2.currentContext') }}</option></select></label>
          </div>
          <p class="hint">{{ t(`moderationV2.${policy.mode}ModeShort`) }}</p>
          <label class="field">{{ t('moderationV2.routing') }}<select v-model="config.routing" class="input" data-test="channel-routing"><option value="lowest_cost">{{ t('moderationV2.cheapest') }}</option><option value="priority">{{ t('moderationV2.priority') }}</option></select></label>
          <p class="hint">{{ t('moderationV2.routingShort') }}</p>
          <div class="grid gap-4 md:grid-cols-2">
            <label v-if="config.routing !== 'lowest_cost' && policy.mode !== 'quality'" class="field">{{ t('moderationV2.primary') }}<select v-model="config.primary_id" class="input"><option value="">{{ t('moderationV2.choose') }}</option><option v-for="p in config.providers" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            <label v-if="config.routing !== 'lowest_cost' && policy.mode !== 'quality'" class="field">{{ t('moderationV2.fallback') }}<select :value="config.fallback_ids[0] || ''" class="input" @change="setFallback(($event.target as HTMLSelectElement).value)"><option value="">{{ t('moderationV2.noFallback') }}</option><option v-for="p in config.providers.filter(p => p.id !== config!.primary_id)" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
          </div>
          <template v-if="policy.mode !== 'legacy'">
            <div v-if="config.routing !== 'lowest_cost'" class="grid gap-4 md:grid-cols-2">
              <label v-for="index in [0, 1]" :key="index" class="field">{{ t('moderationV2.reviewChannel', { index: index + 1 }) }}<select :value="policy.review_ids[index] || ''" class="input" @change="setReview(index, ($event.target as HTMLSelectElement).value)"><option value="">{{ t('moderationV2.choose') }}</option><option v-for="p in config.providers.filter(p => p.purpose === 'review' || p.purpose === 'both')" :key="p.id" :value="p.id">{{ p.name || p.id }}</option></select></label>
            </div>
          </template>
          <div class="grid gap-4 md:grid-cols-2">
            <label class="field md:col-span-2">{{ t('moderationV2.unresolved') }}<select v-model="config.unresolved_policy" class="input" data-test="unresolved-policy"><option value="">{{ t('moderationV2.choose') }}</option><option value="reject_temporary">{{ t('moderationV2.reject') }}</option><option v-if="policy.mode === 'legacy'" value="allow_record">{{ t('moderationV2.allow') }}</option></select></label>
          </div>
          <details class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50" data-test="policy-advanced">
            <summary class="min-h-8 cursor-pointer text-sm font-medium">{{ t('moderationV2.policyAdvanced') }}</summary>
            <div class="mt-4 space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
            <label class="field">{{ t('moderationV2.attempts') }}<select v-model.number="config.max_attempts" class="input"><option :value="1">1</option><option :value="2">2</option></select></label>
            <label class="field">{{ t('moderationV2.cache') }}<input v-model.number="config.cache_ttl_seconds" class="input" type="number" min="0" max="86400" /></label>
              </div>
              <template v-if="policy.mode !== 'legacy'">
            <div><div class="mt-4 grid gap-4 md:grid-cols-2">
              <label class="field">{{ t('moderationV2.historyMessages') }}<input v-model.number="policy.history_messages" class="input" type="number" min="0" max="256" /></label>
              <label class="field">{{ t('moderationV2.historyBytes') }}<input v-model.number="policy.history_bytes" class="input" type="number" min="0" max="262144" /></label>
              <label class="field">{{ t('moderationV2.reviewHistoryMessages') }}<input v-model.number="policy.review_history_messages" class="input" type="number" :min="policy.history_messages" max="256" /></label>
              <label class="field">{{ t('moderationV2.reviewHistoryBytes') }}<input v-model.number="policy.review_history_bytes" class="input" type="number" :min="policy.history_bytes" max="1048576" /></label>
              <label class="field">{{ t('moderationV2.uncertaintyMargin') }}<input v-model.number="policy.uncertainty_margin" class="input" type="number" min="0" max="0.5" step="0.01" /></label>
            </div></div>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="field">{{ t('moderationV2.requestTimeout') }}<input v-model.number="policy.request_timeout_ms" class="input" type="number" min="1000" max="240000" step="1000" /></label>
              <label class="field">{{ t('moderationV2.requestAmount') }} · {{ config.currency }}<input v-model="policy.max_request_amount" class="input" inputmode="decimal" /></label>
            </div>
            <p class="hint">{{ t('moderationV2.sharedBudgetHint') }}</p>
                <p class="hint">{{ t('moderationV2.contextHint') }}</p>
                <p class="hint">{{ t('moderationV2.contractHint') }}</p>
              </template>
              <p class="hint">{{ t('moderationV2.routingHint') }}</p>
              <p class="hint">{{ t('moderationV2.policyHint') }}</p>
            </div>
          </details>
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
          <div v-if="needsSave" class="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-primary-50 p-4 text-sm text-primary-900 dark:bg-primary-950/30 dark:text-primary-200" role="status" data-test="trial-needs-save">
            <p>{{ t('moderationV2.saveBeforeTrial') }}</p>
            <button type="button" class="btn btn-secondary" :disabled="saving || busy" data-test="save-before-trial" @click="emit('save')">{{ t(saving ? 'moderationV2.savingForTrial' : 'moderationV2.saveForTrial') }}</button>
          </div>
          <p class="hint">{{ t('moderationV2.testHint') }}</p>
          <label class="field">{{ t('moderationV2.testFormat') }}<select v-model="testProtocol" class="input"><option value="">{{ t('moderationV2.plainText') }}</option><option value="openai_chat_completions">Chat Completions JSON</option><option value="openai_responses">Responses JSON</option><option value="anthropic_messages">Anthropic Messages JSON</option><option value="gemini">Gemini JSON</option></select></label><label class="field">{{ t('moderationV2.testText') }}<textarea v-model="testText" class="input" rows="6" data-test="v2-test-text" /></label>
          <div class="flex flex-wrap gap-3"><button type="button" class="btn btn-secondary" :disabled="!canRun || !testText.trim()" data-test="preview-v2" @click="preview">{{ t('moderationV2.preview') }}</button><button type="button" class="btn btn-primary" :disabled="!canRun || !testText.trim()" data-test="test-v2" @click="test">{{ t('moderationV2.run') }}</button></div>
          <p v-if="busy" role="status" class="hint">{{ t('moderationV2.pending') }}</p>
          <div v-if="estimate" class="space-y-2 rounded-xl bg-gray-50 p-4 text-sm dark:bg-dark-900/50" role="status"><strong>{{ estimate.fits ? t('moderationV2.fits') : t('moderationV2.tooLong') }}</strong><p>{{ t('moderationV2.selectedChannel') }}: {{ config.providers.find(p => p.id === estimate!.provider_id)?.name || estimate.provider_id || '—' }}</p><p>{{ t('moderationV2.estimatedInput') }}: {{ estimate.estimated_input }} · {{ t('moderationV2.outputLimit') }}: {{ estimate.max_output }}</p><p>{{ t('moderationV2.reserve') }}: {{ estimate.reserved_amount || t('moderationV2.unknownAmount') }} {{ config.currency }}</p><p v-if="estimate.reason">{{ reasonLabel(estimate.reason) }}</p><p class="hint">{{ t('moderationV2.estimateHint') }}</p></div>
          <div v-if="result" class="space-y-3 rounded-xl border border-primary-200 p-4 text-sm dark:border-primary-900" role="status" data-test="v2-result">
            <strong>{{ result.status === 'reviewed' ? t('moderationV2.reviewed') : t('moderationV2.unresolvedResult') }}</strong><p>{{ t('moderationV2.actualAttempts') }}: {{ result.attempts }}</p><p v-if="result.reason">{{ reasonLabel(result.reason) }}</p>
            <template v-if="result.verdict"><p class="font-semibold" :class="result.verdict.flagged ? 'text-red-600 dark:text-red-400' : 'text-emerald-700 dark:text-emerald-300'">{{ result.verdict.flagged ? t('moderationV2.hit') : t('moderationV2.pass') }}</p><p>{{ result.verdict.provider_id }} · {{ result.verdict.model }}</p><p v-if="result.verdict.decision_source === 'flagged'">{{ t('moderationV2.boolean') }}</p><p v-else>{{ t('moderationV2.score') }}: {{ result.verdict.score }} / {{ result.verdict.threshold }}</p><p v-if="result.verdict.reason">{{ t('moderationV2.reason') }}: {{ result.verdict.reason }}</p></template>
            <p v-if="result.total_ms !== undefined">{{ t('moderationV2.totalTime') }}: {{ result.total_ms }} ms</p>
            <article v-for="(trace, index) in result.traces || []" :key="index" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-900/50"><p>{{ trace.stage === 'review' ? t('moderationV2.reviewPurpose') : t('moderationV2.primaryPurpose') }} · {{ trace.provider_id }}</p><p>{{ t('moderationV2.responseWait') }}: {{ trace.header_ms }} ms · {{ t('moderationV2.firstText') }}: {{ trace.first_text_ms === undefined ? '—' : `${trace.first_text_ms} ms` }} · {{ t('moderationV2.totalTime') }}: {{ trace.total_ms }} ms</p><p v-if="trace.reason">{{ reasonLabel(trace.reason) }}</p><p>{{ t('moderationV2.reserve') }}: {{ trace.reserved_amount || t('moderationV2.unknownAmount') }} {{ config.currency }}</p></article>
            <p v-if="result.usage">{{ t('moderationV2.tokenUsage') }}: {{ result.usage.input }} / {{ result.usage.cached_input }} / {{ result.usage.output }}</p>
          </div>
          <div v-if="estimate?.coverage || result?.coverage" class="rounded-xl bg-primary-50 p-4 text-sm dark:bg-primary-950/30" data-test="context-coverage"><p>{{ t('moderationV2.coverageSummary', { selected: (estimate?.coverage || result?.coverage)?.selected_messages, omitted: (estimate?.coverage || result?.coverage)?.omitted_messages, bytes: (estimate?.coverage || result?.coverage)?.selected_bytes }) }}</p><p class="hint">{{ t('moderationV2.coverageHint') }}</p></div>
          <details v-if="estimate?.fragments?.length"><summary class="min-h-8 cursor-pointer text-sm">{{ t('moderationV2.evidencePreview') }}</summary><article v-for="f in estimate.fragments" :key="f.id" class="mt-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700"><p class="hint">{{ f.id }} · {{ f.role }} · {{ f.kind }} · {{ f.current ? t('moderationV2.currentContext') : t('moderationV2.history') }}</p><pre class="mt-2 whitespace-pre-wrap break-words text-xs">{{ f.text }}</pre></article></details>
        </div>
 </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AuditLimitFields from './moderation/AuditLimitFields.vue'
import AuditPriceLookup from './moderation/AuditPriceLookup.vue'
import { moderationV2API, newAuditProvider, normalizeAuditDraft } from '@/api/admin/moderationV2'
import type { AuditConfig, AuditPreview, AuditResult, AuditUsage, AuditPrices } from '@/api/admin/moderationV2'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = withDefaults(defineProps<{
  section: 'services' | 'policy' | 'usage' | 'trial' | 'prompt'
  configSaved?: boolean
  saving?: boolean
}>(), { configSaved: true, saving: false })
const emit = defineEmits<{ save: [] }>()
const config = defineModel<AuditConfig>({ required: true })
normalizeAuditDraft(config.value)
const needsSave = computed(() => !props.configSaved || !config.value.revision)
const canRun = computed(() => !needsSave.value && !props.saving && !busy.value)
const policy = computed(() => config.value.policy!)
const { t, te } = useI18n()
const usage = ref<AuditUsage>()
const busy = ref(false)
const error = ref('')
const expanded = ref(config.value.providers[0]?.id || '')
const usdRate = ref('')
const testText = ref('')
const testProtocol = ref('')
function testInput() { return testProtocol.value ? { protocol: testProtocol.value, body: JSON.parse(testText.value) } : testText.value }
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
  policy.value.review_ids = policy.value.review_ids.filter(v => v !== id)
  if (config.value.primary_id === id) config.value.primary_id = ''
}
function onAuditModeChange() {
  if (policy.value.mode !== 'legacy') config.value.unresolved_policy = 'reject_temporary'
}
function setReview(index: number, id: string) { const ids = [...policy.value.review_ids]; ids[index] = id; policy.value.review_ids = [...new Set(ids.filter(Boolean))] }
function setFallback(id: string) { config.value.fallback_ids = id ? [id] : [] }
async function refreshUsage() { busy.value = true; error.value = ''; try { usage.value = await moderationV2API.usage() } catch (e) { displayError(e) } finally { busy.value = false } }
async function preview() {
  if (!canRun.value) return
  const snapshot = JSON.stringify(config.value)
  busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined
  try {
    const response = await moderationV2API.preview(testInput())
    if (props.configSaved && snapshot === JSON.stringify(config.value)) estimate.value = response
  } catch (e) { displayError(e) } finally { busy.value = false }
}
async function test() {
  if (!canRun.value) return
  const snapshot = JSON.stringify(config.value)
  busy.value = true; error.value = ''; result.value = undefined; estimate.value = undefined
  try {
    const response = await moderationV2API.test(testInput())
    if (props.configSaved && snapshot === JSON.stringify(config.value)) result.value = response
  } catch (e) { displayError(e) } finally { busy.value = false }
}
watch(() => [props.configSaved, config.value.revision], () => { result.value = undefined; estimate.value = undefined })
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
