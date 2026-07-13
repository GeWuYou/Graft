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
          <t-card
            v-if="runtime.supported"
            bordered
            hover-shadow
            class="project-runtime-card project-runtime-card--actionable"
            :data-testid="`project-runtime-${runtime.key}`"
            role="button"
            tabindex="0"
            @click="selectRuntime(runtime.key)"
            @keydown.enter.prevent="selectRuntime(runtime.key)"
            @keydown.space.prevent="selectRuntime(runtime.key)"
          >
            <img class="project-runtime-card__icon" :src="runtime.iconSrc" alt="" />
            <h2>{{ t(runtime.titleKey) }}</h2>
            <p>{{ t(runtime.descriptionKey) }}</p>
          </t-card>
          <t-tooltip v-else :content="t('project.runtime.unsupportedTooltip')" placement="top">
            <t-card
              bordered
              class="project-runtime-card project-runtime-card--disabled"
              :data-testid="`project-runtime-${runtime.key}`"
              aria-disabled="true"
              tabindex="-1"
            >
              <img class="project-runtime-card__icon" :src="runtime.iconSrc" alt="" />
              <h2>{{ t(runtime.titleKey) }}</h2>
              <p>{{ t(runtime.descriptionKey) }}</p>
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

import dockerIcon from '../../assets/runtime/docker.svg?url';
import kubernetesIcon from '../../assets/runtime/kubernetes.svg?url';
import nomadIcon from '../../assets/runtime/nomad.svg?url';
import podmanIcon from '../../assets/runtime/podman.svg?url';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';

defineOptions({ name: 'ProjectRuntimeIndex' });

const router = useRouter();
const { t } = useI18n();
const runtimes = [
  {
    key: 'docker-compose',
    supported: true,
    iconSrc: dockerIcon,
    titleKey: 'project.runtime.items.dockerCompose.title',
    descriptionKey: 'project.runtime.items.dockerCompose.description',
  },
  {
    key: 'docker-swarm',
    supported: false,
    iconSrc: dockerIcon,
    titleKey: 'project.runtime.items.dockerSwarm.title',
    descriptionKey: 'project.runtime.items.dockerSwarm.description',
  },
  {
    key: 'kubernetes',
    supported: false,
    iconSrc: kubernetesIcon,
    titleKey: 'project.runtime.items.kubernetes.title',
    descriptionKey: 'project.runtime.items.kubernetes.description',
  },
  {
    key: 'podman-compose',
    supported: false,
    iconSrc: podmanIcon,
    titleKey: 'project.runtime.items.podmanCompose.title',
    descriptionKey: 'project.runtime.items.podmanCompose.description',
  },
  {
    key: 'nomad',
    supported: false,
    iconSrc: nomadIcon,
    titleKey: 'project.runtime.items.nomad.title',
    descriptionKey: 'project.runtime.items.nomad.description',
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.project-runtime-card {
  aspect-ratio: 7 / 5;
  display: grid;
  grid-template-rows: minmax(0, 1fr);
  transition:
    box-shadow var(--td-anim-duration-base) var(--td-anim-time-fn-easing),
    transform var(--td-anim-duration-base) var(--td-anim-time-fn-easing);
}

.project-runtime-card :deep(.t-card__body) {
  align-content: center;
  box-sizing: border-box;
  display: grid;
  gap: var(--graft-density-gap-12);
  height: 100%;
  justify-items: center;
  min-height: 0;
  text-align: center;
}

.project-runtime-card__icon {
  display: block;
  height: 136px;
  object-fit: contain;
  width: 136px;
}

.project-runtime-card h2,
.project-runtime-card p {
  margin: 0;
}

.project-runtime-card p {
  color: var(--td-text-color-secondary);
}

.project-runtime-card--actionable {
  cursor: pointer;
}

.project-runtime-card--actionable:hover,
.project-runtime-card--actionable:focus-visible {
  transform: scale(1.025);
}

.project-runtime-card--actionable:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-runtime-card--disabled {
  cursor: not-allowed;
  opacity: 0.62;
}

@media (width <= 900px) {
  .project-runtime-page__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 640px) {
  .project-runtime-page__grid {
    grid-template-columns: 1fr;
  }
}
</style>
