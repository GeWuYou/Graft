import { computed, ref } from 'vue';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectImportExecute, postProjectImportRuntimeInspect } from '../api/import';
import type {
  ProjectImportInspectResponse,
  ProjectImportRuntimeCandidate,
  ProjectImportRuntimeInspectRequest,
} from '../types/import';
import type { ProjectLifecycleConfigurationDraft } from '../types/project';
import { buildSuggestedDisplayName, hasBlockingImportConflicts, normalizeProjectImportInspectResponse } from './import';
import { buildImportLifecycleConfigurationDraft, buildLifecycleConfigurationRequest } from './lifecycle';

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
  const lifecycleDraft = ref<ProjectLifecycleConfigurationDraft | null>(null);
  const lifecycleConfigError = ref('');

  const canImport = computed(
    () =>
      Boolean(inspectResult.value?.inspection_id) &&
      !inspectLoading.value &&
      !importLoading.value &&
      !hasBlockingImportConflicts(inspectResult.value),
  );

  const hasPreview = computed(() => Boolean(inspectResult.value));

  /**
   * 将项目导入流程恢复到初始状态。
   */
  function reset() {
    latestInspectRequestId += 1;
    selectedCandidateKey.value = '';
    inspectLoading.value = false;
    importLoading.value = false;
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    displayName.value = '';
    canonicalProjectNameOverride.value = '';
    lifecycleDraft.value = null;
    lifecycleConfigError.value = '';
  }

  /**
   * 清除当前项目导入预览及其相关状态。
   */
  function clearPreview() {
    inspectError.value = '';
    importError.value = '';
    inspectResult.value = null;
    displayName.value = '';
    canonicalProjectNameOverride.value = '';
    lifecycleDraft.value = null;
    lifecycleConfigError.value = '';
  }

  /**
   * 检查指定的运行时候选项并更新项目导入预览。
   *
   * @param candidateKey - 要检查的运行时候选项标识
   * @returns 检查结果状态：`'applied'` 表示结果已应用，`'stale'` 表示结果已过期
   */
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
      const normalizedResponse = normalizeProjectImportInspectResponse(response);
      inspectResult.value = normalizedResponse;
      displayName.value = normalizedResponse ? buildSuggestedDisplayName(normalizedResponse) : '';
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

  /**
   * 刷新当前选中候选运行时的检查结果。
   *
   * @returns 当前未选择候选运行时时为 `'idle'`，否则为检查请求的执行状态
   */
  async function refreshInspect() {
    if (!selectedCandidateKey.value) {
      return 'idle' as const;
    }

    const result = await inspectCandidateByKey(selectedCandidateKey.value);
    if (result === 'applied' && canImport.value) {
      prepareLifecycleConfiguration();
    }
    return result;
  }

  /**
   * 根据当前检查结果准备生命周期配置草稿。
   *
   * @returns 成功生成配置草稿时为 `true`，否则为 `false`
   */
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

  /**
   * 提交项目导入请求。
   *
   * @returns 导入执行请求的结果
   * @throws 当缺少检查标识或生命周期配置时抛出错误；请求执行失败时重新抛出原始错误
   */
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
      return await postProjectImportExecute({
        inspection_id: inspectResult.value.inspection_id,
        display_name: displayName.value.trim() || undefined,
        canonical_project_name_override: canonicalProjectNameOverride.value.trim() || null,
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
    canonicalProjectNameOverride,
    clearPreview,
    displayName,
    hasPreview,
    importError,
    importLoading,
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
