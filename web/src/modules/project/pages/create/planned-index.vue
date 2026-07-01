<template>
  <div class="project-planned-page">
    <management-page-content>
      <management-page-header
        :title-key="titleKey"
        description-key="project.createPlanned.description"
        :source="{ labelKey: 'project.create.eyebrow', fallback: t('project.create.eyebrow') }"
      />

      <t-card :bordered="true">
        <t-space direction="vertical" size="large">
          <t-alert theme="info" :message="t('project.createPlanned.notice')" />
          <t-descriptions size="small" :column="1" bordered>
            <t-descriptions-item :label="t('project.createPlanned.routePath')">
              <code>{{ route.path }}</code>
            </t-descriptions-item>
            <t-descriptions-item :label="t('project.createPlanned.boundary')">
              {{ t('project.createPlanned.boundaryValue') }}
            </t-descriptions-item>
          </t-descriptions>
        </t-space>
      </t-card>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';

defineOptions({
  name: 'ProjectPlannedSourceIndex',
});

const route = useRoute();
const { t } = useI18n();

const titleKey = computed(() =>
  String(route.name) === 'ProjectGitCreateIndex'
    ? 'project.route.createGit.title'
    : 'project.route.createTemplate.title',
);
</script>
