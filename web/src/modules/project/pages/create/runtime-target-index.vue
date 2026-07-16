<template>
  <div class="project-runtime-target-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.createRuntimeTarget.title"
        description-key="project.runtimeTarget.description"
        :source="{ labelKey: 'project.runtimeTarget.eyebrow', fallback: t('project.runtimeTarget.eyebrow') }"
      >
        <template #actions>
          <t-button variant="text" data-testid="project-runtime-target-back" @click="goToDeploymentModels">
            <template #icon><chevron-left-icon /></template>
            {{ t('project.runtimeTarget.backToDeploymentModels') }}
          </t-button>
        </template>
      </management-page-header>
      <t-alert v-if="loadError" theme="warning" :message="loadError" class="project-runtime-target-page__notice" />
      <div class="project-runtime-target-page__grid" :aria-busy="loading">
        <template v-for="target in targets" :key="target.runtime_target_id">
          <t-card
            v-if="target.readiness === 'ready'"
            bordered
            hover-shadow
            class="project-runtime-target-card project-runtime-target-card--actionable"
            :data-testid="`project-runtime-target-${target.runtime_target_id}`"
            role="button"
            tabindex="0"
            @click="selectTarget(target.runtime_target_id)"
            @keydown.enter.prevent="selectTarget(target.runtime_target_id)"
            @keydown.space.prevent="selectTarget(target.runtime_target_id)"
          >
            <img
              v-if="providerIcon(target.provider)"
              class="project-runtime-target-card__icon"
              :src="providerIcon(target.provider)"
              alt=""
            />
            <h2>{{ target.provider }}</h2>
            <p>{{ target.display_name }}</p>
          </t-card>
          <t-tooltip v-else :content="t('project.runtimeTarget.unavailableTooltip')" placement="top">
            <div class="project-runtime-target-card__disabled-wrap" tabindex="0" :aria-label="target.display_name">
              <t-card
                bordered
                class="project-runtime-target-card project-runtime-target-card--disabled"
                :data-testid="`project-runtime-target-${target.runtime_target_id}`"
                aria-disabled="true"
              >
                <img
                  v-if="providerIcon(target.provider)"
                  class="project-runtime-target-card__icon"
                  :src="providerIcon(target.provider)"
                  alt=""
                />
                <h2>{{ target.provider }}</h2>
                <p>{{ target.display_name }}</p>
              </t-card>
            </div>
          </t-tooltip>
        </template>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { ChevronLeftIcon } from 'tdesign-icons-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getApplicationComposeRuntimeTargets } from '../../api/project';
import dockerIcon from '../../assets/runtime/docker.svg?url';
import podmanIcon from '../../assets/runtime/podman.svg?url';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { useApplicationCreateRouteNavigation } from '../../shared/navigation';
import type { ApplicationComposeRuntimeTarget } from '../../types/project';

defineOptions({ name: 'ApplicationCreateRuntimeTargetIndex' });
// Compose 目标页依赖服务端能力目录；缺少合法部署上下文时必须回到创建入口。
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const navigateApplicationCreateRoute = useApplicationCreateRouteNavigation(router);
const loading = ref(false);
const loadError = ref('');
const targets = ref<ApplicationComposeRuntimeTarget[]>([]);
const deployment = computed(() => (route.query.deployment === 'compose' ? 'compose' : null));
onMounted(() => {
  if (!deployment.value) {
    void router.replace({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE.pageRouteName });
    return;
  }
  void loadTargets();
});
async function loadTargets() {
  loading.value = true;
  loadError.value = '';
  try {
    targets.value = (await getApplicationComposeRuntimeTargets()).items;
  } catch (error) {
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.runtimeTarget.loadFailed'));
  } finally {
    loading.value = false;
  }
}
function providerIcon(provider: string) {
  return provider === 'docker' ? dockerIcon : provider === 'podman' ? podmanIcon : '';
}
function goToDeploymentModels() {
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE.pageRouteName,
    query: { deployment: 'compose' },
  });
}
function selectTarget(runtimeTargetId: number) {
  navigateApplicationCreateRoute(
    {
      name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName,
      query: { deployment: 'compose', runtime_target_id: String(runtimeTargetId) },
    },
    'project.route.createSource.title',
  );
}
</script>
<style scoped>
.project-runtime-target-page__notice {
  margin-bottom: var(--graft-density-gap-16);
}

.project-runtime-target-page__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.project-runtime-target-card {
  min-height: 224px;
}

.project-runtime-target-card :deep(.t-card__body) {
  align-content: center;
  display: grid;
  gap: var(--graft-density-gap-12);
  height: 100%;
  justify-items: center;
  text-align: center;
}

.project-runtime-target-card__icon {
  height: 84px;
  object-fit: contain;
  width: 84px;
}

.project-runtime-target-card h2,
.project-runtime-target-card p {
  margin: 0;
}

.project-runtime-target-card p {
  color: var(--td-text-color-secondary);
}

.project-runtime-target-card--actionable {
  cursor: pointer;
}

.project-runtime-target-card--actionable:hover,
.project-runtime-target-card--actionable:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-2);
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
  transform: translateY(-2px);
}

.project-runtime-target-card__disabled-wrap:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-runtime-target-card--disabled {
  cursor: not-allowed;
  opacity: 0.62;
}
</style>
