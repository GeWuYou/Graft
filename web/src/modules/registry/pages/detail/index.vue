<template>
  <section class="registry-detail" data-page-type="form-detail">
    <management-page-content>
      <management-page-header
        :title="connection?.display_name || t('registry.route.detail.title')"
        :description="connection?.endpoint || t('registry.route.detail.description')"
      >
        <template #actions>
          <t-button v-if="connection" variant="outline" @click="openConnectionEdit">
            {{ t('registry.list.edit') }}
          </t-button>
          <t-button variant="outline" :loading="loading" @click="load">
            {{ t('registry.list.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
      <t-card :title="t('registry.route.detail.connection')" class="registry-detail__section">
        <t-descriptions v-if="connection" :column="isCompactDensity ? 1 : 2" bordered>
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
          cards-visible
          density-scope="viewport"
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
          <template #cards>
            <responsive-card-list v-if="repositories.length" class="registry-detail__repository-list">
              <article v-for="row in repositories" :key="row.repository_ref" class="registry-detail__repository-card">
                <div class="registry-detail__repository-card-main">
                  <strong>{{ row.display_name }}</strong>
                  <span>{{ row.repository_ref }}</span>
                </div>
                <div class="registry-detail__repository-card-permissions">
                  <t-tag :theme="row.allow_pull ? 'success' : 'default'" size="small" variant="light">
                    {{ t('registry.list.form.allowPull') }}
                  </t-tag>
                  <t-tag :theme="row.allow_push ? 'success' : 'default'" size="small" variant="light">
                    {{ t('registry.list.form.allowPush') }}
                  </t-tag>
                </div>
                <t-space class="registry-detail__repository-card-actions" size="small">
                  <t-button
                    v-for="action in repositoryCardActions(row)"
                    :key="action.value"
                    size="small"
                    variant="outline"
                    @click="handleRepositoryCardAction(row, action.value)"
                  >
                    {{ action.label }}
                  </t-button>
                </t-space>
              </article>
            </responsive-card-list>
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

    <responsive-dialog
      :visible="connectionEditVisible"
      :close-label="t('components.common.close')"
      :title="t('registry.route.detail.editConnection')"
      purpose="form"
      size="medium"
      :close-on-esc-keydown="!connectionSaving"
      :close-on-overlay-click="!connectionSaving"
      @update:visible="handleConnectionEditVisible"
    >
      <registry-connection-form ref="connectionFormRef" v-model="connectionForm" editing />
      <template #footer>
        <t-button variant="outline" :disabled="connectionSaving" @click="closeConnectionEdit">
          {{ t('registry.list.cancel') }}
        </t-button>
        <t-button theme="primary" :loading="connectionSaving" @click="saveConnection">
          {{ t('registry.list.save') }}
        </t-button>
      </template>
    </responsive-dialog>

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

    <paged-multi-select
      v-model:visible="batchRepositoryAssignmentDialogVisible"
      v-model:current="candidatePagination.current"
      v-model:keyword="candidateSearch"
      v-model:page-size="candidatePagination.pageSize"
      v-model:selection="batchAssignmentSelection"
      :cancel-label="t('registry.list.cancel')"
      :cell-slot-names="['authorizationState']"
      :columns="candidateColumns"
      :confirm-label="t('registry.list.batchAuthorizeRepositories')"
      :confirm-loading="batchRepositoryAssignmentSaving"
      :empty-description="t('registry.list.candidatesEmpty')"
      :empty-title="t('registry.list.candidatesEmpty')"
      :loading="candidateLoading"
      row-key="id"
      :rows="batchAssignmentCandidates"
      :search="{
        label: t('registry.list.candidateSearch'),
        placeholder: t('registry.list.candidateSearchPlaceholder'),
      }"
      :selected-count-label="(count) => t('registry.list.candidateSelected', { count })"
      :title="t('registry.list.batchAuthorizeRepositoriesTitle')"
      :total="candidateTotal"
      @cancel="closeBatchRepositoryAssignments"
      @confirm="grantSelectedRepositoryAssignments"
      @page-change="loadBatchAssignmentCandidates"
      @search="searchBatchAssignmentCandidates"
    >
      <template #authorizationState="{ row }">
        <t-tag :theme="candidateAuthorizationTheme(row.authorization_state)" variant="light">
          {{ candidateAuthorizationLabel(row.authorization_state) }}
        </t-tag>
      </template>
    </paged-multi-select>

    <paged-multi-select
      v-model:visible="assignmentDialogVisible"
      v-model:current="assignmentCandidatePagination.current"
      v-model:keyword="assignmentCandidateSearch"
      v-model:page-size="assignmentCandidatePagination.pageSize"
      v-model:selection="assignmentSelection"
      :cancel-label="t('registry.list.cancel')"
      :cell-slot-names="['status']"
      :columns="assignmentCandidateColumns"
      :confirm-label="t('registry.list.save')"
      :confirm-loading="assignmentSaving"
      confirm-without-selection
      :empty-description="t('registry.list.candidatesEmpty')"
      :empty-title="t('registry.list.candidatesEmpty')"
      :loading="assignmentCandidateLoading"
      row-key="id"
      :rows="assignmentCandidates"
      :search="{
        label: t('registry.list.candidateSearch'),
        placeholder: t('registry.list.candidateSearchPlaceholder'),
      }"
      :selected-count-label="(count) => t('registry.list.candidateSelected', { count })"
      :title="t('registry.list.authorizeRepositoryTitle')"
      :total="assignmentCandidateTotal"
      @cancel="closeAssignments"
      @confirm="saveAssignments"
      @page-change="loadRepositoryAssignmentCandidates"
      @search="searchRepositoryAssignmentCandidates"
    >
      <template #status="{ row }">
        <t-tag :theme="row.status === 'enabled' ? 'success' : 'default'" variant="light">
          {{ t(`registry.list.userStatus.${row.status}`) }}
        </t-tag>
      </template>
    </paged-multi-select>
  </section>
</template>
<script setup lang="ts">
// Registry 详情页拥有仓库路径与使用授权操作；连接列表只负责导航和连接级操作。
import type { FormInstanceFunctions, TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import { ManagementBatchBar, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';
import { createExplicitSelection, type ExplicitSelection, PagedMultiSelect } from '@/shared/components/selection';
import { useViewportResponsiveVariant } from '@/shared/composables';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { buildDetailTitleWithFallback } from '@/utils/route/title';

import {
  addRegistryRepositoryAssignments,
  createRegistryRepository,
  deleteRegistryRepository,
  getRegistry,
  getRegistryRepositories,
  getRegistryRepositoryAssignmentCandidates,
  getRegistryRepositoryAssignments,
  replaceRegistryRepositoryAssignments,
  updateRegistry,
  updateRegistryRepository,
} from '../../api/registry';
import RegistryConnectionForm, { type RegistryConnectionFormData } from '../../components/RegistryConnectionForm.vue';
import { REGISTRY_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { REGISTRY_DETAIL_MODE, registryDetailPath } from '../../contract/paths';

type RegistryConnection = components['schemas']['registry-connection'];
type RegistryRepository = components['schemas']['registry-artifact-repository'];
type RegistryAssignmentCandidate = components['schemas']['registry-repository-assignment-candidate'];

const { t } = useI18n();
const viewportVariant = useViewportResponsiveVariant();
const isCompactDensity = computed(() => viewportVariant.value.density === 'compact');
const route = useRoute();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const connectionRef = computed(() => String(route.params.connectionRef || ''));
const connectionEditVisible = computed(() => route.query.mode === REGISTRY_DETAIL_MODE.EDIT);
const connection = ref<RegistryConnection | null>(null);
const repositories = ref<RegistryRepository[]>([]);
const loading = ref(false);
const connectionSaving = ref(false);
const connectionEditClosing = ref(false);
const connectionFormRef = ref<{ validate: () => ReturnType<FormInstanceFunctions['validate']> } | null>(null);
const connectionForm = ref<RegistryConnectionFormData>({
  connection_ref: '',
  display_name: '',
  endpoint: '',
  enabled: true,
  insecure: false,
  description: '',
});
const repositoryLoading = ref(false);
const repositorySaving = ref(false);
const repositoryPagination = ref({ current: 1, pageSize: 20 });
const repositoryTotal = ref(0);
const assignmentSaving = ref(false);
const batchRepositoryAssignmentSaving = ref(false);
const candidateLoading = ref(false);
const batchAssignmentCandidates = ref<RegistryAssignmentCandidate[]>([]);
const assignmentCandidates = ref<RegistryAssignmentCandidate[]>([]);
const candidateTotal = ref(0);
const candidateSearch = ref('');
const candidatePagination = ref({ current: 1, pageSize: 20 });
const batchAssignmentSelection = ref<ExplicitSelection<number>>(createExplicitSelection());
const assignmentCandidateLoading = ref(false);
const assignmentCandidateTotal = ref(0);
const assignmentCandidateSearch = ref('');
const assignmentCandidatePagination = ref({ current: 1, pageSize: 20 });
const assignmentSelection = ref<ExplicitSelection<number>>(createExplicitSelection());
const initialAssignmentUserIds = ref(new Set<number>());
const selectedRepositoryRefs = ref<Array<string | number>>([]);
const errorMessage = ref('');
const repositoryDrawerVisible = ref(false);
const assignmentDialogVisible = ref(false);
const batchRepositoryAssignmentDialogVisible = ref(false);
const editingRepositoryRef = ref('');
const assignmentRepositoryRef = ref('');
const repositoryForm = ref({
  repository_ref: '',
  display_name: '',
  allow_pull: true,
  allow_push: true,
  grant_creator_use: true,
});

const repositoryColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'repository_ref', title: t('registry.list.form.repositoryRef'), minWidth: 240 },
  { colKey: 'display_name', title: t('registry.list.form.repositoryName'), minWidth: 160 },
  { colKey: 'repositoryActions', title: t('registry.list.columns.actions'), width: 280 },
]);
const assignmentCandidateColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'display', title: t('registry.list.columns.candidateUser'), minWidth: 180 },
  { colKey: 'username', title: t('registry.list.columns.candidateUsername'), minWidth: 150 },
  { colKey: 'status', title: t('registry.list.columns.status'), width: 120 },
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

watch(connectionEditVisible, (visible) => {
  if (!visible) {
    connectionEditClosing.value = false;
    return;
  }

  if (connection.value) hydrateConnectionForm(connection.value);
});

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
    updateCurrentTabTitle(connectionResult.display_name);
    if (connectionEditVisible.value) hydrateConnectionForm(connectionResult);
    repositories.value = repositoryResult.items ?? [];
    repositoryTotal.value = repositoryResult.total ?? repositories.value.length;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function updateCurrentTabTitle(name: string) {
  tabsRouterStore.updateActiveTabTitle(
    REGISTRY_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    route,
    buildDetailTitleWithFallback('registry.route.detail.title', name),
  );
}

function hydrateConnectionForm(value: RegistryConnection) {
  connectionForm.value = {
    connection_ref: value.connection_ref,
    display_name: value.display_name,
    endpoint: value.endpoint,
    enabled: value.enabled,
    insecure: value.insecure,
    description: value.description ?? '',
  };
}

function openConnectionEdit() {
  void router.push({ query: { ...route.query, mode: REGISTRY_DETAIL_MODE.EDIT } });
}

function closeConnectionEdit() {
  if (connectionSaving.value || connectionEditClosing.value || !connectionEditVisible.value) return;
  connectionEditClosing.value = true;
  void router.replace({
    path: registryDetailPath(connectionRef.value),
    query: { ...route.query, mode: undefined },
  });
}

function handleConnectionEditVisible(visible: boolean) {
  if (!visible) closeConnectionEdit();
}

async function saveConnection() {
  if (!connectionRef.value || (await connectionFormRef.value?.validate()) !== true) return;
  connectionSaving.value = true;
  try {
    await updateRegistry(connectionRef.value, {
      display_name: connectionForm.value.display_name,
      endpoint: connectionForm.value.endpoint,
      enabled: connectionForm.value.enabled,
      insecure: connectionForm.value.insecure,
      description: connectionForm.value.description || null,
    });
    MessagePlugin.success(t('registry.route.detail.connectionSaveSuccess'));
    await load();
    await router.replace({
      path: registryDetailPath(connectionRef.value),
      query: { ...route.query, mode: undefined },
    });
  } catch (error) {
    const message = resolveLocalizedErrorMessage(t, error, t('registry.route.detail.connectionSaveFailed'));
    errorMessage.value = message;
    MessagePlugin.error(message);
  } finally {
    connectionSaving.value = false;
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

function repositoryCardActions(_repository: RegistryRepository) {
  return [
    { label: t('registry.list.edit'), value: 'edit' },
    { label: t('registry.list.assignments'), value: 'assignments' },
  ] as const;
}

function handleRepositoryCardAction(
  repository: RegistryRepository,
  action: ReturnType<typeof repositoryCardActions>[number]['value'],
) {
  if (action === 'edit') {
    openRepository(repository);
    return;
  }
  void openAssignments(repository.repository_ref);
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
  candidateSearch.value = '';
  candidatePagination.value.current = 1;
  batchAssignmentSelection.value = createExplicitSelection();
  batchRepositoryAssignmentDialogVisible.value = true;
  void loadBatchAssignmentCandidates();
}

function closeBatchRepositoryAssignments() {
  if (batchRepositoryAssignmentSaving.value) return;
  batchRepositoryAssignmentDialogVisible.value = false;
  batchAssignmentSelection.value = createExplicitSelection();
}

async function grantSelectedRepositoryAssignments() {
  if (!connectionRef.value || !selectedRepositoryRefs.value.length) return;
  const userIds = normalizeSelectedUserIds(batchAssignmentSelection.value);
  if (!userIds.length) return;

  batchRepositoryAssignmentSaving.value = true;
  try {
    await addRegistryRepositoryAssignments(connectionRef.value, {
      repository_refs: selectedRepositoryRefs.value.map(String),
      user_ids: userIds,
    });
    closeBatchRepositoryAssignments();
    selectedRepositoryRefs.value = [];
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    batchRepositoryAssignmentSaving.value = false;
  }
}

async function loadBatchAssignmentCandidates() {
  if (!connectionRef.value || !selectedRepositoryRefs.value.length) return;
  candidateLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignmentCandidates(connectionRef.value, {
      repository_ref: selectedRepositoryRefs.value.map(String),
      search: candidateSearch.value.trim() || undefined,
      limit: candidatePagination.value.pageSize,
      offset: (candidatePagination.value.current - 1) * candidatePagination.value.pageSize,
    });
    batchAssignmentCandidates.value = response.items ?? [];
    candidateTotal.value = response.total ?? 0;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    candidateLoading.value = false;
  }
}

function searchBatchAssignmentCandidates() {
  candidatePagination.value.current = 1;
  void loadBatchAssignmentCandidates();
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
  assignmentCandidateSearch.value = '';
  assignmentCandidatePagination.value.current = 1;
  assignmentSelection.value = createExplicitSelection();
  initialAssignmentUserIds.value = new Set();
  assignmentDialogVisible.value = true;
  await Promise.all([loadRepositoryAssignmentCandidates(), loadInitialAssignmentSelection()]);
}

function closeAssignments() {
  if (assignmentSaving.value) return;
  assignmentDialogVisible.value = false;
  assignmentSelection.value = createExplicitSelection();
  initialAssignmentUserIds.value = new Set();
}

async function loadInitialAssignmentSelection() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  try {
    const userIds: number[] = [];
    let offset = 0;
    let total = Number.POSITIVE_INFINITY;
    while (offset < total) {
      const response = await getRegistryRepositoryAssignments(connectionRef.value, {
        repository_ref: assignmentRepositoryRef.value,
        limit: 100,
        offset,
      });
      const page = response.items ?? [];
      userIds.push(...page.map((item) => item.user_id));
      total = response.total ?? offset + page.length;
      offset += page.length;
      if (page.length === 0) break;
    }
    const selection = createExplicitSelection(userIds);
    initialAssignmentUserIds.value = new Set(selection.selectedIds);
    assignmentSelection.value = selection;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}

async function saveAssignments() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  const userIds = normalizeSelectedUserIds(assignmentSelection.value);
  if (sameUserIDSet(initialAssignmentUserIds.value, userIds)) {
    closeAssignments();
    return;
  }

  assignmentSaving.value = true;
  try {
    await replaceRegistryRepositoryAssignments(connectionRef.value, assignmentRepositoryRef.value, {
      user_ids: userIds,
    });
    MessagePlugin.success(t('registry.list.assignmentSaveSuccess'));
    closeAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}

async function loadRepositoryAssignmentCandidates() {
  if (!connectionRef.value || !assignmentRepositoryRef.value) return;
  assignmentCandidateLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignmentCandidates(connectionRef.value, {
      repository_ref: [assignmentRepositoryRef.value],
      search: assignmentCandidateSearch.value.trim() || undefined,
      limit: assignmentCandidatePagination.value.pageSize,
      offset: (assignmentCandidatePagination.value.current - 1) * assignmentCandidatePagination.value.pageSize,
    });
    assignmentCandidates.value = response.items ?? [];
    assignmentCandidateTotal.value = response.total ?? 0;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentCandidateLoading.value = false;
  }
}

function searchRepositoryAssignmentCandidates() {
  assignmentCandidatePagination.value.current = 1;
  void loadRepositoryAssignmentCandidates();
}

function normalizeSelectedUserIds(selection: ExplicitSelection) {
  return Array.from(selection.selectedIds)
    .map(Number)
    .filter((id) => Number.isInteger(id) && id > 0);
}

function sameUserIDSet(initial: Set<number>, current: number[]) {
  return initial.size === current.length && current.every((userID) => initial.has(userID));
}
</script>
<style scoped lang="less">
.registry-detail__section {
  margin-top: var(--graft-density-gap-16);
}

.registry-detail__repository-list {
  padding-top: var(--graft-density-gap-12);
}

.registry-detail__repository-card {
  border-bottom: 1px solid var(--td-component-stroke);
  display: grid;
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-12) 0;
}

.registry-detail__repository-card-main {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.registry-detail__repository-card-main span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.registry-detail__repository-card-permissions,
.registry-detail__repository-card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}
</style>
