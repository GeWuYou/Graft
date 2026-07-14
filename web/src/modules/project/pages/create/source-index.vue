<template>
  <div class="project-creation-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.creation.title"
        description-key="project.creation.description"
        :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
      >
        <template #actions>
          <t-button variant="text" data-testid="project-creation-back" @click="goToRuntimeTargets">
            <template #icon><project-back-icon /></template>
            {{ t('project.creation.actions.backToRuntimeTargets') }}
          </t-button>
        </template>
      </management-page-header>

      <t-alert v-if="loadError" theme="warning" :message="loadError" class="project-creation-page__notice">
        <template #operation>
          <t-button theme="warning" variant="text" @click="loadCreationMethods">
            {{ t('project.creation.actions.retry') }}
          </t-button>
        </template>
      </t-alert>

      <div v-if="hasComposeTarget" class="project-creation-page__grid" :aria-busy="loading">
        <t-card v-for="method in creationMethods" :key="method.method" :bordered="true" class="project-creation-card">
          <template #header>
            <div class="project-creation-card__header">
              <div>
                <h2>{{ t(method.titleKey) }}</h2>
                <p>{{ t(method.descriptionKey) }}</p>
              </div>
              <t-tag :theme="method.availability === 'ready' ? 'success' : 'warning'" variant="light-outline">
                {{ availabilityLabel(method.availability) }}
              </t-tag>
            </div>
          </template>

          <t-space direction="vertical" size="large" class="project-creation-card__body">
            <ul class="project-creation-card__benefits">
              <li v-for="benefitKey in method.benefitKeys" :key="benefitKey">{{ t(benefitKey) }}</li>
            </ul>

            <t-alert
              v-if="method.availability === 'blocked'"
              theme="warning"
              :message="blockedReasonLabel(method.blocked_reason)"
            />

            <t-collapse borderless>
              <t-collapse-panel :header="t('project.creation.advanced.title')" :value="method.method">
                <p class="project-creation-card__advanced">{{ t(method.advancedKey) }}</p>
              </t-collapse-panel>
            </t-collapse>

            <t-button
              v-if="method.availability === 'ready'"
              :data-testid="`project-creation-method-${method.method}`"
              theme="primary"
              @click="openMethod(method.method)"
              >{{ t('project.creation.actions.start') }}</t-button
            >
            <t-tooltip v-else :content="t('project.workflow.unsupportedTooltip')" placement="top"
              ><span class="project-creation-card__disabled-wrap" tabindex="0"
                ><t-button :data-testid="`project-creation-method-${method.method}`" theme="primary" disabled>{{
                  t('project.creation.actions.unavailable')
                }}</t-button></span
              ></t-tooltip
            >
          </t-space>
        </t-card>
        <t-tooltip :content="t('project.workflow.unsupportedTooltip')" placement="top">
          <div class="project-creation-card project-creation-card--disabled" tabindex="0" aria-disabled="true">
            <t-card :bordered="true" class="project-creation-card__disabled-card">
              <template #header
                ><div class="project-creation-card__header">
                  <div>
                    <h2>{{ t('project.creation.methods.git.title') }}</h2>
                    <p>{{ t('project.creation.methods.git.description') }}</p>
                  </div>
                  <t-tag theme="default" variant="light-outline">
                    {{ t('project.creation.availability.comingSoon') }}
                  </t-tag>
                </div></template
              >
              <div class="project-creation-card__disabled-body">
                <p>{{ t('project.creation.methods.git.unavailableHint') }}</p>
                <t-button disabled>{{ t('project.creation.actions.unavailable') }}</t-button>
              </div>
            </t-card>
          </div>
        </t-tooltip>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { ChevronLeftIcon as ProjectBackIcon } from 'tdesign-icons-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getProjectCreationMethods } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { useProjectCreateRouteNavigation } from '../../shared/navigation';
import type { ProjectCreationMethod, ProjectCreationMethodType } from '../../types/project';

defineOptions({
  name: 'ProjectCreateMethodIndex',
});

type CreationMethodDefinition = {
  titleKey: string;
  descriptionKey: string;
  benefitKeys: string[];
  advancedKey: string;
};

const creationMethodDefinitions: Record<ProjectCreationMethodType, CreationMethodDefinition> = {
  blank: {
    titleKey: 'project.creation.methods.blank.title',
    descriptionKey: 'project.creation.methods.blank.description',
    benefitKeys: [
      'project.creation.methods.blank.benefits.workspace',
      'project.creation.methods.blank.benefits.lifecycle',
      'project.creation.methods.blank.benefits.authority',
    ],
    advancedKey: 'project.creation.methods.blank.advanced',
  },
  template: {
    titleKey: 'project.creation.methods.template.title',
    descriptionKey: 'project.creation.methods.template.description',
    benefitKeys: [
      'project.creation.methods.template.benefits.workspace',
      'project.creation.methods.template.benefits.lifecycle',
      'project.creation.methods.template.benefits.authority',
    ],
    advancedKey: 'project.creation.methods.template.advanced',
  },
  import: {
    titleKey: 'project.creation.methods.import.title',
    descriptionKey: 'project.creation.methods.import.description',
    benefitKeys: [
      'project.creation.methods.import.benefits.detect',
      'project.creation.methods.import.benefits.preserve',
      'project.creation.methods.import.benefits.register',
    ],
    advancedKey: 'project.creation.methods.import.advanced',
  },
};

const routeNames: Record<ProjectCreationMethodType, string> = {
  blank: PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK.pageRouteName,
  template: PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName,
  import: PROJECT_BOOTSTRAP_ROUTE.CREATE_IMPORT.pageRouteName,
};

const routeTitleKeys: Record<ProjectCreationMethodType, string> = {
  blank: 'project.route.createBlank.title',
  template: 'project.route.createTemplate.title',
  import: 'project.route.createImport.title',
};

const router = useRouter();
const route = useRoute();
const navigateProjectCreateRoute = useProjectCreateRouteNavigation(router);
const { t } = useI18n();
const entries = ref<ProjectCreationMethod[]>([]);
const loadError = ref('');
const loading = ref(false);
const hasComposeTarget = computed(
  () => route.query.deployment === 'compose' && /^\d+$/.test(String(route.query.runtime_target_id || '')),
);

const creationMethods = computed(() =>
  entries.value.map((entry) => ({
    ...entry,
    ...creationMethodDefinitions[entry.method],
  })),
);

onMounted(() => {
  if (!hasComposeTarget.value) {
    void router.replace({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE.pageRouteName });
    return;
  }
  void loadCreationMethods();
});

async function loadCreationMethods() {
  loading.value = true;
  loadError.value = '';
  try {
    const response = await getProjectCreationMethods();
    entries.value = response.items;
  } catch (error) {
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.creation.messages.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function availabilityLabel(availability: ProjectCreationMethod['availability']) {
  return t(`project.creation.availability.${availability}`);
}

function blockedReasonLabel(reason?: string | null) {
  if (reason && ['managed_root_unconfigured', 'managed_root_invalid', 'managed_root_unknown'].includes(reason)) {
    return t(`project.creation.blockedReasons.${reason}`);
  }
  return t('project.creation.blockedReasons.unknown');
}

function goToRuntimeTargets() {
  router.back();
}

function openMethod(method: ProjectCreationMethodType) {
  const target = {
    name: routeNames[method],
    query: { deployment: 'compose', runtime_target_id: String(route.query.runtime_target_id) },
  };
  navigateProjectCreateRoute(target, routeTitleKeys[method]);
}
</script>
<style scoped>
.project-creation-page {
  min-height: 100%;
}

.project-creation-page__notice,
.project-creation-page__grid,
.project-creation-card__body {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-creation-page__grid {
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  margin-top: var(--graft-density-gap-16);
}

.project-creation-card {
  min-height: 348px;
}

.project-creation-card--disabled {
  cursor: not-allowed;
  display: flex;
  flex-direction: column;
}

.project-creation-card__disabled-card {
  cursor: not-allowed;
  display: flex;
  flex: 1;
  flex-direction: column;
}

.project-creation-card__disabled-card :deep(.t-card__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
}

.project-creation-card__disabled-body {
  color: var(--td-text-color-placeholder);
  display: flex;
  flex: 1;
  flex-direction: column;
  font-size: var(--td-font-size-body-small);
  gap: var(--graft-density-gap-16);
}

.project-creation-card__disabled-body p {
  margin: 0;
}

.project-creation-card__disabled-body :deep(.t-button) {
  align-self: flex-start;
  margin-top: auto;
}

.project-creation-card__disabled-wrap {
  display: block;
  outline: none;
}

.project-creation-card__disabled-wrap:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-creation-card--disabled:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-creation-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-creation-card__header h2,
.project-creation-card__header p,
.project-creation-card__advanced {
  margin: 0;
}

.project-creation-card__header h2 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-body-large);
  font-weight: var(--td-font-weight-medium);
}

.project-creation-card__header p,
.project-creation-card__advanced {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-8);
}

.project-creation-card__benefits {
  display: grid;
  gap: var(--graft-density-gap-8);
  margin: 0;
  padding-left: calc(20px * var(--graft-theme-density-scale));
}

.project-creation-card__body :deep(.t-button) {
  justify-self: start;
}

.project-creation-card:not(.project-creation-card--disabled) {
  cursor: pointer;
  transition:
    border-color var(--td-transition-duration-base) var(--td-transition-timing-function-ease-in-out),
    box-shadow var(--td-transition-duration-base) var(--td-transition-timing-function-ease-in-out),
    transform var(--td-transition-duration-base) var(--td-transition-timing-function-ease-in-out);
}

.project-creation-card:not(.project-creation-card--disabled):hover,
.project-creation-card:not(.project-creation-card--disabled):focus-within {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-2);
  transform: translateY(-2px);
}

@media (width <= 640px) {
  .project-creation-page__grid {
    grid-template-columns: 1fr;
  }
}
</style>
