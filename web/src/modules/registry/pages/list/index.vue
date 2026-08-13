<template>
  <section class="registry-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="menu.registries.title"
        :description="t('registry.list.description')"
        description-key="registry.list.description"
        :source="{ labelKey: 'menu.section.shared_resources', fallback: t('menu.section.shared_resources') }"
      >
        <template #actions>
          <t-button data-testid="registry-create" theme="primary" @click="openCreate">{{
            t('registry.list.add')
          }}</t-button>
        </template>
      </management-page-header>

      <resource-query-panel
        v-model="resourceQueryState"
        :config="queryConfig"
        :loading="loading"
        @reset="resetQuery"
        @search="applyQuery"
      />

      <management-paged-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :columns="columns"
        :empty-description="
          t(resourceQueryState.keyword ? 'registry.list.filteredEmptyDescription' : 'registry.list.emptyDescription')
        "
        :empty-title="t(resourceQueryState.keyword ? 'registry.list.filteredEmpty' : 'registry.list.empty')"
        :footer-summary="t('registry.list.summary', { count: total })"
        :loading="loading"
        :rows="items"
        :total="total"
        row-key="connection_ref"
        :pagination-props="{ showPageNumber: true }"
        @page-change="handlePageChange"
      >
        <template #feedback><t-alert v-if="errorMessage" theme="error" :message="errorMessage" /></template>
        <template #toolbar>
          <table-view-toolbar :refresh-label="t('registry.list.refresh')" :refresh-loading="loading" @refresh="load" />
        </template>
        <template #credential="{ row }">
          <t-tag :theme="row.credential_configured ? 'success' : 'default'" variant="light">
            {{
              row.credential_configured
                ? t('registry.list.credential.configured')
                : t('registry.list.credential.anonymous')
            }}
          </t-tag>
        </template>
        <template #status="{ row }">
          <t-tag :theme="row.availability ? 'success' : 'warning'" variant="light">
            {{ registryStatusLabel(row) }}
          </t-tag>
        </template>
        <template #verified="{ row }">{{ formatLocaleDateTime(row.last_verified_at, locale) || '-' }}</template>
        <template #actions="{ row }">
          <table-action-menu
            :actions="registryRowActions(row)"
            :more-label="t('registry.list.more')"
            :more-label-fallback="t('registry.list.more')"
            @action="handleRegistryRowAction($event, row)"
          />
        </template>
        <template #empty-action>
          <t-button v-if="resourceQueryState.keyword" variant="outline" @click="resetQuery(resourceQueryState)">
            {{ t('registry.list.clearSearch') }}
          </t-button>
        </template>
      </management-paged-table>
    </management-page-content>

    <t-dialog
      v-model:visible="deleteDialogVisible"
      theme="danger"
      :header="t('registry.list.delete')"
      :body="t('registry.list.deleteConfirm')"
      :cancel-btn="t('registry.list.cancel')"
      :confirm-btn="{ content: t('registry.list.delete'), theme: 'danger', loading: deleting }"
      @confirm="confirmRemove"
    />

    <t-drawer
      v-model:visible="drawerVisible"
      :header="drawerTitle"
      size="min(720px, 92vw)"
      :confirm-btn="t('registry.list.save')"
      :confirm-loading="saving"
      @confirm="save"
    >
      <t-form :data="form" :rules="rules" label-align="top">
        <t-form-item :label="t('registry.list.form.connectionRef')" name="connection_ref">
          <t-input v-model="form.connection_ref" :disabled="Boolean(editingRef)" />
        </t-form-item>
        <t-form-item :label="t('registry.list.form.displayName')" name="display_name"
          ><t-input v-model="form.display_name"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.endpoint')" name="endpoint"
          ><t-input v-model="form.endpoint" placeholder="https://registry.example.com"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.description')"
          ><t-textarea v-model="form.description"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.enabled')"><t-switch v-model="form.enabled" /></t-form-item>
        <t-form-item :label="t('registry.list.form.insecure')"><t-switch v-model="form.insecure" /></t-form-item>
      </t-form>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Registry 管理页协调连接、仓库路径和用户授权；Build 仅通过受限目的地 API 消费这些事实。
import type { PageInfo, TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import {
  createActionColumn,
  ManagementPageContent,
  ManagementPageHeader,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { type ResourceQueryConfig, ResourceQueryPanel, type ResourceQueryState } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { createRegistry, deleteRegistry, getRegistries, updateRegistry, verifyRegistry } from '../../api/registry';
import { registryDetailPath } from '../../contract/paths';

type RegistryConnection = components['schemas']['registry-connection'];

const { locale, t } = useI18n();
const router = useRouter();
const items = ref<RegistryConnection[]>([]);
const pagination = ref({ current: 1, pageSize: 20 });
const search = ref('');
const total = ref(0);
const loading = ref(false);
const saving = ref(false);
const verifying = ref('');
const deleting = ref(false);
const errorMessage = ref('');
const drawerVisible = ref(false);
const editingRef = ref('');
const deleteDialogVisible = ref(false);
const deleteTargetRef = ref('');
const form = ref({
  connection_ref: '',
  display_name: '',
  provider: 'generic_oci' as const,
  endpoint: '',
  enabled: true,
  insecure: false,
  description: '',
});
let registryListRequestSequence = 0;

const drawerTitle = computed(() => (editingRef.value ? t('registry.list.edit') : t('registry.list.add')));
const rules = computed(() => ({
  connection_ref: [{ required: true, message: t('registry.list.form.connectionRef') }],
  display_name: [{ required: true, message: t('registry.list.form.displayName') }],
  endpoint: [{ required: true, message: t('registry.list.form.endpoint') }],
}));
const queryConfig = computed<ResourceQueryConfig>(() => ({
  resource: 'registry.list',
  search: true,
  filterBuilder: { enabled: false },
  placeholder: t('registry.list.search'),
}));
const resourceQueryState = computed<ResourceQueryState>({
  get: () => ({
    keyword: search.value,
    filters: {},
    page: pagination.value.current,
    pageSize: pagination.value.pageSize,
  }),
  set: (value) => {
    search.value = value.keyword;
    pagination.value.current = value.page;
    pagination.value.pageSize = value.pageSize;
  },
});
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'display_name', title: t('registry.list.columns.name'), minWidth: 160 },
  { colKey: 'provider', title: t('registry.list.columns.type'), width: 130 },
  { colKey: 'endpoint', title: t('registry.list.columns.endpoint'), minWidth: 230 },
  { colKey: 'credential', title: t('registry.list.columns.credential'), width: 120 },
  { colKey: 'status', title: t('registry.list.columns.status'), width: 130 },
  { colKey: 'verified', title: t('registry.list.columns.verified'), minWidth: 170 },
  createActionColumn(t('registry.list.columns.actions'), 160, 'center', 'actions'),
]);

onMounted(() => void load());

async function load() {
  const requestSequence = ++registryListRequestSequence;
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getRegistries({
      search: search.value || undefined,
      limit: pagination.value.pageSize,
      offset: (pagination.value.current - 1) * pagination.value.pageSize,
    });
    if (requestSequence !== registryListRequestSequence) {
      return;
    }

    items.value = response.items ?? [];
    total.value = response.total ?? 0;
  } catch (error) {
    if (requestSequence !== registryListRequestSequence) {
      return;
    }

    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    if (requestSequence === registryListRequestSequence) {
      loading.value = false;
    }
  }
}
function applyQuery(value: ResourceQueryState) {
  resourceQueryState.value = value;
  pagination.value.current = 1;
  void load();
}
function resetQuery(value: ResourceQueryState) {
  resourceQueryState.value = value;
  pagination.value.current = 1;
  void load();
}
function handlePageChange(pageInfo: PageInfo) {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  void load();
}
function registryStatusLabel(row: RegistryConnection) {
  if (row.availability) {
    return t('registry.list.status.available');
  }

  switch (row.verification_status) {
    case 'failed':
      return t('registry.list.status.failed');
    case 'unknown':
      return t('registry.list.status.unknown');
    case 'verified':
      return t('registry.list.status.unavailable');
  }

  return t('registry.list.status.unknown');
}
function registryRowActions(row: RegistryConnection) {
  return [
    { label: t('registry.list.edit'), value: 'edit' },
    {
      disabled: verifying.value === row.connection_ref,
      label: t('registry.list.verify'),
      value: 'verify',
    },
    { danger: true, label: t('registry.list.delete'), value: 'delete' },
  ];
}
function handleRegistryRowAction(action: string, row: RegistryConnection) {
  if (action === 'edit') {
    void router.push(registryDetailPath(row.connection_ref));
    return;
  }
  if (action === 'verify') {
    void verify(row.connection_ref);
    return;
  }
  if (action === 'delete') {
    deleteTargetRef.value = row.connection_ref;
    deleteDialogVisible.value = true;
  }
}
function openCreate() {
  editingRef.value = '';
  form.value = {
    connection_ref: '',
    display_name: '',
    provider: 'generic_oci',
    endpoint: '',
    enabled: true,
    insecure: false,
    description: '',
  };
  drawerVisible.value = true;
}
async function save() {
  saving.value = true;
  try {
    if (editingRef.value)
      await updateRegistry(editingRef.value, {
        display_name: form.value.display_name,
        endpoint: form.value.endpoint,
        enabled: form.value.enabled,
        insecure: form.value.insecure,
        description: form.value.description || null,
      });
    else await createRegistry({ ...form.value, description: form.value.description || null });
    drawerVisible.value = false;
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    saving.value = false;
  }
}
async function verify(connectionRef: string) {
  verifying.value = connectionRef;
  try {
    const result = await verifyRegistry(connectionRef);
    if (result.status === 'verified') {
      MessagePlugin.success(t('registry.list.verifySuccess'));
    } else {
      MessagePlugin.error(t('registry.list.verifyFailed'));
    }
    await load();
  } catch (error) {
    const message = resolveLocalizedErrorMessage(t, error, t('registry.list.verifyFailed'));
    errorMessage.value = message;
    MessagePlugin.error(message);
  } finally {
    verifying.value = '';
  }
}
async function confirmRemove() {
  if (!deleteTargetRef.value) return;
  deleting.value = true;
  try {
    await deleteRegistry(deleteTargetRef.value);
    deleteDialogVisible.value = false;
    deleteTargetRef.value = '';
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    deleting.value = false;
  }
}
</script>
<style scoped lang="less">
.registry-page {
  min-width: 0;
}
</style>
