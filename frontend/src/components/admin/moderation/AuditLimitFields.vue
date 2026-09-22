<template>
  <div class="grid gap-4 sm:grid-cols-3">
    <label class="flex min-w-0 flex-col gap-2 text-sm">{{ labels.calls }}<input :value="modelValue.daily_calls" class="input w-full" type="number" min="0" max="1000000000" @input="update('daily_calls', Number(($event.target as HTMLInputElement).value))" /></label>
    <label class="flex min-w-0 flex-col gap-2 text-sm">{{ labels.tokens }}<input :value="modelValue.daily_tokens" class="input w-full" type="number" min="0" max="1000000000000" @input="update('daily_tokens', Number(($event.target as HTMLInputElement).value))" /></label>
    <label class="flex min-w-0 flex-col gap-2 text-sm">{{ labels.amount }}<input :value="modelValue.daily_amount" class="input w-full" inputmode="decimal" placeholder="—" @input="update('daily_amount', ($event.target as HTMLInputElement).value)" /></label>
  </div>
</template>
<script setup lang="ts">
import type { AuditLimits } from '@/api/admin/moderationV2'
const props = defineProps<{ modelValue: AuditLimits; labels: { calls: string; tokens: string; amount: string } }>()
const emit = defineEmits<{ 'update:modelValue': [value: AuditLimits] }>()
function update<K extends keyof AuditLimits>(key: K, value: AuditLimits[K]) { emit('update:modelValue', { ...props.modelValue, [key]: value }) }
</script>
