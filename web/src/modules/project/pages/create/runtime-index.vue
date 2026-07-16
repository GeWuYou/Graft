<template>
  <div class="project-deployment-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.create.title"
        description-key="project.deployment.description"
        :source="{ labelKey: 'project.deployment.eyebrow', fallback: t('project.deployment.eyebrow') }"
      >
        <template #actions>
          <t-button variant="text" data-testid="project-deployment-back" @click="goToApplicationList">
            <template #icon><chevron-left-icon /></template>
            {{ t('project.deployment.backToApplicationManagement') }}
          </t-button>
        </template>
      </management-page-header>

      <div class="project-deployment-page__grid">
        <template v-for="item in deploymentTypes" :key="item.key">
          <t-card
            bordered
            :hover-shadow="item.available"
            class="project-deployment-card"
            :class="{
              'project-deployment-card--actionable': item.available,
              'project-deployment-card--disabled': !item.available,
            }"
            :data-testid="`project-deployment-${item.key}`"
            :role="item.available ? 'button' : undefined"
            :tabindex="item.available ? 0 : undefined"
            :aria-disabled="item.available ? undefined : 'true'"
            @click="item.available && selectDeployment(item.key)"
            @keydown.enter.prevent="item.available && selectDeployment(item.key)"
            @keydown.space.prevent="item.available && selectDeployment(item.key)"
          >
            <div class="project-deployment-card__header">
              <img v-if="item.iconSrc" class="project-deployment-card__runtime-icon" :src="item.iconSrc" alt="" />
              <div
                v-else
                class="project-deployment-card__glyph"
                :class="`project-deployment-card__glyph--${item.glyph}`"
                aria-hidden="true"
              >
                <span v-for="node in item.glyphNodes" :key="node" class="project-deployment-card__glyph-node" />
              </div>
              <t-tag :theme="item.available ? 'primary' : 'default'" variant="light-outline" size="small">
                {{ t(item.statusKey) }}
              </t-tag>
            </div>

            <div class="project-deployment-card__content">
              <div>
                <h2>{{ t(item.titleKey) }}</h2>
                <p class="project-deployment-card__description">{{ t(item.descriptionKey) }}</p>
              </div>
              <ul class="project-deployment-card__capabilities">
                <li v-for="capabilityKey in item.capabilityKeys" :key="capabilityKey">{{ t(capabilityKey) }}</li>
              </ul>
            </div>

            <div class="project-deployment-card__footer">
              <t-button
                v-if="item.available"
                theme="primary"
                :data-testid="`project-deployment-${item.key}-select`"
                @click.stop="selectDeployment(item.key)"
              >
                {{ t('project.deployment.selectCompose') }}
              </t-button>
              <p v-else class="project-deployment-card__coming-soon">{{ t('project.deployment.comingSoonHelper') }}</p>
            </div>
          </t-card>
        </template>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { ChevronLeftIcon } from 'tdesign-icons-vue-next';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';

import dockerIcon from '../../assets/runtime/docker.svg?url';
import kubernetesIcon from '../../assets/runtime/kubernetes.svg?url';
import nomadIcon from '../../assets/runtime/nomad.svg?url';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { useApplicationCreateRouteNavigation } from '../../shared/navigation';

defineOptions({ name: 'ApplicationDeploymentTypeIndex' });
// 创建流程的运行时选择页只负责展示能力目录并保留路由上下文，具体目标由下一页加载。

const { t } = useI18n();
const router = useRouter();
const navigateApplicationCreateRoute = useApplicationCreateRouteNavigation(router);
const deploymentTypes = [
  {
    key: 'compose',
    available: true,
    iconSrc: '',
    glyph: 'compose',
    glyphNodes: 4,
    titleKey: 'project.deployment.items.compose.title',
    descriptionKey: 'project.deployment.items.compose.description',
    statusKey: 'project.deployment.recommended',
    capabilityKeys: [
      'project.deployment.items.compose.capabilities.0',
      'project.deployment.items.compose.capabilities.1',
      'project.deployment.items.compose.capabilities.2',
    ],
  },
  {
    key: 'swarm',
    available: false,
    glyph: 'swarm',
    glyphNodes: 6,
    iconSrc: dockerIcon,
    titleKey: 'project.deployment.items.swarm.title',
    descriptionKey: 'project.deployment.items.swarm.description',
    statusKey: 'project.deployment.comingSoon',
    capabilityKeys: [
      'project.deployment.items.swarm.capabilities.0',
      'project.deployment.items.swarm.capabilities.1',
      'project.deployment.items.swarm.capabilities.2',
    ],
  },
  {
    key: 'kubernetes',
    available: false,
    glyph: 'kubernetes',
    glyphNodes: 5,
    iconSrc: kubernetesIcon,
    titleKey: 'project.deployment.items.kubernetes.title',
    descriptionKey: 'project.deployment.items.kubernetes.description',
    statusKey: 'project.deployment.comingSoon',
    capabilityKeys: [
      'project.deployment.items.kubernetes.capabilities.0',
      'project.deployment.items.kubernetes.capabilities.1',
      'project.deployment.items.kubernetes.capabilities.2',
    ],
  },
  {
    key: 'nomad',
    available: false,
    glyph: 'nomad',
    glyphNodes: 4,
    iconSrc: nomadIcon,
    titleKey: 'project.deployment.items.nomad.title',
    descriptionKey: 'project.deployment.items.nomad.description',
    statusKey: 'project.deployment.comingSoon',
    capabilityKeys: ['project.deployment.items.nomad.capabilities.0', 'project.deployment.items.nomad.capabilities.1'],
  },
] as const;

type DeploymentType = (typeof deploymentTypes)[number]['key'];

function goToApplicationList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName });
}

function selectDeployment(deployment: DeploymentType) {
  navigateApplicationCreateRoute(
    { name: PROJECT_BOOTSTRAP_ROUTE.CREATE_RUNTIME_TARGET.pageRouteName, query: { deployment } },
    'project.route.createRuntimeTarget.title',
  );
}
</script>
<style scoped>
.project-deployment-page__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.project-deployment-card {
  display: flex;
  min-height: 340px;
}

.project-deployment-card :deep(.t-card__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.project-deployment-card__header {
  align-items: flex-start;
  display: flex;
  justify-content: space-between;
}

.project-deployment-card__glyph {
  display: grid;
  gap: var(--graft-density-gap-6);
  grid-template-columns: repeat(2, 10px);
  min-height: 32px;
  padding: var(--graft-density-gap-4);
  position: relative;
}

.project-deployment-card__glyph::before,
.project-deployment-card__glyph::after {
  background: var(--td-brand-color);
  content: '';
  opacity: 0.42;
  position: absolute;
}

.project-deployment-card__glyph::before {
  height: 2px;
  left: 10px;
  top: 14px;
  width: 28px;
}

.project-deployment-card__glyph::after {
  height: 28px;
  left: 23px;
  top: 1px;
  width: 2px;
}

.project-deployment-card__glyph-node {
  background: var(--td-brand-color-light);
  border: 2px solid var(--td-brand-color);
  border-radius: 2px;
  height: 10px;
  width: 10px;
  z-index: 1;
}

.project-deployment-card__runtime-icon {
  height: 32px;
  object-fit: contain;
  width: 48px;
}

.project-deployment-card__glyph--swarm {
  grid-template-columns: repeat(3, 8px);
}

.project-deployment-card__glyph--swarm::before {
  left: 8px;
  top: 13px;
  width: 39px;
}

.project-deployment-card__glyph--swarm::after {
  height: 21px;
  left: 26px;
  top: 5px;
}

.project-deployment-card__glyph--kubernetes {
  grid-template-columns: repeat(3, 8px);
}

.project-deployment-card__glyph--kubernetes::before {
  left: 8px;
  top: 13px;
  width: 39px;
}

.project-deployment-card__glyph--kubernetes::after {
  height: 21px;
  left: 26px;
  top: 5px;
}

.project-deployment-card__glyph--kubernetes .project-deployment-card__glyph-node:first-child {
  grid-column: 2;
}

.project-deployment-card__glyph--nomad {
  grid-template-columns: repeat(2, 10px);
  transform: rotate(45deg);
}

.project-deployment-card__glyph--nomad::before,
.project-deployment-card__glyph--nomad::after {
  display: none;
}

.project-deployment-card__content {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-deployment-card h2,
.project-deployment-card p {
  margin: 0;
}

.project-deployment-card h2 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-title-medium);
  line-height: var(--td-line-height-title-medium);
}

.project-deployment-card__description {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4) !important;
}

.project-deployment-card__capabilities {
  color: var(--td-text-color-secondary);
  display: grid;
  gap: var(--graft-density-gap-8);
  list-style: none;
  margin: 0;
  padding: 0;
}

.project-deployment-card__capabilities li {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.project-deployment-card__capabilities li::before {
  background: var(--td-brand-color);
  border-radius: 50%;
  content: '';
  height: 5px;
  width: 5px;
}

.project-deployment-card__footer {
  margin-top: auto;
}

.project-deployment-card__coming-soon {
  color: var(--td-text-color-placeholder);
  font-size: var(--td-font-size-body-small);
  line-height: var(--td-line-height-body-small);
}

.project-deployment-card--actionable {
  cursor: pointer;
}

.project-deployment-card--actionable:hover,
.project-deployment-card--actionable:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-2);
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
  transform: translateY(-2px);
}

.project-deployment-card--disabled {
  background: var(--td-bg-color-container);
  cursor: not-allowed;
}

.project-deployment-card--disabled :deep(*) {
  cursor: not-allowed;
}

.project-deployment-card--disabled .project-deployment-card__glyph-node {
  background: var(--td-bg-color-secondarycontainer);
  border-color: var(--td-component-border);
}

.project-deployment-card--disabled .project-deployment-card__glyph::before,
.project-deployment-card--disabled .project-deployment-card__glyph::after,
.project-deployment-card--disabled .project-deployment-card__capabilities li::before {
  background: var(--td-component-border);
}

@media (width <= 1199px) {
  .project-deployment-page__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 767px) {
  .project-deployment-page__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .project-deployment-card {
    min-height: 0;
  }
}
</style>
