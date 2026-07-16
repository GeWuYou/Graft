<template>
  <div class="application-template-detail" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.templateDetail.title"
        :description="templateRecord?.description || t('project.templates.detailDescription')"
        :source="{ labelKey: 'project.templates.eyebrow', fallback: t('project.templates.eyebrow') }"
      >
        <template #actions>
          <t-space size="small">
            <t-button variant="outline" @click="backToList">{{ t('project.templates.back') }}</t-button>
            <t-button v-if="isPublished" variant="outline" :loading="deriving" @click="deriveDraft">{{
              t('project.templates.derive')
            }}</t-button>
            <t-button v-if="isDraft" theme="primary" :loading="saving" @click="saveDraft">{{
              t('project.templates.save')
            }}</t-button>
            <t-button v-if="isDraft" theme="success" :loading="publishing" @click="publishDraft">{{
              t('project.templates.publish')
            }}</t-button>
            <t-button v-if="!isArchived" theme="danger" variant="outline" @click="archiveVisible = true">{{
              t('project.templates.archive')
            }}</t-button>
          </t-space>
        </template>
        <template #meta>
          <t-space size="small">
            <t-tag :theme="statusTheme" variant="light-outline">{{ statusLabel }}</t-tag>
            <t-tag variant="light-outline">{{ templateRecord?.deployment_adapter_kind || 'compose' }}</t-tag>
            <t-tag variant="light-outline"
              >{{ t('project.templates.version') }}
              {{ templateRecord ? `v${templateRecord.version.version_number}` : '-' }}</t-tag
            >
          </t-space>
        </template>
      </management-page-header>

      <t-loading :loading="loading">
        <t-alert v-if="errorMessage" theme="error" :message="errorMessage"
          ><template #operation
            ><t-button theme="danger" variant="text" @click="loadTemplate">{{
              t('project.templates.retry')
            }}</t-button></template
          ></t-alert
        >
        <template v-else-if="templateRecord">
          <t-alert
            v-if="isPublished"
            theme="info"
            :message="t('project.templates.publishedImmutableHint')"
            class="application-template-detail__notice"
          />
          <t-alert
            v-if="isArchived"
            theme="warning"
            :message="t('project.templates.archivedHint')"
            class="application-template-detail__notice"
          />
          <t-card bordered class="application-template-detail__section">
            <t-form label-align="top">
              <div class="application-template-detail__form-grid">
                <t-form-item :label="t('project.templates.name')"
                  ><t-input v-model="displayName" :disabled="!isDraft"
                /></t-form-item>
                <t-form-item :label="t('project.templates.adapter')"
                  ><t-input :value="templateRecord.deployment_adapter_kind" disabled
                /></t-form-item>
              </div>
              <t-form-item :label="t('project.templates.descriptionField')"
                ><t-textarea v-model="description" :disabled="!isDraft" :autosize="{ minRows: 2, maxRows: 4 }"
              /></t-form-item>
            </t-form>
          </t-card>

          <section class="application-template-detail__section">
            <project-create-workspace-editor
              v-model:files="workspaceFiles"
              :class="{ 'application-template-detail__readonly': !isDraft }"
            />
          </section>

          <section v-if="lifecycleDraft" class="application-template-detail__section">
            <project-lifecycle-configuration-review
              v-model:draft="lifecycleDraft"
              :title="t('project.templates.lifecycleTitle')"
              :description="t('project.templates.lifecycleDescription')"
              :authority-message="t('project.templates.lifecycleAuthority')"
              :configuration-title="t('project.create.lifecycle.configurationTitle')"
              :command-preview-title="t('project.create.lifecycle.commandPreviewTitle')"
            />
          </section>
        </template>
      </t-loading>
    </management-page-content>

    <t-dialog
      v-model:visible="archiveVisible"
      theme="warning"
      :header="t('project.templates.archiveTitle')"
      :body="t('project.templates.archiveBody')"
      :confirm-btn="t('project.templates.archiveConfirm')"
      :cancel-btn="t('project.templates.cancel')"
      :confirm-loading="archiving"
      @confirm="archiveTemplate"
    />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  getApplicationTemplate,
  postApplicationTemplateArchive,
  postApplicationTemplateDerive,
  postApplicationTemplatePublish,
  putApplicationTemplate,
} from '../../api/project';
import ProjectCreateWorkspaceEditor from '../../components/ProjectCreateWorkspaceEditor.vue';
import ProjectLifecycleConfigurationReview from '../../components/ProjectLifecycleConfigurationReview.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  buildBlankLifecycleConfigurationDraft,
  buildLifecycleConfigurationRequest,
  type LifecycleDraftSource,
} from '../../shared/lifecycle';
import type {
  ApplicationLifecycleConfigurationDraft,
  ApplicationTemplate,
  ApplicationWorkspaceDraftEntry,
} from '../../types/project';

defineOptions({ name: 'ApplicationTemplateDetailIndex' });

// 详情页将 adapter-owned definition 映射为共享工作区和生命周期编辑器，提交时仍保留单一模板版本写入边界。

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const templateRecord = ref<ApplicationTemplate | null>(null);
const loading = ref(false);
const saving = ref(false);
const publishing = ref(false);
const deriving = ref(false);
const archiving = ref(false);
const archiveVisible = ref(false);
const errorMessage = ref('');
const displayName = ref('');
const description = ref('');
const workspaceFiles = ref<ApplicationWorkspaceDraftEntry[]>([]);
const composeFilePath = ref('compose.yaml');
const lifecycleDraft = ref<ApplicationLifecycleConfigurationDraft | null>(null);
const templateId = computed(() => String(route.params.templateId || ''));
const isArchived = computed(() => Boolean(templateRecord.value?.archived_at));
const isDraft = computed(() => templateRecord.value?.version.status === 'draft' && !isArchived.value);
const isPublished = computed(() => templateRecord.value?.version.status === 'published' && !isArchived.value);
const statusLabel = computed(() =>
  isArchived.value
    ? t('project.templates.statusArchived')
    : isDraft.value
      ? t('project.templates.statusDraft')
      : t('project.templates.statusPublished'),
);
const statusTheme = computed(() => (isArchived.value ? 'default' : isDraft.value ? 'warning' : 'success'));

watch(templateId, () => void loadTemplate(), { immediate: true });

async function loadTemplate() {
  if (!templateId.value) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    hydrate(await getApplicationTemplate(templateId.value));
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.templates.detailLoadFailed'));
  } finally {
    loading.value = false;
  }
}

function hydrate(template: ApplicationTemplate) {
  templateRecord.value = template;
  displayName.value = template.display_name;
  description.value = template.description || '';
  const definition = template.version.definition as Record<string, unknown>;
  composeFilePath.value =
    typeof definition.compose_file_path === 'string' ? definition.compose_file_path : 'compose.yaml';
  workspaceFiles.value = Array.isArray(definition.workspace_entries)
    ? definition.workspace_entries.map((entry) => ({
        path: String((entry as Record<string, unknown>).path || ''),
        node_type: (entry as Record<string, unknown>).node_type === 'directory' ? 'directory' : 'file',
        content: String((entry as Record<string, unknown>).content || ''),
      }))
    : [];
  lifecycleDraft.value = buildBlankLifecycleConfigurationDraft(
    { lifecycle_configuration: (definition.lifecycle_configuration || {}) as LifecycleDraftSource },
    { composeFilePath: composeFilePath.value, composeProjectName: 'template' },
  );
}

function draftPayload() {
  if (!lifecycleDraft.value) throw new Error('template lifecycle draft is unavailable');
  return {
    display_name: displayName.value.trim(),
    description: description.value.trim(),
    deployment_adapter_kind: 'compose' as const,
    definition_schema_version: templateRecord.value?.version.definition_schema_version || 1,
    definition: {
      compose_file_path: composeFilePath.value,
      workspace_entries: workspaceFiles.value.map((entry) =>
        entry.node_type === 'directory'
          ? { path: entry.path, node_type: 'directory' }
          : { path: entry.path, node_type: 'file', content: entry.content },
      ),
      lifecycle_configuration: buildLifecycleConfigurationRequest(lifecycleDraft.value),
    },
  } as never;
}

async function saveDraft(): Promise<boolean> {
  if (!isDraft.value || !templateRecord.value) return false;
  saving.value = true;
  try {
    hydrate(await putApplicationTemplate(templateRecord.value.template_id, draftPayload()));
    MessagePlugin.success(t('project.templates.saveSuccess'));
    return true;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.saveFailed')));
    return false;
  } finally {
    saving.value = false;
  }
}

async function publishDraft() {
  if (!templateRecord.value) return;
  if (!(await saveDraft()) || !isDraft.value) return;
  publishing.value = true;
  try {
    hydrate(await postApplicationTemplatePublish(templateRecord.value.template_id));
    MessagePlugin.success(t('project.templates.publishSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.publishFailed')));
  } finally {
    publishing.value = false;
  }
}

async function deriveDraft() {
  if (!templateRecord.value) return;
  deriving.value = true;
  try {
    hydrate(
      await postApplicationTemplateDerive(
        templateRecord.value.template_id,
        templateRecord.value.version.template_version_id,
      ),
    );
    MessagePlugin.success(t('project.templates.deriveSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.deriveFailed')));
  } finally {
    deriving.value = false;
  }
}

async function archiveTemplate() {
  if (!templateRecord.value) return;
  archiving.value = true;
  try {
    await postApplicationTemplateArchive(templateRecord.value.template_id);
    archiveVisible.value = false;
    await loadTemplate();
    MessagePlugin.success(t('project.templates.archiveSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.archiveFailed')));
  } finally {
    archiving.value = false;
  }
}

function backToList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATES.pageRouteName });
}
</script>
<style scoped>
.application-template-detail__section,
.application-template-detail__notice {
  margin-top: var(--graft-density-gap-16);
}

.application-template-detail__form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.application-template-detail__readonly {
  opacity: 0.72;
  pointer-events: none;
}

@media (width <= 720px) {
  .application-template-detail__form-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
