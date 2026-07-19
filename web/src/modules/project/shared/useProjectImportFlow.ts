import { computed, ref } from 'vue';

import { PROJECT_ERROR_CODE } from '@/contracts/generated/modules/project';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postApplicationImportExecute, postApplicationImportRuntimeInspect } from '../api/import';
import type {
  ApplicationImportInspectResponse,
  ApplicationImportRuntimeCandidate,
  ApplicationImportRuntimeInspectRequest,
} from '../types/import';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import {
  buildSuggestedDisplayName,
  hasBlockingImportConflicts,
  normalizeApplicationImportInspectResponse,
} from './import';
import { buildImportLifecycleConfigurationDraft, buildLifecycleConfigurationRequest } from './lifecycle';

type Translate = (key: string, params?: Record<string, unknown>) => string;

/**
 * 判断导入提交是否因服务端检查会话失效而失败。
 */
export function isApplicationImportInspectionExpiredError(error: unknown) {
  return (
    Boolean(
      error && typeof error === 'object' && (error as { isApiRequestError?: unknown }).isApiRequestError === true,
    ) && (error as { messageKey?: unknown }).messageKey === PROJECT_ERROR_CODE.INSPECTION_EXPIRED
  );
}

/** 导入检查结果属于服务端会话；异步结果只有在仍对应最新请求时才能写入页面状态。 */
export function useApplicationImportFlow(t: Translate) {
  let latestInspectRequestId = 0;

  const selectedCandidateKey = ref('');
  const inspectLoading = ref(false);
  const importLoading = ref(false);
  const inspectError = ref('');
  const importError = ref('');
  const inspectResult = ref<ApplicationImportInspectResponse | null>(null);
  const inspectionSessionValid = ref(false);
  const displayName = ref('');
  const composeProjectNameOverride = ref('');
  const lifecycleDraft = ref<ApplicationLifecycleConfigurationDraft | null>(null);
  const lifecycleConfigError = ref('');

  const canImport = computed(
    () =>
      Boolean(inspectResult.value?.inspection_id) &&
      inspectionSessionValid.value &&
      !inspectLoading.value &&
      !importLoading.value &&
      !hasBlockingImportConflicts(inspectResult.value),
  );

  const hasPreview = computed(() => Boolean(inspectResult.value));

  function reset() {
    latestInspectRequestId += 1;
    selectedCandidateKey.value = '';
    inspectLoading.value = false;
    importLoading.value = false;
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    inspectionSessionValid.value = false;
    displayName.value = '';
    composeProjectNameOverride.value = '';
    lifecycleDraft.value = null;
    lifecycleConfigError.value = '';
  }

  function clearPreview() {
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    inspectionSessionValid.value = false;
    displayName.value = '';
    composeProjectNameOverride.value = '';
    lifecycleDraft.value = null;
    lifecycleConfigError.value = '';
  }

  /** 过期请求只能返回状态，不能清除或覆盖较新检查产生的预览与错误。 */
  async function inspectCandidateByKey(candidateKey: string, preserveDraft = false) {
    const requestId = ++latestInspectRequestId;
    selectedCandidateKey.value = candidateKey;
    if (preserveDraft) {
      inspectError.value = '';
      importError.value = '';
      lifecycleConfigError.value = '';
    } else {
      clearPreview();
    }
    inspectLoading.value = true;
    try {
      const payload: ApplicationImportRuntimeInspectRequest = { candidate_key: candidateKey };
      const response = await postApplicationImportRuntimeInspect(payload);
      if (requestId !== latestInspectRequestId) {
        return 'stale' as const;
      }
      const normalizedResponse = normalizeApplicationImportInspectResponse(response);
      inspectResult.value = normalizedResponse;
      inspectionSessionValid.value = Boolean(normalizedResponse?.inspection_id);
      if (!preserveDraft) {
        displayName.value = normalizedResponse ? buildSuggestedDisplayName(normalizedResponse) : '';
      }
      return 'applied' as const;
    } catch (error) {
      if (requestId !== latestInspectRequestId) {
        return 'stale' as const;
      }
      inspectError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.inspectFailed'));
      throw error;
    } finally {
      if (requestId === latestInspectRequestId) {
        inspectLoading.value = false;
      }
    }
  }

  async function inspectCandidate(candidate: ApplicationImportRuntimeCandidate) {
    return inspectCandidateByKey(candidate.candidate_key);
  }

  async function refreshInspect() {
    if (!selectedCandidateKey.value) {
      return 'idle' as const;
    }

    const result = await inspectCandidateByKey(selectedCandidateKey.value, true);
    if (result === 'applied' && canImport.value) {
      prepareLifecycleConfiguration();
    }
    return result;
  }

  /** 编辑检查结果后立即撤销当前会话资格，避免提交过期快照。 */
  function invalidateInspectionSession() {
    inspectionSessionValid.value = false;
  }

  function prepareLifecycleConfiguration() {
    if (lifecycleDraft.value) {
      return true;
    }
    if (!inspectResult.value) {
      return false;
    }
    try {
      lifecycleDraft.value = buildImportLifecycleConfigurationDraft(inspectResult.value);
      lifecycleConfigError.value = '';
      return true;
    } catch {
      lifecycleDraft.value = null;
      lifecycleConfigError.value = t('project.import.messages.lifecycleConfigUnavailable');
      return false;
    }
  }

  /** 提交时再次要求检查会话和生命周期草稿，失败既保留本地错误又向调用方抛出原始异常。 */
  async function submitImport() {
    if (!inspectResult.value?.inspection_id) {
      throw new Error('missing inspection authority');
    }
    if (!lifecycleDraft.value) {
      throw new Error('missing lifecycle configuration');
    }

    importLoading.value = true;
    importError.value = '';
    try {
      return await postApplicationImportExecute({
        inspection_id: inspectResult.value.inspection_id,
        display_name: displayName.value.trim() || undefined,
        compose_project_name_override: composeProjectNameOverride.value.trim() || null,
        lifecycle_configuration: buildLifecycleConfigurationRequest(lifecycleDraft.value),
      });
    } catch (error) {
      importError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.importFailed'));
      throw error;
    } finally {
      importLoading.value = false;
    }
  }

  return {
    canImport,
    composeProjectNameOverride,
    clearPreview,
    displayName,
    hasPreview,
    importError,
    importLoading,
    invalidateInspectionSession,
    inspectionSessionValid,
    lifecycleConfigError,
    lifecycleDraft,
    inspectCandidate,
    inspectError,
    inspectLoading,
    inspectResult,
    prepareLifecycleConfiguration,
    refreshInspect,
    reset,
    selectedCandidateKey,
    submitImport,
  };
}
