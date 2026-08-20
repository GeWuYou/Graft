<template>
  <paged-multi-select
    :visible="visible"
    :current="pagination.current"
    :keyword="search"
    :page-size="pagination.pageSize"
    :selection="selection"
    :cancel-label="t('runtimeTarget.detail.cancel')"
    :cell-slot-names="['authorizationState']"
    :columns="columns"
    :confirm-label="t('runtimeTarget.detail.saveAuthorizedUsers')"
    :confirm-loading="saving || loading"
    confirm-without-selection
    :empty-description="t('runtimeTarget.detail.candidatesEmpty')"
    :empty-title="t('runtimeTarget.detail.candidatesEmpty')"
    :error-message="errorMessage"
    :loading="loading"
    row-key="id"
    :rows="candidates"
    :search="{
      placeholder: t('runtimeTarget.detail.candidateSearchPlaceholder'),
      clearLabel: t('runtimeTarget.detail.clearSearch'),
    }"
    :selected-count-label="(count) => t('runtimeTarget.detail.selectedUsers', { count })"
    :title="t('runtimeTarget.detail.selectAuthorizedUsersTitle')"
    :total="total"
    :total-label="(count) => t('runtimeTarget.detail.candidateTotal', { count })"
    @cancel="close"
    @confirm="save"
    @page-change="loadCandidates"
    @search="searchCandidates"
    @update:current="pagination.current = $event"
    @update:keyword="search = $event"
    @update:page-size="pagination.pageSize = $event"
    @update:selection="selection = $event"
    @update:visible="handleVisibleChange"
  >
    <template #authorizationState="{ row }">
      <t-tag v-if="initialUserIds.has(Number(row.id))" theme="success" variant="light">
        {{ t('runtimeTarget.detail.alreadyAuthorized') }}
      </t-tag>
      <span v-else class="runtime-target-assignment-state-empty">-</span>
    </template>
  </paged-multi-select>
</template>
<script setup lang="ts">
// 单目标授权弹窗统一管理候选分页、revision 与整体替换，列表页和详情页只负责选择目标及消费保存结果。
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { createExplicitSelection, type ExplicitSelection, PagedMultiSelect } from '@/shared/components/selection';
import { isApiRequestError } from '@/utils/request';

import {
  getRuntimeTargetAssignmentCandidates,
  getRuntimeTargetAssignments,
  replaceRuntimeTargetAssignments,
  type RuntimeTargetAssignment,
  type RuntimeTargetAssignmentCandidate,
} from '../api/runtime-target';

const props = defineProps<{ visible: boolean; targetId?: number | null }>();
const emit = defineEmits<{
  'update:visible': [value: boolean];
  saved: [value: { items: RuntimeTargetAssignment[]; revision: number }];
}>();
const { t } = useI18n();
const candidates = ref<RuntimeTargetAssignmentCandidate[]>([]);
const total = ref(0);
const search = ref('');
const pagination = reactive({ current: 1, pageSize: 20 });
const selection = ref<ExplicitSelection>(createExplicitSelection<number>());
const initialUserIds = ref(new Set<number>());
const revision = ref(1);
const loading = ref(false);
const saving = ref(false);
const errorMessage = ref('');
let requestVersion = 0;
const columns = computed(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'display', title: t('runtimeTarget.detail.candidateUser'), minWidth: 180 },
  { colKey: 'username', title: t('runtimeTarget.detail.candidateUsername'), minWidth: 150 },
  { colKey: 'authorizationState', title: t('runtimeTarget.detail.authorizationState'), width: 120 },
]);

function isValidTargetId(value: number | null | undefined): value is number {
  return Number.isInteger(value) && Number(value) > 0;
}

async function open() {
  if (!isValidTargetId(props.targetId)) return;
  const version = ++requestVersion;
  search.value = '';
  pagination.current = 1;
  errorMessage.value = '';
  initialUserIds.value = new Set();
  candidates.value = [];
  total.value = 0;
  loading.value = true;
  try {
    const [assignments, candidatePage] = await Promise.all([
      getRuntimeTargetAssignments(props.targetId),
      getRuntimeTargetAssignmentCandidates(props.targetId, {
        limit: pagination.pageSize,
        offset: 0,
      }),
    ]);
    if (version !== requestVersion || !props.visible) return;
    revision.value = assignments.revision;
    initialUserIds.value = new Set(assignments.items.map((item) => item.user_id));
    selection.value = createExplicitSelection(assignments.items.map((item) => item.user_id));
    candidates.value = candidatePage.items;
    total.value = candidatePage.total;
  } catch {
    if (version === requestVersion) errorMessage.value = t('runtimeTarget.detail.authorizedUsersLoadError');
  } finally {
    if (version === requestVersion) loading.value = false;
  }
}

async function loadCandidates() {
  if (!isValidTargetId(props.targetId)) return;
  const version = ++requestVersion;
  loading.value = true;
  errorMessage.value = '';
  try {
    const result = await getRuntimeTargetAssignmentCandidates(props.targetId, {
      search: search.value.trim() || undefined,
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    });
    if (version !== requestVersion || !props.visible) return;
    candidates.value = result.items;
    total.value = result.total;
  } catch {
    if (version === requestVersion) errorMessage.value = t('runtimeTarget.detail.authorizedUsersLoadError');
  } finally {
    if (version === requestVersion) loading.value = false;
  }
}

function searchCandidates() {
  pagination.current = 1;
  void loadCandidates();
}

function handleVisibleChange(value: boolean) {
  if (!value) close();
}

function close() {
  if (saving.value) return;
  requestVersion += 1;
  emit('update:visible', false);
  selection.value = createExplicitSelection([...initialUserIds.value]);
}

async function save() {
  if (!isValidTargetId(props.targetId)) return;
  const selectedUserIds = [...selection.value.selectedIds].map(Number);
  if (
    selectedUserIds.length === initialUserIds.value.size &&
    selectedUserIds.every((id) => initialUserIds.value.has(id))
  ) {
    close();
    return;
  }
  saving.value = true;
  errorMessage.value = '';
  try {
    const result = await replaceRuntimeTargetAssignments(props.targetId, selectedUserIds, revision.value);
    revision.value = result.revision;
    initialUserIds.value = new Set(result.items.map((item) => item.user_id));
    selection.value = createExplicitSelection(result.items.map((item) => item.user_id));
    emit('saved', result);
    MessagePlugin.success(t('runtimeTarget.detail.authorizedUsersSaveSuccess'));
    requestVersion += 1;
    emit('update:visible', false);
  } catch (error) {
    if (isApiRequestError(error) && error.status === 409) {
      await open();
      errorMessage.value = t('runtimeTarget.detail.authorizationConflict');
    } else {
      errorMessage.value = t('runtimeTarget.detail.authorizedUsersSaveError');
    }
  } finally {
    saving.value = false;
  }
}

watch(
  () => [props.visible, props.targetId] as const,
  ([visible]) => {
    if (visible) void open();
  },
  { immediate: true },
);
</script>
<style scoped lang="less">
.runtime-target-assignment-state-empty {
  color: var(--td-text-color-placeholder);
}
</style>
