<template>
  <section class="registry-detail" data-page-type="form-detail">
    <management-page-content>
      <management-page-header
        :title="connection?.display_name || t('registry.route.detail.title')"
        :description="connection?.endpoint || t('registry.route.detail.description')"
      >
        <template #actions>
          <t-button variant="outline" :loading="loading" @click="load">
            {{ t('registry.list.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
      <t-card :title="t('registry.route.detail.connection')" class="registry-detail__section">
        <t-descriptions v-if="connection" :column="2" bordered>
          <t-descriptions-item :label="t('registry.list.form.connectionRef')">{{
            connection.connection_ref
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('registry.list.form.endpoint')">{{ connection.endpoint }}</t-descriptions-item>
          <t-descriptions-item :label="t('registry.list.form.enabled')">{{
            connection.enabled ? t('registry.route.detail.enabled') : t('registry.route.detail.disabled')
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('registry.list.form.insecure')">{{
            connection.insecure ? t('registry.route.detail.enabled') : t('registry.route.detail.disabled')
          }}</t-descriptions-item>
        </t-descriptions>
        <t-loading v-else-if="loading" />
        <t-empty v-else :title="t('registry.route.detail.notFound')" />
      </t-card>

      <t-card :title="t('registry.list.repositories')" class="registry-detail__section">
        <template #actions>
          <t-button theme="primary" @click="openRepository()">{{ t('registry.list.addRepository') }}</t-button>
        </template>
        <management-paged-table
          v-model:current="repositoryPagination.current"
          v-model:page-size="repositoryPagination.pageSize"
          :columns="repositoryColumns"
          :empty-description="t('registry.route.detail.noRepositories')"
          :empty-title="t('registry.route.detail.noRepositories')"
          :footer-summary="t('registry.route.detail.repositorySummary', { count: repositoryTotal })"
          :loading="repositoryLoading"
          :rows="repositories"
          :selected-row-keys="selectedRepositoryRefs"
          :total="repositoryTotal"
          row-key="repository_ref"
          @page-change="handleRepositoryPageChange"
          @select-change="handleRepositorySelection"
        >
          <template #repositoryActions="{ row }">
            <t-space>
              <t-button size="small" variant="outline" @click="openRepository(row)">{{
                t('registry.list.edit')
              }}</t-button>
              <t-button size="small" variant="outline" @click="openAssignments(row.repository_ref)">{{
                t('registry.list.assignments')
              }}</t-button>
              <t-popconfirm
                :content="t('registry.list.repositoryDeleteConfirm', { repository: row.repository_ref })"
                @confirm="removeRepository(row.repository_ref)"
              >
                <t-button size="small" theme="danger" variant="text">{{ t('registry.list.delete') }}</t-button>
              </t-popconfirm>
            </t-space>
          </template>
          <template v-if="selectedRepositoryRefs.length" #batch>
            <management-batch-bar
              :selected-label="t('registry.list.batchDeleteRepositories', { count: selectedRepositoryRefs.length })"
              :clear-label="t('registry.list.cancel')"
              @clear="selectedRepositoryRefs = []"
            >
              <t-button
                theme="primary"
                variant="outline"
                :loading="batchRepositoryAssignmentSaving"
                @click="openBatchRepositoryAssignments"
              >
                {{ t('registry.list.batchAuthorizeRepositories') }}
              </t-button>
              <t-popconfirm
                theme="danger"
                :content="t('registry.list.batchDeleteConfirm', { count: selectedRepositoryRefs.length })"
                @confirm="removeSelectedRepositories"
              >
                <t-button theme="danger" variant="outline" :loading="repositorySaving">
                  {{ t('registry.list.batchDelete') }}
                </t-button>
              </t-popconfirm>
            </management-batch-bar>
          </template>
        </management-paged-table>
      </t-card>
    </management-page-content>

    <t-drawer
      v-model:visible="repositoryDrawerVisible"
      :header="editingRepositoryRef ? t('registry.list.edit') : t('registry.list.addRepository')"
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
        <t-form-item v-if="!editingRepositoryRef" :label="t('registry.list.form.grantCreatorUse')"
          ><t-switch v-model="repositoryForm.grant_creator_use"
        /></t-form-item>
      </t-form>
    </t-drawer>

    <t-drawer
      v-model:visible="batchRepositoryAssignmentDrawerVisible"
      :header="t('registry.list.batchAuthorizeRepositoriesTitle')"
      size="720px"
      :footer="false"
    >
      <paged-multi-select
        v-model:current="candidatePagination.current"
        v-model:keyword="candidateSearch"
        v-model:page-size="candidatePagination.pageSize"
        v-model:selected-keys="selectedCandidateUserIds"
        :cancel-label="t('registry.list.cancel')"
        :cell-slot-names="['authorizationState']"
        :columns="candidateColumns"
        :confirm-label="t('registry.list.batchAuthorizeRepositories')"
        :confirm-loading="batchRepositoryAssignmentSaving"
        :empty-description="t('registry.list.candidatesEmpty')"
        :empty-title="t('registry.list.candidatesEmpty')"
        :loading="candidateLoading"
        row-key="id"
        :rows="assignmentCandidates"
        :search-label="t('registry.list.candidateSearch')"
        :search-placeholder="t('registry.list.candidateSearchPlaceholder')"
        :selected-count-label="(count) => t('registry.list.candidateSelected', { count })"
        :total="candidateTotal"
        @cancel="closeBatchRepositoryAssignments"
        @confirm="grantSelectedRepositoryAssignments"
        @page-change="loadAssignmentCandidates"
        @search="searchAssignmentCandidates"
      >
        <template #authorizationState="{ row }">
          <t-tag :theme="candidateAuthorizationTheme(row.authorization_state)" variant="light">
            {{ candidateAuthorizationLabel(row.authorization_state) }}
          </t-tag>
        </template>
      </paged-multi-select>
    </t-drawer>

    <t-drawer
      v-model:visible="assignmentDrawerVisible"
      :header="t('registry.list.assignments')"
      size="480px"
      :footer="false"
    >
      <t-form :data="assignmentForm" label-align="top" @submit="grantAssignment">
        <t-form-item :label="t('registry.list.form.userId')"
          ><t-input-number v-model="assignmentForm.user_id" :min="1"
        /></t-form-item>
        <t-button theme="primary" type="submit" :loading="assignmentSaving">{{
          t('registry.list.grantAssignment')
        }}</t-button>
        <t-button v-if="assignments.length" variant="outline" :loading="assignmentSaving" @click="revokeAllAssignments">
          {{ t('registry.list.revokeAllAssignments') }}
        </t-button>
        <t-form-item :label="t('registry.list.form.batchUserIds')">
          <t-input v-model="batchUserIds" :placeholder="t('registry.list.form.batchUserIdsPlaceholder')" />
        </t-form-item>
        <t-button variant="outline" :loading="assignmentSaving" @click="grantBatchAssignments">
          {{ t('registry.list.grantBatchAssignment') }}
        </t-button>
      </t-form>
      <t-table
        row-key="user_id"
        :data="assignments"
        :columns="assignmentColumns"
        :loading="assignmentLoading"
        :selected-row-keys="selectedAssignmentIds"
        @select-change="handleAssignmentSelection"
      >
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
      <t-button
        v-if="selectedAssignmentIds.length"
        theme="danger"
        variant="outline"
        :loading="assignmentSaving"
        @click="revokeSelectedAssignments"
      >
        {{ t('registry.list.revokeBatchAssignment', { count: selectedAssignmentIds.length }) }}
      </t-button>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Registry 详情页拥有仓库路径与使用授权操作；连接列表只负责导航和连接级操作。
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import { ManagementBatchBar, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { PagedMultiSelect } from '@/shared/components/selection';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import {
  createRegistryRepository,
  deleteRegistryRepository,
  getRegistry,
  getRegistryRepositories,
  getRegistryRepositoryAssignmentCandidates,
  getRegistryRepositoryAssignments,
  grantRegistryRepositoryAssignment,
  revokeRegistryRepositoryAssignment,
  updateRegistryRepository,
} from '../../api/registry';

type RegistryConnection = components['schemas']['registry-connection'];
type RegistryRepository = components['schemas']['registry-artifact-repository'];
type RegistryAssignment = components['schemas']['registry-artifact-repository-user-assignment'];
type RegistryAssignmentCandidate = components['schemas']['registry-repository-assignment-candidate'];

const { locale, t } = useI18n();
const MAX_REPOSITORY_ASSIGNMENT_CANDIDATES = 100;
const route = useRoute();
const connectionRef = computed(() => String(route.params.connectionRef || ''));
const connection = ref<RegistryConnection | null>(null);
const repositories = ref<RegistryRepository[]>([]);
const assignments = ref<RegistryAssignment[]>([]);
const loading = ref(false);
const repositoryLoading = ref(false);
const repositorySaving = ref(false);
const repositoryPagination = ref({ current: 1, pageSize: 20 });
const repositoryTotal = ref(0);
const assignmentLoading = ref(false);
const assignmentSaving = ref(false);
const batchRepositoryAssignmentSaving = ref(false);
const candidateLoading = ref(false);
const assignmentCandidates = ref<RegistryAssignmentCandidate[]>([]);
const candidateTotal = ref(0);
const candidateSearch = ref('');
const candidatePagination = ref({ current: 1, pageSize: 20 });
const selectedRepositoryRefs = ref<Array<string | number>>([]);
const selectedCandidateUserIds = ref<Array<string | number>>([]);
const selectedAssignmentIds = ref<Array<string | number>>([]);
const batchUserIds = ref('');
const errorMessage = ref('');
const repositoryDrawerVisible = ref(false);
const assignmentDrawerVisible = ref(false);
const batchRepositoryAssignmentDrawerVisible = ref(false);
const editingRepositoryRef = ref('');
const assignmentRepositoryRef = ref('');
const repositoryForm = ref({
  repository_ref: '',
  display_name: '',
  allow_pull: true,
  allow_push: true,
  grant_creator_use: true,
});
const assignmentForm = ref({ user_id: undefined as number | undefined });

const repositoryColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'repository_ref', title: t('registry.list.form.repositoryRef'), minWidth: 240 },
  { colKey: 'display_name', title: t('registry.list.form.repositoryName'), minWidth: 160 },
  { colKey: 'repositoryActions', title: t('registry.list.columns.actions'), width: 280 },
]);
const assignmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'user_id', title: t('registry.list.form.userId'), width: 120 },
  { colKey: 'created_at', title: t('registry.list.columns.assignmentCreatedAt'), minWidth: 180 },
  { colKey: 'assignmentActions', title: t('registry.list.columns.actions'), width: 140 },
]);
const candidateColumns = computed<TableProps['columns']>(() => [
  {
    colKey: 'row-select',
    type: 'multiple' as const,
    width: 48,
    checkProps: ({ row }) => ({
      disabled: (row as RegistryAssignmentCandidate).authorization_state === 'all',
    }),
  },
  { colKey: 'display', title: t('registry.list.columns.candidateUser'), minWidth: 180 },
  { colKey: 'username', title: t('registry.list.columns.candidateUsername'), minWidth: 150 },
  { colKey: 'authorizationState', title: t('registry.list.columns.authorizationState'), width: 130 },
]);

onMounted(() => void load());

async function load() {
  if (!connectionRef.value) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    const [connectionResult, repositoryResult] = await Promise.all([
      getRegistry(connectionRef.value),
      getRegistryRepositories(connectionRef.value, {
        limit: repositoryPagination.value.pageSize,
        offset: (repositoryPagination.value.current - 1) * repositoryPagination.value.pageSize,
      }),
    ]);
    connection.value = connectionResult;
    repositories.value = repositoryResult.items ?? [];
    repositoryTotal.value = repositoryResult.total ?? repositories.value.length;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    loading.value = false;
  }
}

async function handleRepositoryPageChange(pageInfo: { current: number; pageSize: number }) {
  repositoryPagination.value = pageInfo;
  await load();
}

function openRepository(repository?: RegistryRepository) {
  editingRepositoryRef.value = repository?.repository_ref ?? '';
  repositoryForm.value = repository
    ? {
        repository_ref: repository.repository_ref,
        display_name: repository.display_name,
        allow_pull: repository.allow_pull,
        allow_push: repository.allow_push,
        grant_creator_use: true,
      }
    : { repository_ref: '', display_name: '', allow_pull: true, allow_push: true, grant_creator_use: true };
  repositoryDrawerVisible.value = true;
}

async function saveRepository() {
  if (!connectionRef.value) return;
  repositorySaving.value = true;
  try {
    if (editingRepositoryRef.value) {
      await updateRegistryRepository(connectionRef.value, editingRepositoryRef.value, {
        display_name: repositoryForm.value.display_name,
        allow_pull: repositoryForm.value.allow_pull,
        allow_push: repositoryForm.value.allow_push,
      });
    } else {
      await createRegistryRepository(connectionRef.value, repositoryForm.value);
    }
    repositoryDrawerVisible.value = false;
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    repositorySaving.value = false;
  }
}

async function removeRepository(repositoryRef: string) {
  if (!connectionRef.value) return;
  try {
    await deleteRegistryRepository(connectionRef.value, repositoryRef);
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}

function handleRepositorySelection(keys: Array<string | number>) {
  if (keys.length > MAX_REPOSITORY_ASSIGNMENT_CANDIDATES) {
    MessagePlugin.error(t('registry.list.repositorySelectionLimit', { count: MAX_REPOSITORY_ASSIGNMENT_CANDIDATES }));
    return;
  }
  selectedRepositoryRefs.value = keys;
}

async function removeSelectedRepositories() {
  if (!selectedRepositoryRefs.value.length || !connectionRef.value) return;
  repositorySaving.value = true;
  try {
    await Promise.all(
      selectedRepositoryRefs.value.map((key) => deleteRegistryRepository(connectionRef.value, String(key))),
    );
    selectedRepositoryRefs.value = [];
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    repositorySaving.value = false;
  }
}

function openBatchRepositoryAssignments() {
  if (selectedRepositoryRefs.value.length > MAX_REPOSITORY_ASSIGNMENT_CANDIDATES) {
    MessagePlugin.error(t('registry.list.repositorySelectionLimit', { count: MAX_REPOSITORY_ASSIGNMENT_CANDIDATES }));
    return;
  }
  candidateSearch.value = '';
  candidatePagination.value.current = 1;
  selectedCandidateUserIds.value = [];
  batchRepositoryAssignmentDrawerVisible.value = true;
  void loadAssignmentCandidates();
}

function closeBatchRepositoryAssignments() {
  if (batchRepositoryAssignmentSaving.value) return;
  batchRepositoryAssignmentDrawerVisible.value = false;
  selectedCandidateUserIds.value = [];
}

function parseUserIds(value: string) {
  return [
    ...new Set(
      value
        .split(/[,\s]+/)
        .map((item) => Number(item.trim()))
        .filter((item) => Number.isInteger(item) && item > 0),
    ),
  ];
}

async function grantSelectedRepositoryAssignments() {
  if (!connectionRef.value || !selectedRepositoryRefs.value.length) return;
  const userIds = [
    ...new Set(selectedCandidateUserIds.value.map(Number).filter((id) => Number.isInteger(id) && id > 0)),
  ];
  if (!userIds.length) return;

  batchRepositoryAssignmentSaving.value = true;
  try {
    await Promise.all(
      selectedRepositoryRefs.value.flatMap((repositoryRef) =>
        userIds.map((userId) =>
          grantRegistryRepositoryAssignment(connectionRef.value, String(repositoryRef), { user_id: userId }),
        ),
      ),
    );
    closeBatchRepositoryAssignments();
    selectedRepositoryRefs.value = [];
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    batchRepositoryAssignmentSaving.value = false;
  }
}

async function loadAssignmentCandidates() {
  if (!connectionRef.value || !selectedRepositoryRefs.value.length) return;
  if (selectedRepositoryRefs.value.length > MAX_REPOSITORY_ASSIGNMENT_CANDIDATES) {
    MessagePlugin.error(t('registry.list.repositorySelectionLimit', { count: MAX_REPOSITORY_ASSIGNMENT_CANDIDATES }));
    return;
  }
  candidateLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignmentCandidates(connectionRef.value, {
      repository_ref: selectedRepositoryRefs.value.map(String),
      search: candidateSearch.value.trim() || undefined,
      limit: candidatePagination.value.pageSize,
      offset: (candidatePagination.value.current - 1) * candidatePagination.value.pageSize,
    });
    assignmentCandidates.value = response.items ?? [];
    candidateTotal.value = response.total ?? 0;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    candidateLoading.value = false;
  }
}

function searchAssignmentCandidates() {
  candidatePagination.value.current = 1;
  void loadAssignmentCandidates();
}

function candidateAuthorizationLabel(state: RegistryAssignmentCandidate['authorization_state']) {
  return t(`registry.list.authorizationState.${state}`);
}

function candidateAuthorizationTheme(state: RegistryAssignmentCandidate['authorization_state']) {
  if (state === 'all') return 'success';
  if (state === 'partial') return 'warning';
  return 'default';
}

async function openAssignments(repositoryRef: string) {
  assignmentRepositoryRef.value = repositoryRef;
  assignmentForm.value = { user_id: undefined };
  assignmentDrawerVisible.value = true;
  await loadAssignments();
}

async function loadAssignments() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  assignmentLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignments(connectionRef.value, assignmentRepositoryRef.value);
    assignments.value = response.items ?? [];
    selectedAssignmentIds.value = [];
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentLoading.value = false;
  }
}

function handleAssignmentSelection(keys: Array<string | number>) {
  selectedAssignmentIds.value = keys;
}

function parseBatchUserIds() {
  return parseUserIds(batchUserIds.value);
}

async function grantBatchAssignments() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  const userIds = parseBatchUserIds();
  if (!userIds.length) return;
  assignmentSaving.value = true;
  try {
    await Promise.all(
      userIds.map((userId) =>
        grantRegistryRepositoryAssignment(connectionRef.value, assignmentRepositoryRef.value, { user_id: userId }),
      ),
    );
    batchUserIds.value = '';
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}

async function revokeSelectedAssignments() {
  if (!connectionRef.value || !assignmentRepositoryRef.value || !selectedAssignmentIds.value.length) return;
  assignmentSaving.value = true;
  try {
    await Promise.all(
      selectedAssignmentIds.value.map((userId) =>
        revokeRegistryRepositoryAssignment(connectionRef.value, assignmentRepositoryRef.value, Number(userId)),
      ),
    );
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}

async function grantAssignment() {
  if (!connectionRef.value || !assignmentRepositoryRef.value || !assignmentForm.value.user_id) return;
  assignmentSaving.value = true;
  try {
    await grantRegistryRepositoryAssignment(connectionRef.value, assignmentRepositoryRef.value, {
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
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  try {
    await revokeRegistryRepositoryAssignment(connectionRef.value, assignmentRepositoryRef.value, userId);
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}

async function revokeAllAssignments() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  assignmentSaving.value = true;
  try {
    await Promise.all(
      assignments.value.map(({ user_id: userId }) =>
        revokeRegistryRepositoryAssignment(connectionRef.value, assignmentRepositoryRef.value, userId),
      ),
    );
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}
</script>
<style scoped lang="less">
.registry-detail__section {
  margin-top: var(--graft-density-gap-16);
}
</style>
