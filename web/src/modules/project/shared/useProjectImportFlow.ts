import { computed, ref } from 'vue';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectImportExecute, postProjectImportRuntimeInspect } from '../api/import';
import type {
  ProjectImportInspectResponse,
  ProjectImportRuntimeCandidate,
  ProjectImportRuntimeInspectRequest,
} from '../types/import';
import { buildSuggestedDisplayName, hasBlockingImportConflicts } from './import';

type Translate = (key: string, params?: Record<string, unknown>) => string;

/**
 * 管理项目导入流程的候选 inspect、预览检查和导入提交状态。
 *
 * @param t - 用于生成本地化错误消息的翻译函数
 * @returns 包含导入流程状态、计算结果和操作方法的对象
 */
export function useProjectImportFlow(t: Translate) {
  let latestInspectRequestId = 0;

  const selectedCandidateKey = ref('');
  const inspectLoading = ref(false);
  const importLoading = ref(false);
  const inspectError = ref('');
  const importError = ref('');
  const inspectResult = ref<ProjectImportInspectResponse | null>(null);
  const displayName = ref('');
  const canonicalProjectNameOverride = ref('');

  const canImport = computed(
    () =>
      Boolean(inspectResult.value?.inspection_id) &&
      !inspectLoading.value &&
      !importLoading.value &&
      !hasBlockingImportConflicts(inspectResult.value),
  );

  const hasPreview = computed(() => Boolean(inspectResult.value));

  function reset() {
    selectedCandidateKey.value = '';
    inspectLoading.value = false;
    importLoading.value = false;
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    displayName.value = '';
    canonicalProjectNameOverride.value = '';
  }

  function clearPreview() {
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    displayName.value = '';
    canonicalProjectNameOverride.value = '';
  }

  async function inspectCandidateByKey(candidateKey: string) {
    const requestId = ++latestInspectRequestId;
    selectedCandidateKey.value = candidateKey;
    clearPreview();
    inspectLoading.value = true;
    try {
      const payload: ProjectImportRuntimeInspectRequest = { candidate_key: candidateKey };
      const response = await postProjectImportRuntimeInspect(payload);
      if (requestId !== latestInspectRequestId) {
        return 'stale' as const;
      }
      inspectResult.value = response;
      displayName.value = buildSuggestedDisplayName(response);
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

  async function inspectCandidate(candidate: ProjectImportRuntimeCandidate) {
    return inspectCandidateByKey(candidate.candidate_key);
  }

  async function refreshInspect() {
    if (!selectedCandidateKey.value) {
      return 'idle' as const;
    }

    return inspectCandidateByKey(selectedCandidateKey.value);
  }

  async function submitImport() {
    if (!inspectResult.value?.inspection_id) {
      throw new Error('missing inspection authority');
    }

    importLoading.value = true;
    importError.value = '';
    try {
      return await postProjectImportExecute({
        inspection_id: inspectResult.value.inspection_id,
        display_name: displayName.value.trim() || undefined,
        canonical_project_name_override: canonicalProjectNameOverride.value.trim() || null,
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
    canonicalProjectNameOverride,
    clearPreview,
    displayName,
    hasPreview,
    importError,
    importLoading,
    inspectCandidate,
    inspectError,
    inspectLoading,
    inspectResult,
    refreshInspect,
    reset,
    selectedCandidateKey,
    submitImport,
  };
}
