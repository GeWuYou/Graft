<template>
  <div class="project-runtime-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.create.title"
        description-key="project.runtime.description"
        :source="{ labelKey: 'project.runtime.eyebrow', fallback: t('project.runtime.eyebrow') }"
      />

      <div class="project-runtime-page__grid">
        <template v-for="runtime in runtimes" :key="runtime.key">
          <t-card v-if="runtime.supported" bordered class="project-runtime-card">
            <h2>{{ t(runtime.titleKey) }}</h2>
            <p>{{ t(runtime.descriptionKey) }}</p>
            <t-tag theme="success" variant="light-outline">{{ t(runtime.statusKey) }}</t-tag>
            <t-button data-testid="project-runtime-docker-compose" theme="primary" @click="selectRuntime(runtime.key)">
              {{ t('project.runtime.actions.select') }}
            </t-button>
          </t-card>
          <t-tooltip v-else :content="t('project.runtime.unsupportedTooltip')" placement="top">
            <t-card class="project-runtime-card project-runtime-card--disabled" bordered aria-disabled="true">
              <h2>{{ t(runtime.titleKey) }}</h2>
              <p>{{ t(runtime.descriptionKey) }}</p>
              <t-tag theme="default" variant="light-outline">{{ t(runtime.statusKey) }}</t-tag>
              <span class="project-runtime-card__disabled-action">{{ t('project.runtime.unsupported') }}</span>
            </t-card>
          </t-tooltip>
        </template>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';

import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';

defineOptions({ name: 'ProjectRuntimeIndex' });

const router = useRouter();
const { t } = useI18n();
const runtimes = [
  {
    key: 'docker-compose',
    supported: true,
    titleKey: 'project.runtime.items.dockerCompose.title',
    descriptionKey: 'project.runtime.items.dockerCompose.description',
    statusKey: 'project.runtime.status.supported',
  },
  {
    key: 'docker-swarm',
    supported: false,
    titleKey: 'project.runtime.items.dockerSwarm.title',
    descriptionKey: 'project.runtime.items.dockerSwarm.description',
    statusKey: 'project.runtime.status.comingSoon',
  },
  {
    key: 'kubernetes',
    supported: false,
    titleKey: 'project.runtime.items.kubernetes.title',
    descriptionKey: 'project.runtime.items.kubernetes.description',
    statusKey: 'project.runtime.status.comingSoon',
  },
  {
    key: 'podman-compose',
    supported: false,
    titleKey: 'project.runtime.items.podmanCompose.title',
    descriptionKey: 'project.runtime.items.podmanCompose.description',
    statusKey: 'project.runtime.status.comingSoon',
  },
  {
    key: 'nomad',
    supported: false,
    titleKey: 'project.runtime.items.nomad.title',
    descriptionKey: 'project.runtime.items.nomad.description',
    statusKey: 'project.runtime.status.comingSoon',
  },
] as const;

function selectRuntime(runtime: 'docker-compose') {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName, query: { runtime } });
}
</script>
<style scoped>
.project-runtime-page__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.project-runtime-card {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-height: 220px;
}

.project-runtime-card h2,
.project-runtime-card p {
  margin: 0;
}

.project-runtime-card p {
  color: var(--td-text-color-secondary);
}

.project-runtime-card--disabled {
  opacity: 0.62;
}

.project-runtime-card__disabled-action {
  color: var(--td-text-color-placeholder);
}

@media (width <= 640px) {
  .project-runtime-page__grid {
    grid-template-columns: 1fr;
  }
}
</style>
