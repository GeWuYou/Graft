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
        cards-visible
        density-scope="viewport"
        row-key="connection_ref"
        :pagination-props="{ showPageNumber: true }"
        @page-change="handlePageChange"
        @row-click="handleRowClick"
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
        <template #cards>
          <responsive-card-list v-if="items.length" class="registry-page__mobile-list">
            <article v-for="row in items" :key="row.connection_ref" class="registry-page__mobile-card">
              <button
                class="registry-page__mobile-card-main"
                type="button"
                :aria-label="t('registry.list.openDetail', { name: row.display_name })"
                @click="openDetail(row)"
              >
                <strong>{{ row.display_name }}</strong>
                <span>{{ row.endpoint }}</span>
              </button>
              <div class="registry-page__mobile-card-status">
                <t-tag :theme="row.credential_configured ? 'success' : 'default'" size="small" variant="light">
                  {{
                    row.credential_configured
                      ? t('registry.list.credential.configured')
                      : t('registry.list.credential.anonymous')
                  }}
                </t-tag>
                <t-tag :theme="row.availability ? 'success' : 'warning'" size="small" variant="light">
                  {{ registryStatusLabel(row) }}
                </t-tag>
              </div>
              <table-action-menu
                class="registry-page__mobile-card-actions"
                :actions="registryRowActions(row)"
                :more-label="t('registry.list.more')"
                :more-label-fallback="t('registry.list.more')"
                @click.stop
                @action="handleRegistryRowAction($event, row)"
              />
            </article>
          </responsive-card-list>
          <t-empty
            v-else-if="!loading"
            :title="t(resourceQueryState.keyword ? 'registry.list.filteredEmpty' : 'registry.list.empty')"
            :description="
              t(
                resourceQueryState.keyword
                  ? 'registry.list.filteredEmptyDescription'
                  : 'registry.list.emptyDescription',
              )
            "
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

    <t-dialog
      v-model:visible="verifyDialogVisible"
      :header="t('registry.list.verify')"
      :cancel-btn="t('registry.list.cancel')"
      :confirm-btn="{ content: t('registry.list.verify'), loading: verifying !== '' }"
      @confirm="confirmVerify"
    >
      <t-form label-align="top">
        <t-form-item :label="t('registry.list.verifyRepository')">
          <t-select
            v-model="verificationRepositoryRef"
            :loading="verificationOptionsLoading"
            :options="verificationRepositoryOptions"
          />
        </t-form-item>
        <t-form-item :label="t('registry.list.verifyTarget')">
          <t-select
            v-model="verificationRuntimeTargetID"
            :disabled="!verificationTargetOptions.length"
            :loading="verificationOptionsLoading"
            :options="verificationTargetOptions"
          />
        </t-form-item>
        <t-alert v-if="verificationTargetsLoaded && !verificationTargetOptions.length" theme="warning">
          <template #message>{{ t('registry.list.noAuthorizedBuildTargets') }}</template>
          <template #default>
            <router-link v-if="canManageRuntimeTargetAssignments" to="/infrastructure/runtime-targets">
              {{ t('registry.list.manageRuntimeTargetAccess') }}
            </router-link>
            <span v-else>{{ t('registry.list.contactAdministratorForRuntimeTargetAccess') }}</span>
          </template>
        </t-alert>
      </t-form>
    </t-dialog>

    <t-drawer
      v-model:visible="drawerVisible"
      :header="t('registry.list.add')"
      size="min(720px, 92vw)"
      :confirm-btn="t('registry.list.save')"
      :confirm-loading="saving"
      @confirm="save"
    >
      <registry-connection-form v-model="form" />
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Registry 列表页负责连接查询、创建和详情导航；仓库路径与使用授权由详情页管理。
import type { PageInfo, TableProps, TableRowData } from 'tdesign-vue-next';
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
import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';
import { getPermissionStore } from '@/store/modules/permission';

import {
  createRegistry,
  deleteRegistry,
  getRegistries,
  getRegistryRepositories,
  getRegistryVerificationTargets,
  verifyRegistry,
} from '../../api/registry';
import RegistryConnectionForm, { type RegistryConnectionFormData } from '../../components/RegistryConnectionForm.vue';
import { REGISTRY_DETAIL_MODE, registryDetailPath } from '../../contract/paths';

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
const deleteDialogVisible = ref(false);
const deleteTargetRef = ref('');
const verifyDialogVisible = ref(false);
const verificationConnectionRef = ref('');
const verificationRepositoryRef = ref('');
const verificationRuntimeTargetID = ref<number>();
const verificationOptionsLoading = ref(false);
const verificationRepositoryOptions = ref<Array<{ label: string; value: string }>>([]);
const verificationTargetOptions = ref<Array<{ label: string; value: number }>>([]);
const verificationTargetsLoaded = ref(false);
const canManageRuntimeTargetAssignments = computed(() =>
  getPermissionStore().hasPermission('runtime_target.assignment.manage'),
);
const form = ref<RegistryConnectionFormData>({
  connection_ref: '',
  display_name: '',
  endpoint: '',
  enabled: true,
  insecure: false,
  description: '',
});
let registryListRequestSequence = 0;

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
    void router.push(registryDetailPath(row.connection_ref, { mode: REGISTRY_DETAIL_MODE.EDIT }));
    return;
  }
  if (action === 'verify') {
    void openVerify(row.connection_ref);
    return;
  }
  if (action === 'delete') {
    deleteTargetRef.value = row.connection_ref;
    deleteDialogVisible.value = true;
  }
}
function openDetail(row: RegistryConnection) {
  void router.push(registryDetailPath(row.connection_ref));
}
function handleRowClick(row: TableRowData) {
  openDetail(row as RegistryConnection);
}
function openCreate() {
  form.value = {
    connection_ref: '',
    display_name: '',
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
    await createRegistry({ ...form.value, provider: 'generic_oci', description: form.value.description || null });
    drawerVisible.value = false;
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    saving.value = false;
  }
}
async function openVerify(connectionRef: string) {
  verificationConnectionRef.value = connectionRef;
  verificationRepositoryRef.value = '';
  verificationRuntimeTargetID.value = undefined;
  verificationOptionsLoading.value = true;
  verificationTargetsLoaded.value = false;
  try {
    const [repositories, targets] = await Promise.all([
      getRegistryRepositories(connectionRef, { limit: 100, offset: 0 }),
      getRegistryVerificationTargets(),
    ]);
    verificationRepositoryOptions.value = (repositories.items ?? []).map((item) => ({
      label: item.display_name || item.repository_ref,
      value: item.repository_ref,
    }));
    verificationTargetOptions.value = (targets.items ?? []).map((item) => ({
      label: item.display_name,
      value: item.target_id,
    }));
    verificationRepositoryRef.value = verificationRepositoryOptions.value[0]?.value ?? '';
    verificationRuntimeTargetID.value = verificationTargetOptions.value[0]?.value;
    verificationTargetsLoaded.value = true;
    verifyDialogVisible.value = true;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.verifyFailed'));
  } finally {
    verificationOptionsLoading.value = false;
  }
}
async function confirmVerify() {
  if (!verificationTargetOptions.value.length) {
    errorMessage.value = t('registry.list.noAuthorizedBuildTargets');
    return;
  }
  if (!verificationConnectionRef.value || !verificationRepositoryRef.value || !verificationRuntimeTargetID.value) {
    errorMessage.value = t('registry.list.verifySelectionRequired');
    return;
  }
  const connectionRef = verificationConnectionRef.value;
  verifying.value = connectionRef;
  try {
    const result = await verifyRegistry(connectionRef, {
      repository_ref: verificationRepositoryRef.value,
      runtime_target_id: verificationRuntimeTargetID.value,
    });
    if (result.status === 'verified') {
      MessagePlugin.success(t('registry.list.verifySuccess'));
    } else {
      MessagePlugin.error(t('registry.list.verifyFailed'));
    }
    await load();
    verifyDialogVisible.value = false;
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
.registry-page__mobile-list {
  padding-top: var(--graft-density-gap-12);
}

.registry-page__mobile-card {
  align-items: center;
  border-bottom: 1px solid var(--td-component-stroke);
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(0, 1fr) auto;
  padding: var(--graft-density-gap-12) 0;
}

.registry-page__mobile-card-main {
  align-items: flex-start;
  background: transparent;
  border: 0;
  color: inherit;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  padding: 0;
  text-align: left;
}

.registry-page__mobile-card-main strong,
.registry-page__mobile-card-main span {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.registry-page__mobile-card-main span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.registry-page__mobile-card-status {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  grid-column: 1;
}

.registry-page__mobile-card-actions {
  grid-column: 2;
  grid-row: 1 / span 2;
}
</style>
<style scoped lang="less">
.registry-page {
  min-width: 0;
}
</style>
