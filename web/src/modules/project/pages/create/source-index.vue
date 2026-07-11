<template>
  <div class="project-creation-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.creation.title"
        description-key="project.creation.description"
        :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
      />

      <t-alert v-if="loadError" theme="warning" :message="loadError" class="project-creation-page__notice">
        <template #operation>
          <t-button theme="warning" variant="text" @click="loadCreationMethods">
            {{ t('project.creation.actions.retry') }}
          </t-button>
        </template>
      </t-alert>

      <div class="project-creation-page__grid" :aria-busy="loading">
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
              :data-testid="`project-creation-method-${method.method}`"
              theme="primary"
              :disabled="method.availability !== 'ready'"
              @click="openMethod(method.method)"
            >
              {{
                t(
                  method.availability === 'ready'
                    ? 'project.creation.actions.start'
                    : 'project.creation.actions.unavailable',
                )
              }}
            </t-button>
          </t-space>
        </t-card>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { getProjectCreationMethods } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { appendResolvedTab } from '../../shared/navigation';
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
const tabsRouterStore = useTabsRouterStore();
const { t } = useI18n();
const entries = ref<ProjectCreationMethod[]>([]);
const loadError = ref('');
const loading = ref(false);

const creationMethods = computed(() =>
  entries.value.map((entry) => ({
    ...entry,
    ...creationMethodDefinitions[entry.method],
  })),
);

onMounted(() => {
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

function openMethod(method: ProjectCreationMethodType) {
  const target = { name: routeNames[method] };
  const resolved = router.resolve(target);
  appendResolvedTab(tabsRouterStore, resolved, localizeRouteTitleKey(routeTitleKeys[method]));
  void router.push(target);
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
  min-height: 100%;
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

@media (width <= 640px) {
  .project-creation-page__grid {
    grid-template-columns: 1fr;
  }
}
</style>
