<template>
  <section class="registry-page">
    <header class="registry-page__header">
      <div>
        <h1>{{ t('registry.list.title') }}</h1>
        <p>{{ t('registry.list.description') }}</p>
      </div>
      <t-button theme="primary" @click="openCreate">{{ t('registry.list.add') }}</t-button>
    </header>

    <div class="registry-page__toolbar">
      <t-input
        v-model="search"
        clearable
        :placeholder="t('registry.list.search')"
        @clear="clearSearch"
        @enter="searchRegistries"
      />
      <t-button variant="outline" @click="load">{{ t('registry.list.refresh') }}</t-button>
    </div>
    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
    <t-table row-key="connection_ref" :data="items" :columns="columns" :loading="loading">
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
        <t-space size="small">
          <t-button theme="primary" variant="text" @click="openEdit(row)">{{ t('registry.list.edit') }}</t-button>
          <t-button
            theme="primary"
            variant="text"
            :loading="verifying === row.connection_ref"
            @click="verify(row.connection_ref)"
            >{{ t('registry.list.verify') }}</t-button
          >
          <t-popconfirm :content="t('registry.list.deleteConfirm')" @confirm="remove(row.connection_ref)">
            <t-button theme="danger" variant="text">{{ t('registry.list.delete') }}</t-button>
          </t-popconfirm>
        </t-space>
      </template>
      <template #empty><t-empty :description="t('registry.list.empty')" /></template>
    </t-table>
    <t-pagination
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :page-size-options="[10, 20, 50, 100]"
      :total="total"
      @change="handlePageChange"
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

      <section v-if="editingRef" class="registry-repositories">
        <div class="registry-repositories__header">
          <h2>{{ t('registry.list.repositories') }}</h2>
          <t-button size="small" variant="outline" @click="openCreateRepository">{{
            t('registry.list.addRepository')
          }}</t-button>
        </div>
        <t-table
          row-key="repository_ref"
          :data="repositories"
          :columns="repositoryColumns"
          :loading="repositoryLoading"
        >
          <template #repositoryActions="{ row }">
            <t-button theme="primary" variant="text" @click="openEditRepository(row)">
              {{ t('registry.list.edit') }}
            </t-button>
            <t-button theme="primary" variant="text" @click="openAssignments(row.repository_ref)">
              {{ t('registry.list.assignments') }}
            </t-button>
            <t-popconfirm
              :content="t('registry.list.repositoryDeleteConfirm', { repository: row.repository_ref })"
              @confirm="removeRepository(row.repository_ref)"
            >
              <t-button theme="danger" variant="text">{{ t('registry.list.delete') }}</t-button>
            </t-popconfirm>
          </template>
        </t-table>
      </section>
    </t-drawer>

    <t-drawer
      v-model:visible="assignmentDrawerVisible"
      :header="t('registry.list.assignments')"
      size="480px"
      :footer="false"
    >
      <t-form :data="assignmentForm" label-align="top" @submit="grantAssignment">
        <t-form-item :label="t('registry.list.form.userId')" name="user_id">
          <t-input-number v-model="assignmentForm.user_id" :min="1" />
        </t-form-item>
        <t-button theme="primary" type="submit" :loading="assignmentSaving">
          {{ t('registry.list.grantAssignment') }}
        </t-button>
      </t-form>
      <t-table row-key="user_id" :data="assignments" :columns="assignmentColumns" :loading="assignmentLoading">
        <template #created_at="{ row }">{{ formatLocaleDateTime(row.created_at, locale) }}</template>
        <template #assignmentActions="{ row }">
          <t-popconfirm
            :content="t('registry.list.revokeAssignmentConfirm', { userId: row.user_id })"
            @confirm="revokeAssignment(row.user_id)"
          >
            <t-button theme="danger" variant="text">{{ t('registry.list.revokeAssignment') }}</t-button>
          </t-popconfirm>
        </template>
      </t-table>
    </t-drawer>

    <t-drawer
      v-model:visible="repositoryDrawerVisible"
      :header="editingRepositoryRef ? t('registry.list.edit') : t('registry.list.addRepository')"
      size="480px"
      :confirm-btn="t('registry.list.save')"
      :confirm-loading="repositorySaving"
      @confirm="saveRepository"
    >
      <t-form :data="repositoryForm" label-align="top">
        <t-form-item :label="t('registry.list.form.repositoryRef')"
          ><t-input v-model="repositoryForm.repository_ref" :disabled="Boolean(editingRepositoryRef)"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.repositoryName')"
          ><t-input v-model="repositoryForm.display_name"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.allowPull')"
          ><t-switch v-model="repositoryForm.allow_pull"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.allowPush')"
          ><t-switch v-model="repositoryForm.allow_push"
        /></t-form-item>
      </t-form>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Registry 管理页协调连接、仓库路径和用户授权；Build 仅通过受限目的地 API 消费这些事实。
import type { PageInfo, TableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { components } from '@/contracts/openapi/generated/schema';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import {
  createRegistry,
  createRegistryRepository,
  deleteRegistry,
  deleteRegistryRepository,
  getRegistries,
  getRegistryRepositories,
  getRegistryRepositoryAssignments,
  grantRegistryRepositoryAssignment,
  revokeRegistryRepositoryAssignment,
  updateRegistry,
  updateRegistryRepository,
  verifyRegistry,
} from '../../api/registry';

type RegistryConnection = components['schemas']['registry-connection'];
type RegistryRepository = components['schemas']['registry-artifact-repository'];
type RegistryAssignment = components['schemas']['registry-artifact-repository-user-assignment'];

const { locale, t } = useI18n();
const items = ref<RegistryConnection[]>([]);
const repositories = ref<RegistryRepository[]>([]);
const assignments = ref<RegistryAssignment[]>([]);
const search = ref('');
const pagination = ref({ current: 1, pageSize: 20 });
const total = ref(0);
const loading = ref(false);
const repositoryLoading = ref(false);
const saving = ref(false);
const repositorySaving = ref(false);
const verifying = ref('');
const errorMessage = ref('');
const drawerVisible = ref(false);
const repositoryDrawerVisible = ref(false);
const assignmentDrawerVisible = ref(false);
const editingRef = ref('');
const assignmentRepositoryRef = ref('');
const editingRepositoryRef = ref('');
const form = ref({
  connection_ref: '',
  display_name: '',
  provider: 'generic_oci' as const,
  endpoint: '',
  enabled: true,
  insecure: false,
  description: '',
});
const repositoryForm = ref({ repository_ref: '', display_name: '', allow_pull: true, allow_push: true });
const assignmentForm = ref({ user_id: undefined as number | undefined });
const assignmentLoading = ref(false);
const assignmentSaving = ref(false);
let registryListRequestSequence = 0;

const drawerTitle = computed(() => (editingRef.value ? t('registry.list.edit') : t('registry.list.add')));
const rules = computed(() => ({
  connection_ref: [{ required: true, message: t('registry.list.form.connectionRef') }],
  display_name: [{ required: true, message: t('registry.list.form.displayName') }],
  endpoint: [{ required: true, message: t('registry.list.form.endpoint') }],
}));
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'display_name', title: t('registry.list.columns.name'), minWidth: 160 },
  { colKey: 'provider', title: t('registry.list.columns.type'), width: 130 },
  { colKey: 'endpoint', title: t('registry.list.columns.endpoint'), minWidth: 230 },
  { colKey: 'credential', title: t('registry.list.columns.credential'), width: 120 },
  { colKey: 'status', title: t('registry.list.columns.status'), width: 130 },
  { colKey: 'verified', title: t('registry.list.columns.verified'), minWidth: 170 },
  { colKey: 'actions', title: t('registry.list.columns.actions'), width: 230, fixed: 'right' },
]);
const repositoryColumns = computed<TableProps['columns']>(() => [
  { colKey: 'repository_ref', title: t('registry.list.form.repositoryRef'), minWidth: 220 },
  { colKey: 'display_name', title: t('registry.list.form.repositoryName'), minWidth: 160 },
  { colKey: 'repositoryActions', title: t('registry.list.columns.actions'), width: 240 },
]);
const assignmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'user_id', title: t('registry.list.form.userId'), width: 120 },
  { colKey: 'created_at', title: t('registry.list.columns.verified'), minWidth: 180 },
  { colKey: 'assignmentActions', title: t('registry.list.columns.actions'), width: 120 },
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
function searchRegistries() {
  pagination.value.current = 1;
  void load();
}
function clearSearch() {
  search.value = '';
  searchRegistries();
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
function openCreate() {
  editingRef.value = '';
  repositories.value = [];
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
async function openEdit(row: RegistryConnection) {
  editingRef.value = row.connection_ref;
  form.value = {
    connection_ref: row.connection_ref,
    display_name: row.display_name,
    provider: 'generic_oci',
    endpoint: row.endpoint,
    enabled: row.enabled,
    insecure: row.insecure,
    description: row.description ?? '',
  };
  drawerVisible.value = true;
  await loadRepositories();
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
    errorMessage.value = t('registry.list.verifyResult', { status: result.status });
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    verifying.value = '';
  }
}
async function remove(connectionRef: string) {
  try {
    await deleteRegistry(connectionRef);
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}
async function loadRepositories() {
  if (!editingRef.value) return;
  repositoryLoading.value = true;
  try {
    const response = await getRegistryRepositories(editingRef.value);
    repositories.value = response.items ?? [];
  } finally {
    repositoryLoading.value = false;
  }
}
async function saveRepository() {
  if (!editingRef.value) return;
  repositorySaving.value = true;
  try {
    if (editingRepositoryRef.value) {
      await updateRegistryRepository(editingRef.value, editingRepositoryRef.value, {
        display_name: repositoryForm.value.display_name,
        allow_pull: repositoryForm.value.allow_pull,
        allow_push: repositoryForm.value.allow_push,
      });
    } else {
      await createRegistryRepository(editingRef.value, repositoryForm.value);
    }
    repositoryDrawerVisible.value = false;
    editingRepositoryRef.value = '';
    repositoryForm.value = { repository_ref: '', display_name: '', allow_pull: true, allow_push: true };
    await loadRepositories();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    repositorySaving.value = false;
  }
}
function openCreateRepository() {
  editingRepositoryRef.value = '';
  repositoryForm.value = { repository_ref: '', display_name: '', allow_pull: true, allow_push: true };
  repositoryDrawerVisible.value = true;
}
function openEditRepository(repository: RegistryRepository) {
  editingRepositoryRef.value = repository.repository_ref;
  repositoryForm.value = {
    repository_ref: repository.repository_ref,
    display_name: repository.display_name,
    allow_pull: repository.allow_pull,
    allow_push: repository.allow_push,
  };
  repositoryDrawerVisible.value = true;
}
async function removeRepository(repositoryRef: string) {
  if (!editingRef.value) return;
  try {
    await deleteRegistryRepository(editingRef.value, repositoryRef);
    await loadRepositories();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}
async function openAssignments(repositoryRef: string) {
  assignmentRepositoryRef.value = repositoryRef;
  assignmentForm.value = { user_id: undefined };
  assignmentDrawerVisible.value = true;
  await loadAssignments();
}
async function loadAssignments() {
  if (!editingRef.value || !assignmentRepositoryRef.value) return;
  assignmentLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignments(editingRef.value, assignmentRepositoryRef.value);
    assignments.value = response.items ?? [];
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentLoading.value = false;
  }
}
async function grantAssignment() {
  if (!editingRef.value || !assignmentRepositoryRef.value || !assignmentForm.value.user_id) return;
  assignmentSaving.value = true;
  try {
    await grantRegistryRepositoryAssignment(editingRef.value, assignmentRepositoryRef.value, {
      user_id: assignmentForm.value.user_id,
    });
    assignmentForm.value = { user_id: undefined };
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}
async function revokeAssignment(userId: number) {
  if (!editingRef.value || !assignmentRepositoryRef.value) return;
  try {
    await revokeRegistryRepositoryAssignment(editingRef.value, assignmentRepositoryRef.value, userId);
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}
</script>
<style scoped lang="less">
.registry-page {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.registry-page__header,
.registry-page__toolbar,
.registry-repositories__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.registry-page__header h1,
.registry-repositories h2 {
  margin: 0;
}

.registry-page__header p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
}

.registry-page__toolbar :deep(.t-input) {
  max-width: 360px;
}

.registry-repositories {
  display: grid;
  gap: var(--graft-density-gap-12);
  margin-top: var(--graft-density-gap-24);
}
</style>
