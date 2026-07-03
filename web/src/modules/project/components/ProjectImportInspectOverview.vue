<template>
  <section class="project-import-overview">
    <project-import-section-heading
      :description="t('project.import.preview.overviewDescription')"
      :title="t('project.import.preview.overviewTitle')"
    />

    <div class="project-import-overview__grid">
      <t-card :bordered="true" :title="t('project.import.preview.authorityTitle')">
        <t-descriptions size="small" :column="1" bordered>
          <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
            <code class="project-import-overview__technical">{{ result.canonical_project_name }}</code>
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.composeFiles')">
            {{ result.compose_files.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.envFiles')">
            {{ result.env_files.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.directory.workingDirectory')">
            <t-tooltip :content="resolvedWorkingDirectory || '-'" placement="top-left">
              <code class="project-import-overview__technical">{{ resolvedWorkingDirectory || '-' }}</code>
            </t-tooltip>
          </t-descriptions-item>
        </t-descriptions>
      </t-card>

      <t-card :bordered="true" :title="t('project.import.preview.summaryTitle')">
        <t-descriptions size="small" :column="1" bordered>
          <t-descriptions-item :label="t('project.import.preview.validationStatus')">
            {{ formatValidationStatus(result.validation_status) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.serviceCount')">
            {{ result.services.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.configHash')">
            <t-tooltip :content="result.config_hash || '-'" placement="top-left">
              <code class="project-import-overview__technical">{{ result.config_hash || '-' }}</code>
            </t-tooltip>
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.canonicalNameSource')">
            {{ formatCanonicalNameSource(result.canonical_project_name_source) }}
          </t-descriptions-item>
        </t-descriptions>
      </t-card>

      <t-card
        class="project-import-overview__span-full"
        :bordered="true"
        :title="t('project.import.preview.discoveryTitle')"
      >
        <t-descriptions size="small" :column="1" bordered>
          <t-descriptions-item :label="t('project.import.preview.services')">
            {{ result.services.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.runtimeMembersTitle')">
            {{ result.runtime_members.length }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.networks')">
            {{ networkCount }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.volumes')">
            {{ volumeCount }}
          </t-descriptions-item>
        </t-descriptions>
      </t-card>

      <t-card
        class="project-import-overview__span-full"
        :bordered="true"
        :title="t('project.import.preview.diagnosticsTitle')"
      >
        <div class="project-import-overview__diagnostics">
          <t-alert v-if="!canImport" theme="error" :message="t('project.import.preview.blockedDescription')" />
          <t-alert
            v-for="(conflict, index) in result.conflicts"
            :key="`overview-conflict-${index}-${conflict}`"
            theme="error"
            :message="conflict"
          />
          <t-alert
            v-for="(warning, index) in result.warnings"
            :key="`overview-warning-${index}-${warning}`"
            theme="warning"
            :message="warning"
          />
          <t-empty
            v-if="!(result.warnings.length || result.conflicts.length)"
            :description="t('project.import.preview.noDiagnostics')"
            type="empty"
          />
        </div>
      </t-card>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  normalizeImportInspectNetworkRows,
  normalizeImportInspectVolumeRows,
} from '../shared/import-inspect-resources';
import { formatImportPreviewCanonicalNameSource, formatImportPreviewValidationStatus } from '../shared/import-preview';
import type { ProjectImportInspectResponse } from '../types/import';
import ProjectImportSectionHeading from './ProjectImportSectionHeading.vue';

defineOptions({
  name: 'ProjectImportInspectOverview',
});

const props = defineProps<{
  canImport: boolean;
  resolvedWorkingDirectory: string;
  result: ProjectImportInspectResponse;
}>();

const { t } = useI18n();

const networkCount = computed(() => normalizeImportInspectNetworkRows(props.result).length);
const volumeCount = computed(() => normalizeImportInspectVolumeRows(props.result).length);

function formatValidationStatus(status: string) {
  return formatImportPreviewValidationStatus(t, status);
}

function formatCanonicalNameSource(source: string) {
  return formatImportPreviewCanonicalNameSource(t, source);
}
</script>
<style scoped lang="less">
.project-import-overview,
.project-import-overview__grid,
.project-import-overview__diagnostics {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-overview__grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-overview__span-full {
  grid-column: 1 / -1;
}

.project-import-overview__technical {
  color: var(--td-text-color-primary);
  display: inline-block;
  font-family: var(--td-font-family-mono, monospace);
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

@media (width <= 1080px) {
  .project-import-overview__grid {
    grid-template-columns: 1fr;
  }
}
</style>
