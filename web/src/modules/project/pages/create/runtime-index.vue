<template>
  <div class="project-deployment-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.create.title"
        description-key="project.deployment.description"
        :source="{ labelKey: 'project.deployment.eyebrow', fallback: t('project.deployment.eyebrow') }"
      />
      <div class="project-deployment-page__grid">
        <template v-for="item in deploymentTypes" :key="item.key">
          <t-card
            v-if="item.available"
            bordered
            hover-shadow
            class="project-deployment-card project-deployment-card--actionable"
            :data-testid="`project-deployment-${item.key}`"
            role="button"
            tabindex="0"
            @click="selectDeployment(item.key)"
            @keydown.enter.prevent="selectDeployment(item.key)"
            @keydown.space.prevent="selectDeployment(item.key)"
          >
            <h2>{{ t(item.titleKey) }}</h2>
            <p>{{ t(item.descriptionKey) }}</p>
          </t-card>
          <t-tooltip v-else :content="t('project.workflow.unsupportedTooltip')" placement="top">
            <div class="project-deployment-card__disabled-wrap" tabindex="0" :aria-label="t(item.titleKey)">
              <t-card
                bordered
                class="project-deployment-card project-deployment-card--disabled"
                :data-testid="`project-deployment-${item.key}`"
                aria-disabled="true"
              >
                <h2>{{ t(item.titleKey) }}</h2>
                <p>{{ t(item.descriptionKey) }}</p>
              </t-card>
            </div>
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

defineOptions({ name: 'ProjectDeploymentTypeIndex' });

const { t } = useI18n();
const router = useRouter();
const deploymentTypes = [
  {
    key: 'compose',
    available: true,
    titleKey: 'project.deployment.items.compose.title',
    descriptionKey: 'project.deployment.items.compose.description',
  },
  {
    key: 'swarm',
    available: false,
    titleKey: 'project.deployment.items.swarm.title',
    descriptionKey: 'project.deployment.items.swarm.description',
  },
  {
    key: 'kubernetes',
    available: false,
    titleKey: 'project.deployment.items.kubernetes.title',
    descriptionKey: 'project.deployment.items.kubernetes.description',
  },
  {
    key: 'nomad',
    available: false,
    titleKey: 'project.deployment.items.nomad.title',
    descriptionKey: 'project.deployment.items.nomad.description',
  },
] as const;

function selectDeployment(deployment: 'compose') {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_RUNTIME_TARGET.pageRouteName, query: { deployment } });
}
</script>
<style scoped>
.project-deployment-page__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.project-deployment-card {
  min-height: 196px;
}

.project-deployment-card :deep(.t-card__body) {
  align-content: center;
  display: grid;
  gap: var(--graft-density-gap-12);
  height: 100%;
  text-align: center;
}

.project-deployment-card h2,
.project-deployment-card p {
  margin: 0;
}

.project-deployment-card p {
  color: var(--td-text-color-secondary);
}

.project-deployment-card--actionable {
  cursor: pointer;
}

.project-deployment-card--actionable:hover,
.project-deployment-card--actionable:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
  transform: translateY(-2px);
}

.project-deployment-card__disabled-wrap {
  outline: none;
}

.project-deployment-card__disabled-wrap:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-deployment-card--disabled {
  cursor: not-allowed;
  opacity: 0.62;
}
</style>
