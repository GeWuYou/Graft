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
    />

    <management-toolbar>
      <template #actions>
        <t-button theme="default" variant="outline" :loading="resetting" @click="resetToDefault">
          {{ t('network.outbound.resetToDefault') }}
        </t-button>
        <t-button theme="primary" :loading="saving" @click="save">
          {{ t('network.outbound.save') }}
        </t-button>
      </template>
    </management-toolbar>

    <t-alert
      v-if="errorMessage"
      class="outbound-network-page__alert"
      theme="error"
      :title="t('network.outbound.loadFailed')"
      :message="errorMessage"
    />

    <t-loading :loading="loading" class="outbound-network-page__loading">
      <div class="outbound-network-page__grid">
        <t-card class="outbound-network-page__policy" :title="t('network.outbound.title')">
          <t-form :data="form" label-align="top">
            <t-form-item :label="t('network.outbound.enabled')" :help="t('network.outbound.enabledHelp')">
              <t-switch v-model="form.enabled" />
            </t-form-item>
            <t-form-item :label="t('network.outbound.httpProxy')">
              <t-input v-model="form.http_proxy" clearable :placeholder="t('network.outbound.proxyPlaceholder')" />
            </t-form-item>
            <t-form-item :label="t('network.outbound.httpsProxy')">
              <t-input v-model="form.https_proxy" clearable :placeholder="t('network.outbound.proxyPlaceholder')" />
            </t-form-item>
            <t-form-item
              :label="t('network.outbound.noProxy')"
              :help="`${t('network.outbound.noProxyHelp')} ${t('network.outbound.noProxySemantics')}`"
            >
              <t-tag-input
                v-model="form.no_proxy"
                clearable
                excess-tags-display-type="break-line"
                :placeholder="t('network.outbound.noProxyPlaceholder')"
              />
            </t-form-item>
          </t-form>
        </t-card>

        <aside class="outbound-network-page__side">
          <t-card :title="t('network.outbound.effectivePolicy')">
            <t-descriptions bordered :column="1" size="small">
              <t-descriptions-item :label="t('network.outbound.configurationSource')">
                <t-tag :theme="policySourceTheme" variant="light">{{ policySourceLabel }}</t-tag>
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.enabled')">
                <t-tag :theme="effectivePolicy.enabled ? 'success' : 'default'" variant="light">
                  {{ effectiveEnabledLabel }}
                </t-tag>
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.httpProxy')">
                {{ effectivePolicy.http_proxy || t('network.outbound.notConfigured') }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.httpsProxy')">
                {{ effectivePolicy.https_proxy || t('network.outbound.notConfigured') }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.noProxy')">
                {{
                  effectivePolicy.no_proxy.length
                    ? effectivePolicy.no_proxy.join(', ')
                    : t('network.outbound.directConnection')
                }}
              </t-descriptions-item>
            </t-descriptions>
          </t-card>

          <t-card :title="t('network.outbound.diagnostic.title')">
            <p class="outbound-network-page__card-description">{{ t('network.outbound.diagnostic.description') }}</p>
            <div class="outbound-network-page__diagnostic-target">{{ diagnosticTargetLabel }}</div>
            <t-button
              block
              theme="primary"
              variant="outline"
              :disabled="!diagnosticTarget"
              :loading="diagnosing"
              @click="runDiagnostic"
            >
              <template #icon><link-icon /></template>
              {{ diagnosing ? t('network.outbound.diagnostic.running') : t('network.outbound.diagnostic.run') }}
            </t-button>
            <t-alert
              v-if="diagnostic"
              class="outbound-network-page__diagnostic-result"
              :theme="diagnostic.status === 'connected' ? 'success' : 'error'"
              :message="diagnostic.error || diagnosticStatusLabel"
            />
            <t-descriptions class="outbound-network-page__diagnostic-details" :column="1" size="small">
              <t-descriptions-item :label="t('network.outbound.diagnostic.latency')">
                {{ diagnostic?.latency_ms == null ? '-' : `${diagnostic.latency_ms} ms` }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.diagnostic.httpStatus')">
                {{ diagnostic?.http_status ?? '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('network.outbound.diagnostic.lastTested')">
                {{
                  diagnostic
                    ? formatCompactDateTime(diagnostic.tested_at, locale)
                    : t('network.outbound.diagnostic.notTested')
                }}
              </t-descriptions-item>
            </t-descriptions>
          </t-card>
        </aside>
      </div>

      <t-alert
        class="outbound-network-page__docker-notice"
        theme="info"
        :message="t('network.outbound.dockerNotice')"
      />
    </t-loading>
  </section>
</template>
<script setup lang="ts">
// 出站网络页面只管理平台网络策略；保存后的连通性测试由服务器执行固定目标，避免管理员输入任意地址。
import { LinkIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime, ManagementToolbar } from '@/shared/components/management';
import { PageHeader } from '@/shared/components/page';

import {
  diagnoseOutboundNetwork,
  getOutboundNetworkPolicy,
  resetOutboundNetworkPolicy,
  updateOutboundNetworkPolicy,
} from '../../api/outbound';
import type {
  OutboundNetworkDiagnostic,
  OutboundNetworkDiagnosticTarget,
  OutboundNetworkPolicy,
  OutboundNetworkPolicyResponse,
} from '../../types/outbound';

defineOptions({ name: 'OutboundNetworkPage' });

const { locale, t } = useI18n();
const loading = ref(true);
const saving = ref(false);
const resetting = ref(false);
const diagnosing = ref(false);
const errorMessage = ref('');
const diagnostic = ref<OutboundNetworkDiagnostic | null>(null);
const diagnosticTarget = ref<OutboundNetworkDiagnosticTarget | null>(null);
const source = ref<'default' | 'override'>('default');
const form = reactive<OutboundNetworkPolicy>(createDefaultPolicy());
const effectivePolicy = reactive<OutboundNetworkPolicy>(createDefaultPolicy());

const policySourceLabel = computed(() =>
  source.value === 'override' ? t('network.outbound.sourceOverride') : t('network.outbound.sourceDefault'),
);
const policySourceTheme = computed(() => (source.value === 'override' ? 'primary' : 'default'));
const effectiveEnabledLabel = computed(() =>
  effectivePolicy.enabled ? t('network.outbound.enabledState.enabled') : t('network.outbound.enabledState.disabled'),
);
const diagnosticTargetLabel = computed(() =>
  diagnosticTarget.value ? t(diagnosticTarget.value.title_key) : t('network.outbound.diagnostic.unavailable'),
);
const diagnosticStatusLabel = computed(() =>
  diagnostic.value?.status === 'connected'
    ? t('network.outbound.diagnostic.connected')
    : t('network.outbound.diagnostic.failed'),
);

function createDefaultPolicy(): OutboundNetworkPolicy {
  return { enabled: false, http_proxy: '', https_proxy: '', no_proxy: [] };
}

function copyPolicy(target: OutboundNetworkPolicy, sourcePolicy: OutboundNetworkPolicy) {
  target.enabled = sourcePolicy.enabled;
  target.http_proxy = sourcePolicy.http_proxy;
  target.https_proxy = sourcePolicy.https_proxy;
  // COMPAT(owner=server/modules/network/policy.go, cleanup=所有已部署后端均保证 no_proxy 序列化为数组): 兼容旧服务返回的 null，避免页面加载失败。
  target.no_proxy = Array.isArray(sourcePolicy.no_proxy) ? [...sourcePolicy.no_proxy] : [];
}

function applyResponse(response: OutboundNetworkPolicyResponse) {
  copyPolicy(form, response.policy.config);
  copyPolicy(effectivePolicy, response.policy.config);
  source.value = response.policy.source;
  diagnosticTarget.value = response.diagnostic_targets[0] ?? null;
}

async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    applyResponse(await getOutboundNetworkPolicy());
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('network.outbound.loadFailed');
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    applyResponse(await updateOutboundNetworkPolicy({ ...form, no_proxy: [...form.no_proxy] }));
    MessagePlugin.success(t('network.outbound.saveSuccess'));
  } catch {
    MessagePlugin.error(t('network.outbound.saveFailed'));
  } finally {
    saving.value = false;
  }
}

async function resetToDefault() {
  resetting.value = true;
  try {
    applyResponse(await resetOutboundNetworkPolicy());
    MessagePlugin.success(t('network.outbound.resetSuccess'));
  } catch {
    MessagePlugin.error(t('network.outbound.resetFailed'));
  } finally {
    resetting.value = false;
  }
}

async function runDiagnostic() {
  if (!diagnosticTarget.value) return;
  diagnosing.value = true;
  try {
    diagnostic.value = await diagnoseOutboundNetwork(diagnosticTarget.value.id);
  } catch {
    diagnostic.value = {
      target_id: diagnosticTarget.value.id,
      status: 'failed',
      tested_at: new Date().toISOString(),
      error: t('network.outbound.diagnostic.failedMessage'),
    };
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

.outbound-network-page__alert,
.outbound-network-page__docker-notice {
  margin: 0;
}

.outbound-network-page__loading {
  min-height: 18rem;
}

.outbound-network-page__grid {
  align-items: start;
  display: grid;
  gap: var(--td-comp-margin-xl);
  grid-template-columns: minmax(0, 1fr) minmax(20rem, 26rem);
}

.outbound-network-page__side {
  display: grid;
  gap: var(--td-comp-margin-xl);
}

.outbound-network-page__policy :deep(.t-form) {
  max-width: 46rem;
}

.outbound-network-page__card-description {
  color: var(--td-text-color-secondary);
  line-height: var(--td-line-height-body-medium);
  margin: 0 0 var(--td-comp-margin-l);
}

.outbound-network-page__diagnostic-target {
  color: var(--td-text-color-primary);
  font-weight: 600;
  margin-bottom: var(--td-comp-margin-l);
}

.outbound-network-page__diagnostic-result {
  margin-top: var(--td-comp-margin-l);
}

.outbound-network-page__diagnostic-details {
  margin-top: var(--td-comp-margin-l);
}

@media (width <= 900px) {
  .outbound-network-page__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
