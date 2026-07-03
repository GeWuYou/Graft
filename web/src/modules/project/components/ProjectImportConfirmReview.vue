<template>
  <section class="project-import-confirm">
    <project-import-section-heading
      :description="t('project.import.confirm.description')"
      :title="t('project.import.confirm.title')"
    />

    <div class="project-import-confirm__grid">
      <t-card :bordered="true" :title="t('project.import.confirm.identityTitle')">
        <div class="project-import-confirm__card-content">
          <t-alert theme="info" :message="t('project.import.confirm.identityHint')" />
          <t-alert v-if="importError" theme="error" :message="importError" />

          <t-form
            :id="formId"
            :data="formData"
            :rules="formRules"
            label-align="top"
            scroll-to-first-error="smooth"
            @submit="handleSubmit"
          >
            <div class="project-import-confirm__form-grid">
              <t-form-item :label="t('project.import.form.displayName')" name="display_name">
                <t-input v-model="displayNameModel" :placeholder="t('project.import.form.displayNamePlaceholder')" />
              </t-form-item>

              <t-form-item
                :label="t('project.import.form.canonicalProjectNameOverride')"
                name="canonical_project_name_override"
              >
                <t-input
                  v-model="canonicalProjectNameOverrideModel"
                  :placeholder="t('project.import.form.canonicalProjectNameOverridePlaceholder')"
                />
              </t-form-item>
            </div>

            <t-descriptions bordered :column="1" size="small">
              <t-descriptions-item :label="t('project.import.confirm.importSource')">
                {{ t('project.import.confirm.importSourceValue') }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.confirm.candidateKey')">
                <code class="project-import-confirm__technical">{{
                  candidate?.candidate_key || result.candidate_key
                }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.confirm.inspectionId')">
                <code class="project-import-confirm__technical">{{ result.inspection_id }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.confirm.canonicalNameSource')">
                {{ formatCanonicalNameSource(result.canonical_project_name_source) }}
              </t-descriptions-item>
              <t-descriptions-item v-if="candidateRuntimeLabel" :label="t('project.import.confirm.runtimeLabel')">
                {{ candidateRuntimeLabel }}
              </t-descriptions-item>
            </t-descriptions>
          </t-form>
        </div>
      </t-card>

      <t-card :bordered="true" :title="t('project.import.confirm.summaryTitle')">
        <t-descriptions bordered :column="1" size="small">
          <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
            <code class="project-import-confirm__technical">{{ result.canonical_project_name }}</code>
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.directory.workingDirectory')">
            <code class="project-import-confirm__technical">{{ resolvedWorkingDirectory || '-' }}</code>
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.validationStatus')">
            {{ formatValidationStatus(result.validation_status) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.composeFiles')">
            {{ result.compose_files.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.envFiles')">
            {{ result.env_files.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.serviceCount')">
            {{ result.services.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.runtimeMembersTitle')">
            {{ result.runtime_members.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.networks')">
            {{ networkRows.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.volumes')">
            {{ volumeRows.length }}
          </t-descriptions-item>
        </t-descriptions>
      </t-card>

      <t-card :bordered="true" :title="t('project.import.confirm.inspectionTitle')">
        <div class="project-import-confirm__card-content">
          <t-alert
            :theme="canImport ? 'success' : 'error'"
            :message="
              canImport ? t('project.import.confirm.inspectionReady') : t('project.import.confirm.inspectionBlocked')
            "
          />

          <div class="project-import-confirm__meta-row">
            <t-tag theme="primary" variant="light-outline">
              {{ t('project.import.confirm.conflictCount', { count: result.conflicts.length }) }}
            </t-tag>
            <t-tag theme="warning" variant="light-outline">
              {{ t('project.import.confirm.warningCount', { count: result.warnings.length }) }}
            </t-tag>
          </div>

          <ul class="project-import-confirm__checklist">
            <li>
              {{
                t('project.import.confirm.checks.validation', {
                  status: formatValidationStatus(result.validation_status),
                })
              }}
            </li>
            <li>{{ t('project.import.confirm.checks.composeFiles', { count: result.compose_files.length }) }}</li>
            <li>{{ t('project.import.confirm.checks.services', { count: result.services.length }) }}</li>
            <li>{{ t('project.import.confirm.checks.networks', { count: networkRows.length }) }}</li>
            <li>{{ t('project.import.confirm.checks.volumes', { count: volumeRows.length }) }}</li>
          </ul>

          <div v-if="result.conflicts.length || result.warnings.length" class="project-import-confirm__alerts">
            <t-alert
              v-for="(conflict, index) in result.conflicts"
              :key="`confirm-conflict-${index}`"
              theme="error"
              :message="conflict"
            />
            <t-alert
              v-for="(warning, index) in result.warnings"
              :key="`confirm-warning-${index}`"
              theme="warning"
              :message="warning"
            />
          </div>
        </div>
      </t-card>

      <t-card :bordered="true" :title="t('project.import.confirm.previewTitle')">
        <div class="project-import-confirm__preview-grid">
          <div class="project-import-confirm__preview-block">
            <strong>{{ t('project.import.preview.services') }}</strong>
            <p>{{ formatPreview(result.services) }}</p>
          </div>
          <div class="project-import-confirm__preview-block">
            <strong>{{ t('project.import.preview.networks') }}</strong>
            <p>{{ formatPreview(networkRows.map((item) => item.name)) }}</p>
          </div>
          <div class="project-import-confirm__preview-block">
            <strong>{{ t('project.import.preview.volumes') }}</strong>
            <p>{{ formatPreview(volumeRows.map((item) => item.name)) }}</p>
          </div>
          <div class="project-import-confirm__preview-block">
            <strong>{{ t('project.import.preview.composeFiles') }}</strong>
            <p>{{ formatPreview(result.compose_files.map((item) => item.display_path)) }}</p>
          </div>
        </div>
      </t-card>

      <t-card
        class="project-import-confirm__span-full"
        :bordered="true"
        :title="t('project.import.confirm.effectsTitle')"
      >
        <div class="project-import-confirm__card-content">
          <p class="project-import-confirm__supporting-copy">{{ t('project.import.confirm.effectsDescription') }}</p>
          <ul class="project-import-confirm__checklist">
            <li>{{ t('project.import.confirm.effects.registerProject') }}</li>
            <li>{{ t('project.import.confirm.effects.useInspection') }}</li>
            <li>{{ t('project.import.confirm.effects.applyOverrides') }}</li>
            <li>{{ t('project.import.confirm.effects.recordDiscoveredResources') }}</li>
          </ul>

          <div class="project-import-confirm__actions">
            <t-button theme="default" variant="outline" type="button" @click="$emit('back')">
              {{ t('project.import.actions.backToInspect') }}
            </t-button>
            <t-button :form="formId" theme="primary" type="submit" :disabled="!canImport" :loading="importLoading">
              {{ t('project.import.actions.import') }}
            </t-button>
            <t-button theme="default" variant="text" type="button" @click="$emit('reset')">
              {{ t('project.import.actions.reset') }}
            </t-button>
          </div>
        </div>
      </t-card>
    </div>
  </section>
</template>
<script setup lang="ts">
import type { FormProps, SubmitContext } from 'tdesign-vue-next';
import { computed } from 'vue';

import {
  normalizeImportInspectNetworkRows,
  normalizeImportInspectVolumeRows,
} from '../shared/import-inspect-resources';
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectImportInspectResponse, ProjectImportRuntimeCandidate } from '../types/import';
import ProjectImportSectionHeading from './ProjectImportSectionHeading.vue';

defineOptions({
  name: 'ProjectImportConfirmReview',
});

const props = defineProps<{
  canImport: boolean;
  candidate: ProjectImportRuntimeCandidate | null;
  canonicalProjectNameOverride: string;
  displayName: string;
  formData: FormProps['data'];
  formRules: FormProps['rules'];
  importError: string;
  importLoading: boolean;
  resolvedWorkingDirectory: string;
  result: ProjectImportInspectResponse;
}>();

const emit = defineEmits<{
  (event: 'back'): void;
  (event: 'reset'): void;
  (event: 'submit', context?: SubmitContext): void;
  (event: 'update:canonicalProjectNameOverride', value: string): void;
  (event: 'update:displayName', value: string): void;
}>();

const { t } = useProjectPageContext();
const formId = 'project-import-confirm-form';

const displayNameModel = computed({
  get: () => props.displayName,
  set: (value: string) => emit('update:displayName', value),
});

const canonicalProjectNameOverrideModel = computed({
  get: () => props.canonicalProjectNameOverride,
  set: (value: string) => emit('update:canonicalProjectNameOverride', value),
});

const candidateRuntimeLabel = computed(() => {
  if (!props.candidate) {
    return '';
  }

  return props.candidate.runtime_version?.trim()
    ? `${props.candidate.runtime_type} ${props.candidate.runtime_version.trim()}`
    : props.candidate.runtime_type;
});

const networkRows = computed(() => normalizeImportInspectNetworkRows(props.result));
const volumeRows = computed(() => normalizeImportInspectVolumeRows(props.result));

function formatCanonicalNameSource(source: string) {
  const key = `project.import.preview.canonicalNameSourceValues.${source}`;
  const translated = t(key);
  return translated === key ? source : translated;
}

function formatValidationStatus(status: string) {
  const key = `project.import.preview.validationStatusValues.${status}`;
  const translated = t(key);
  return translated === key ? status : translated;
}

function formatPreview(items: string[]) {
  if (!items.length) {
    return t('project.import.preview.none');
  }

  if (items.length <= 3) {
    return items.join(', ');
  }

  return t('project.import.confirm.previewWithMore', {
    preview: items.slice(0, 3).join(', '),
    count: items.length - 3,
  });
}

function handleSubmit(context: SubmitContext) {
  emit('submit', context);
}
</script>
<style scoped lang="less">
.project-import-confirm,
.project-import-confirm__card-content,
.project-import-confirm__alerts,
.project-import-confirm__preview-grid,
.project-import-confirm__preview-block {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-confirm__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-confirm__span-full {
  grid-column: 1 / -1;
}

.project-import-confirm__form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-confirm__meta-row,
.project-import-confirm__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.project-import-confirm__preview-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-confirm__preview-block {
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.project-import-confirm__preview-block strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.project-import-confirm__preview-block p,
.project-import-confirm__supporting-copy {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.project-import-confirm__checklist {
  color: var(--td-text-color-primary);
  display: grid;
  gap: var(--graft-density-gap-8);
  margin: 0;
  padding-left: var(--graft-density-gap-20);
}

.project-import-confirm__technical {
  color: var(--td-text-color-primary);
  display: inline-block;
  font-family: var(--td-font-family-mono, monospace);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

@media (width <= 1080px) {
  .project-import-confirm__grid,
  .project-import-confirm__form-grid,
  .project-import-confirm__preview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
