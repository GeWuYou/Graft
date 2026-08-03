<template>
  <section class="outbound-network-page" data-page-type="settings">
    <page-header
      :source="{
        labelKey: 'network.outbound.eyebrow',
        fallback: t('network.outbound.eyebrow'),
        color: 'var(--td-brand-color-6)',
      }"
      title-key="network.outbound.title"
      :title-fallback="t('network.outbound.title')"
      description-key="network.outbound.description"
      :description-fallback="t('network.outbound.description')"
    >
      <template #extra>
        <t-tag :theme="dirty ? 'warning' : 'success'" variant="light">
          {{ dirty ? t('network.outbound.unsaved') : t('network.outbound.saved') }}
        </t-tag>
      </template>
      <template #actions>
        <t-button
          theme="default"
          variant="outline"
          :disabled="policySource === 'default'"
          :loading="resetting"
          @click="resetToDefault"
        >
          {{ t('network.outbound.resetToDefault') }}
        </t-button>
        <t-button theme="primary" :disabled="!dirty" :loading="saving" @click="save">
          {{ t('network.outbound.save') }}
        </t-button>
      </template>
    </page-header>

    <t-alert class="outbound-network-page__scope" theme="info">
      <template #message>
        <div class="outbound-network-page__scope-content">
          <div>
            <strong>{{ t('network.outbound.scope.title') }}</strong>
            <span>{{ t('network.outbound.scope.description') }}</span>
          </div>
          <div class="outbound-network-page__scope-items">
            <t-tag theme="success" variant="light">{{ t('network.outbound.scope.platformHttp') }}</t-tag>
            <t-tag theme="success" variant="light">{{ t('network.outbound.scope.registeredConsumers') }}</t-tag>
            <t-tag theme="default" variant="light">{{ t('network.outbound.scope.docker') }}</t-tag>
            <t-tag theme="default" variant="light">{{ t('network.outbound.scope.browser') }}</t-tag>
          </div>
        </div>
      </template>
    </t-alert>

    <t-alert
      v-if="errorMessage"
      class="outbound-network-page__alert"
      theme="error"
      :title="t('network.outbound.loadFailed')"
      :message="errorMessage"
    />
    <div v-if="preconditionMessage" class="outbound-network-page__precondition-alert">
      <t-alert theme="warning" :title="t('network.outbound.precondition.title')" :message="preconditionMessage" />
      <t-button theme="warning" variant="outline" :loading="loading" @click="reloadLatestPolicy">
        {{ t('network.outbound.precondition.reload') }}
      </t-button>
    </div>

    <t-loading :loading="loading" class="outbound-network-page__loading">
      <section class="outbound-network-page__overview" aria-labelledby="network-overview-title">
        <div class="outbound-network-page__overview-copy">
          <p class="outbound-network-page__section-kicker">{{ t('network.outbound.overview.kicker') }}</p>
          <h2 id="network-overview-title">{{ overviewTitle }}</h2>
          <p>{{ overviewDescription }}</p>
        </div>
        <dl class="outbound-network-page__overview-facts">
          <div>
            <dt>{{ t('network.outbound.overview.mode') }}</dt>
            <dd>{{ effectiveModeLabel }}</dd>
          </div>
          <div>
            <dt>{{ t('network.outbound.overview.lastTested') }}</dt>
            <dd>{{ latestTestedLabel }}</dd>
          </div>
          <div>
            <dt>{{ t('network.outbound.overview.lastChanged') }}</dt>
            <dd>{{ lastChangedLabel }}</dd>
          </div>
        </dl>
      </section>

      <div class="outbound-network-page__workspace">
        <main class="outbound-network-page__main">
          <t-card :title="t('network.outbound.routing.title')" class="outbound-network-page__configuration">
            <p class="outbound-network-page__card-description">{{ t('network.outbound.routing.description') }}</p>
            <t-form :data="form" label-align="top">
              <section class="outbound-network-page__form-section">
                <h3>{{ t('network.outbound.routing.mode.title') }}</h3>
                <p>{{ t('network.outbound.routing.mode.description') }}</p>
                <t-radio-group v-model="form.enabled" variant="default-filled">
                  <t-radio-button :value="false">{{ t('network.outbound.routing.mode.direct') }}</t-radio-button>
                  <t-radio-button :value="true">{{ t('network.outbound.routing.mode.proxy') }}</t-radio-button>
                </t-radio-group>
              </section>

              <section class="outbound-network-page__form-section">
                <h3>{{ t('network.outbound.routing.proxy.title') }}</h3>
                <p>{{ t('network.outbound.routing.proxy.description') }}</p>
                <t-form-item :label="t('network.outbound.httpProxy')">
                  <t-input
                    v-model="form.http_proxy"
                    clearable
                    :disabled="!form.enabled"
                    :placeholder="t('network.outbound.proxyPlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('network.outbound.httpsProxy')">
                  <t-input
                    v-model="form.https_proxy"
                    clearable
                    :disabled="!form.enabled"
                    :placeholder="t('network.outbound.proxyPlaceholder')"
                  />
                </t-form-item>
              </section>

              <section class="outbound-network-page__form-section">
                <h3>{{ t('network.outbound.routing.bypass.title') }}</h3>
                <p>{{ t('network.outbound.routing.bypass.description') }}</p>
                <t-form-item :label="t('network.outbound.noProxy')" :help="t('network.outbound.noProxySemantics')">
                  <t-tag-input
                    v-model="form.no_proxy"
                    clearable
                    excess-tags-display-type="break-line"
                    :placeholder="t('network.outbound.noProxyPlaceholder')"
                  />
                </t-form-item>
              </section>
            </t-form>
          </t-card>
        </main>

        <aside class="outbound-network-page__runtime" :aria-label="t('network.outbound.runtime.title')">
          <t-card :title="t('network.outbound.runtime.title')" class="outbound-network-page__runtime-card">
            <template #actions
              ><t-tag :theme="runtimeTheme" variant="light">{{ runtimeStatusLabel }}</t-tag></template
            >
            <section class="outbound-network-page__runtime-section">
              <h3>{{ t('network.outbound.runtime.policy') }}</h3>
              <dl class="outbound-network-page__runtime-list">
                <div>
                  <dt>{{ t('network.outbound.runtime.mode') }}</dt>
                  <dd>{{ effectiveModeLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('network.outbound.configurationSource') }}</dt>
                  <dd>{{ policySourceLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('network.outbound.runtime.bypass') }}</dt>
                  <dd>{{ bypassLabel }}</dd>
                </div>
              </dl>
            </section>
            <section class="outbound-network-page__runtime-section">
              <h3>{{ t('network.outbound.runtime.consumers') }}</h3>
              <ul class="outbound-network-page__consumer-list">
                <li v-for="consumer in consumers" :key="consumer.id">
                  <span>{{ t(consumer.title_key) }}</span>
                  <t-tag theme="primary" variant="light">{{ t('network.outbound.runtime.inherited') }}</t-tag>
                </li>
                <li v-if="!consumers.length" class="outbound-network-page__muted">
                  {{ t('network.outbound.runtime.noConsumers') }}
                </li>
              </ul>
            </section>
          </t-card>

          <t-card :title="t('network.outbound.diagnostics.title')" class="outbound-network-page__diagnostics">
            <p class="outbound-network-page__card-description">{{ t('network.outbound.diagnostics.description') }}</p>
            <t-select
              v-model="selectedTargetID"
              :disabled="!diagnosticTargets.length || diagnosing"
              :options="diagnosticTargetOptions"
              :placeholder="t('network.outbound.diagnostics.noTarget')"
              @change="loadDiagnosticHistory"
            />
            <t-button
              block
              class="outbound-network-page__diagnostic-action"
              theme="primary"
              variant="outline"
              :disabled="!selectedTargetID"
              :loading="diagnosing"
              @click="runDiagnostic"
            >
              <template #icon><link-icon /></template>
              {{ diagnosing ? t('network.outbound.diagnostics.running') : t('network.outbound.diagnostics.run') }}
            </t-button>

            <section class="outbound-network-page__latest-result">
              <div class="outbound-network-page__result-heading">
                <h3>{{ t('network.outbound.diagnostics.latest') }}</h3>
                <t-tag :theme="latestDiagnosticTheme" variant="light">{{ latestDiagnosticLabel }}</t-tag>
              </div>
              <dl class="outbound-network-page__runtime-list">
                <div>
                  <dt>{{ t('network.outbound.diagnostics.latency') }}</dt>
                  <dd>{{ latestLatencyLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('network.outbound.diagnostics.httpStatus') }}</dt>
                  <dd>{{ latestHTTPStatusLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('network.outbound.diagnostics.lastTested') }}</dt>
                  <dd>{{ latestTestedLabel }}</dd>
                </div>
              </dl>
              <t-alert v-if="latestDiagnostic?.error" theme="error" :message="latestDiagnostic.error" />
            </section>

            <section class="outbound-network-page__history">
              <h3>{{ t('network.outbound.diagnostics.history') }}</h3>
              <ul v-if="diagnosticHistory.length">
                <li v-for="item in diagnosticHistory.slice(0, 5)" :key="`${item.target_id}-${item.tested_at}`">
                  <t-tag :theme="item.status === 'connected' ? 'success' : 'danger'" variant="light">
                    {{
                      item.status === 'connected'
                        ? t('network.outbound.diagnostics.connected')
                        : t('network.outbound.diagnostics.failed')
                    }}
                  </t-tag>
                  <span>{{ formatCompactDateTime(item.tested_at, locale) }}</span>
                  <span>{{
                    item.latency_ms === null || item.latency_ms === undefined ? '-' : `${item.latency_ms} ms`
                  }}</span>
                </li>
              </ul>
              <p v-else class="outbound-network-page__muted">{{ t('network.outbound.diagnostics.noHistory') }}</p>
            </section>
          </t-card>
        </aside>
      </div>
    </t-loading>
  </section>
</template>
<script setup lang="ts">
// 出站网络设置页只消费服务端注册的运行对象与净化诊断数据，草稿永远不替代已生效策略。
import { LinkIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';
import { PageHeader } from '@/shared/components/page';

import {
  diagnoseOutboundNetwork,
  getOutboundNetworkDiagnosticHistory,
  getOutboundNetworkPolicy,
  resetOutboundNetworkPolicy,
  updateOutboundNetworkPolicy,
} from '../../api/outbound';
import type {
  OutboundNetworkConfig,
  OutboundNetworkConsumer,
  OutboundNetworkDiagnostic,
  OutboundNetworkDiagnosticTarget,
  OutboundNetworkOverview,
} from '../../types/outbound';

defineOptions({ name: 'OutboundNetworkPage' });

const { locale, t } = useI18n();
const loading = ref(true);
const saving = ref(false);
const resetting = ref(false);
const diagnosing = ref(false);
const errorMessage = ref('');
const preconditionMessage = ref('');
const policySource = ref<'default' | 'override'>('default');
const selectedTargetID = ref('');
const diagnosticTargets = ref<OutboundNetworkDiagnosticTarget[]>([]);
const consumers = ref<OutboundNetworkConsumer[]>([]);
const diagnosticHistory = ref<OutboundNetworkDiagnostic[]>([]);
const form = reactive<OutboundNetworkConfig>(createDefaultPolicy());
const effectivePolicy = reactive<OutboundNetworkConfig>(createDefaultPolicy());
const etag = ref<string | null>(null);
let savedPolicy = JSON.stringify(createDefaultPolicy());

const dirty = computed(() => JSON.stringify(form) !== savedPolicy);
const latestDiagnostic = computed(() => diagnosticHistory.value[0] ?? null);
const policySourceLabel = computed(() =>
  policySource.value === 'override' ? t('network.outbound.sourceOverride') : t('network.outbound.sourceDefault'),
);
const effectiveModeLabel = computed(() =>
  effectivePolicy.enabled ? t('network.outbound.routing.mode.proxy') : t('network.outbound.routing.mode.direct'),
);
const bypassLabel = computed(() =>
  effectivePolicy.no_proxy.length
    ? t('network.outbound.runtime.ruleCount', { count: effectivePolicy.no_proxy.length })
    : t('network.outbound.notConfigured'),
);
const runtimeStatusLabel = computed(() =>
  latestDiagnostic.value?.status === 'failed'
    ? t('network.outbound.overview.degraded')
    : latestDiagnostic.value?.status === 'connected'
      ? t('network.outbound.overview.healthy')
      : t('network.outbound.overview.unverified'),
);
const runtimeTheme = computed(() =>
  latestDiagnostic.value?.status === 'failed' ? 'danger' : latestDiagnostic.value ? 'success' : 'warning',
);
const overviewTitle = computed(() => runtimeStatusLabel.value);
const overviewDescription = computed(() =>
  latestDiagnostic.value?.status === 'failed'
    ? t('network.outbound.overview.degradedDescription')
    : latestDiagnostic.value
      ? t('network.outbound.overview.healthyDescription')
      : t('network.outbound.overview.unverifiedDescription'),
);
const latestDiagnosticLabel = computed(() =>
  latestDiagnostic.value?.status === 'connected'
    ? t('network.outbound.diagnostics.connected')
    : latestDiagnostic.value?.status === 'failed'
      ? t('network.outbound.diagnostics.failed')
      : t('network.outbound.diagnostics.notTested'),
);
const latestDiagnosticTheme = computed(() =>
  latestDiagnostic.value?.status === 'connected' ? 'success' : latestDiagnostic.value ? 'danger' : 'default',
);
const latestLatencyLabel = computed(() =>
  latestDiagnostic.value?.latency_ms === null || latestDiagnostic.value?.latency_ms === undefined
    ? '-'
    : `${latestDiagnostic.value.latency_ms} ms`,
);
const latestHTTPStatusLabel = computed(() => latestDiagnostic.value?.http_status ?? '-');
const latestTestedLabel = computed(() =>
  latestDiagnostic.value
    ? formatCompactDateTime(latestDiagnostic.value.tested_at, locale)
    : t('network.outbound.diagnostics.notTested'),
);
const lastChangedLabel = computed(() => {
  const responsePolicy = latestResponsePolicy.value;
  if (!responsePolicy?.updated_at) return t('network.outbound.overview.usingDefault');
  const user = responsePolicy.updated_by_name || t('network.outbound.overview.unknownUser');
  return t('network.outbound.overview.changedBy', {
    user,
    time: formatCompactDateTime(responsePolicy.updated_at, locale),
  });
});
const diagnosticTargetOptions = computed(() =>
  diagnosticTargets.value.map((target) => ({ label: t(target.title_key), value: target.id })),
);
const latestResponsePolicy = ref<OutboundNetworkOverview['policy'] | null>(null);

function createDefaultPolicy(): OutboundNetworkConfig {
  return { enabled: false, http_proxy: '', https_proxy: '', no_proxy: [] };
}

function copyPolicy(target: OutboundNetworkConfig, sourcePolicy: OutboundNetworkConfig) {
  target.enabled = sourcePolicy.enabled;
  target.http_proxy = sourcePolicy.http_proxy;
  target.https_proxy = sourcePolicy.https_proxy;
  target.no_proxy = Array.isArray(sourcePolicy.no_proxy) ? [...sourcePolicy.no_proxy] : [];
}

function applyResponse(response: Awaited<ReturnType<typeof getOutboundNetworkPolicy>>) {
  const payload = response.data;
  copyPolicy(form, payload.policy.config);
  copyPolicy(effectivePolicy, payload.policy.config);
  policySource.value = payload.policy.source;
  latestResponsePolicy.value = payload.policy;
  etag.value = response.etag;
  savedPolicy = JSON.stringify(form);
  diagnosticTargets.value = payload.diagnostic_targets;
  consumers.value = payload.consumers;
  if (!diagnosticTargets.value.some((target) => target.id === selectedTargetID.value)) {
    selectedTargetID.value = diagnosticTargets.value[0]?.id ?? '';
  }
}

function resolveLocalizedErrorMessage(error: unknown, fallbackKey: string): string {
  const messageKey =
    error && typeof error === 'object' && 'messageKey' in error && typeof error.messageKey === 'string'
      ? error.messageKey
      : undefined;
  return messageKey && t(messageKey) !== messageKey ? t(messageKey) : t(fallbackKey);
}

async function loadDiagnosticHistory() {
  if (!selectedTargetID.value) {
    diagnosticHistory.value = [];
    return;
  }
  try {
    diagnosticHistory.value = (await getOutboundNetworkDiagnosticHistory(selectedTargetID.value, 5)).items;
  } catch {
    diagnosticHistory.value = [];
  }
}

async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    applyResponse(await getOutboundNetworkPolicy());
    preconditionMessage.value = '';
    await loadDiagnosticHistory();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(error, 'network.outbound.loadFailed');
  } finally {
    loading.value = false;
  }
}

function isPreconditionFailure(error: unknown) {
  return Boolean(
    error &&
    typeof error === 'object' &&
    'status' in error &&
    ((error as { status?: unknown }).status === 412 || (error as { status?: unknown }).status === 428),
  );
}

function showPreconditionFailure() {
  preconditionMessage.value = t('network.outbound.precondition.message');
}

async function reloadLatestPolicy() {
  await load();
}

async function save() {
  if (form.enabled && !form.http_proxy.trim() && !form.https_proxy.trim()) {
    MessagePlugin.error(t('network.outbound.routing.proxy.required'));
    return;
  }
  if (!etag.value) {
    showPreconditionFailure();
    return;
  }
  saving.value = true;
  try {
    applyResponse(await updateOutboundNetworkPolicy({ ...form, no_proxy: [...form.no_proxy] }, etag.value));
    preconditionMessage.value = '';
    MessagePlugin.success(t('network.outbound.saveSuccess'));
  } catch (error) {
    if (isPreconditionFailure(error)) {
      showPreconditionFailure();
      return;
    }
    MessagePlugin.error(resolveLocalizedErrorMessage(error, 'network.outbound.saveFailed'));
  } finally {
    saving.value = false;
  }
}

async function resetToDefault() {
  if (!etag.value) {
    showPreconditionFailure();
    return;
  }
  resetting.value = true;
  try {
    applyResponse(await resetOutboundNetworkPolicy(etag.value));
    preconditionMessage.value = '';
    MessagePlugin.success(t('network.outbound.resetSuccess'));
  } catch (error) {
    if (isPreconditionFailure(error)) {
      showPreconditionFailure();
      return;
    }
    MessagePlugin.error(resolveLocalizedErrorMessage(error, 'network.outbound.resetFailed'));
  } finally {
    resetting.value = false;
  }
}

async function runDiagnostic() {
  if (!selectedTargetID.value) return;
  diagnosing.value = true;
  try {
    await diagnoseOutboundNetwork(selectedTargetID.value);
    await loadDiagnosticHistory();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(error, 'network.outbound.diagnostics.failedMessage'));
  } finally {
    diagnosing.value = false;
  }
}

onMounted(load);
</script>
<style scoped>
.outbound-network-page {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-width: 0;
}

.outbound-network-page__scope-content,
.outbound-network-page__scope-items {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.outbound-network-page__scope-content {
  justify-content: space-between;
}

.outbound-network-page__scope-content > div:first-child {
  display: grid;
  gap: var(--td-comp-margin-xs);
}

.outbound-network-page__precondition-alert {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-l);
}

.outbound-network-page__scope-content span,
.outbound-network-page__card-description,
.outbound-network-page__form-section > p,
.outbound-network-page__overview-copy > p {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.outbound-network-page__loading {
  min-height: 24rem;
}

.outbound-network-page__overview {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  display: flex;
  gap: var(--td-comp-margin-xl);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xl);
}

.outbound-network-page__section-kicker {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
}

.outbound-network-page__overview h2,
.outbound-network-page__form-section h3,
.outbound-network-page__runtime-section h3,
.outbound-network-page__latest-result h3,
.outbound-network-page__history h3 {
  font: var(--td-font-title-medium);
  margin: 0;
}

.outbound-network-page__overview-copy {
  display: grid;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}

.outbound-network-page__overview-facts {
  display: grid;
  flex: 0 0 auto;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(3, minmax(9rem, 1fr));
  margin: 0;
}

.outbound-network-page__overview-facts div,
.outbound-network-page__runtime-list div {
  display: grid;
  gap: var(--td-comp-margin-xs);
}

.outbound-network-page__overview-facts dt,
.outbound-network-page__runtime-list dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.outbound-network-page__overview-facts dd,
.outbound-network-page__runtime-list dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: 0;
  overflow-wrap: anywhere;
}

.outbound-network-page__workspace {
  align-items: start;
  display: grid;
  gap: var(--td-comp-margin-xl);
  grid-template-columns: minmax(0, 8fr) minmax(19rem, 4fr);
}

.outbound-network-page__main,
.outbound-network-page__runtime {
  min-width: 0;
}

.outbound-network-page__runtime {
  display: grid;
  gap: var(--td-comp-margin-xl);
  position: sticky;
  top: var(--td-comp-margin-xl);
}

.outbound-network-page__configuration :deep(.t-card__body) {
  max-width: 48rem;
}

.outbound-network-page__form-section,
.outbound-network-page__runtime-section,
.outbound-network-page__latest-result,
.outbound-network-page__history {
  border-top: 1px solid var(--td-component-border);
  display: grid;
  gap: var(--td-comp-margin-m);
  margin-top: var(--td-comp-margin-xl);
  padding-top: var(--td-comp-paddingTB-l);
}

.outbound-network-page__form-section:first-of-type {
  border-top: 0;
  margin-top: var(--td-comp-margin-l);
  padding-top: 0;
}

.outbound-network-page__runtime-list {
  display: grid;
  gap: var(--td-comp-margin-m);
  margin: 0;
}

.outbound-network-page__consumer-list,
.outbound-network-page__history ul {
  display: grid;
  gap: var(--td-comp-margin-s);
  list-style: none;
  margin: 0;
  padding: 0;
}

.outbound-network-page__consumer-list li,
.outbound-network-page__history li {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
  min-width: 0;
}

.outbound-network-page__history li span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.outbound-network-page__diagnostic-action {
  margin-top: var(--td-comp-margin-l);
}

.outbound-network-page__result-heading {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.outbound-network-page__muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

@media (width <= 1199px) {
  .outbound-network-page__workspace {
    grid-template-columns: minmax(0, 1fr);
  }

  .outbound-network-page__runtime {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    position: static;
  }
}

@media (width <= 767px) {
  .outbound-network-page__overview {
    align-items: stretch;
    flex-direction: column;
  }

  .outbound-network-page__overview-facts,
  .outbound-network-page__runtime {
    grid-template-columns: minmax(0, 1fr);
  }

  .outbound-network-page__scope-content {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
