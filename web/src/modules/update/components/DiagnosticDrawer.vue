<template>
  <t-drawer
    v-model:visible="visible"
    :header="check ? t(check.title_key, check.params ?? {}) : ''"
    :footer="false"
    placement="right"
    size="520px"
    destroy-on-close
  >
    <template v-if="check">
      <p class="diagnostic-drawer__summary">{{ t(check.summary_key, check.params ?? {}) }}</p>
      <p v-if="check.detail_key" class="diagnostic-drawer__detail">{{ t(check.detail_key, check.params ?? {}) }}</p>

      <section v-if="check.evidence.length" class="diagnostic-drawer__section">
        <h3>{{ t('update.center.diagnostics.evidence') }}</h3>
        <dl class="diagnostic-drawer__evidence">
          <div v-for="item in check.evidence" :key="item.code" class="diagnostic-drawer__evidence-row">
            <dt>{{ t(item.label_key) }}</dt>
            <dd>
              <span>{{ item.value || t('update.center.diagnostics.notAvailable') }}</span>
              <small v-if="item.expected">{{
                t('update.center.diagnostics.expected', { value: item.expected })
              }}</small>
            </dd>
          </div>
        </dl>
      </section>

      <section v-if="check.actions.length" class="diagnostic-drawer__section">
        <h3>{{ t('update.center.diagnostics.actions') }}</h3>
        <div class="diagnostic-drawer__actions">
          <t-button v-for="action in check.actions" :key="action.id" variant="outline" @click="emit('action', action)">
            {{ t(action.label_key, action.params ?? {}) }}
          </t-button>
        </div>
      </section>
    </template>
  </t-drawer>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { UpdateReadinessAction, UpdateReadinessCheck } from '../types/update';

defineProps<{ check: UpdateReadinessCheck | null }>();
const visible = defineModel<boolean>('visible', { required: true });
const emit = defineEmits<{ action: [action: UpdateReadinessAction] }>();
const { t } = useI18n();
</script>
<style scoped lang="less">
.diagnostic-drawer__summary,
.diagnostic-drawer__detail {
  margin: 0;
}

.diagnostic-drawer__summary {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.diagnostic-drawer__detail {
  color: var(--td-text-color-secondary);
  margin-top: var(--td-comp-margin-s);
}

.diagnostic-drawer__section {
  margin-top: var(--td-comp-margin-xl);
}

.diagnostic-drawer__section h3 {
  font: var(--td-font-title-small);
  margin: 0 0 var(--td-comp-margin-s);
}

.diagnostic-drawer__evidence {
  display: grid;
  gap: var(--td-comp-margin-s);
  margin: 0;
}

.diagnostic-drawer__evidence-row {
  border-bottom: 1px solid var(--td-component-border);
  display: grid;
  gap: var(--td-comp-margin-xs);
  padding-bottom: var(--td-comp-margin-s);
}

.diagnostic-drawer__evidence-row dt,
.diagnostic-drawer__evidence-row small {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.diagnostic-drawer__evidence-row dd {
  display: grid;
  gap: var(--td-comp-margin-xs);
  margin: 0;
  overflow-wrap: anywhere;
}

.diagnostic-drawer__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}
</style>
