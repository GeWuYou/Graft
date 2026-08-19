<template>
  <section class="build-workspace-create-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        class="build-workspace-create-page__surface"
        title-key="build.workspaces.create.title"
        description-key="build.workspaces.create.description"
        :source="{ labelKey: 'build.workspaces.eyebrow', fallback: t('build.workspaces.eyebrow') }"
      />

      <t-card bordered class="build-workspace-create-page__surface">
        <t-form :data="form" :rules="rules" layout="vertical" @submit="submit">
          <t-form-item name="name" :label="t('build.workspaces.create.name')">
            <t-input v-model="form.name" :placeholder="t('build.workspaces.create.namePlaceholder')" />
          </t-form-item>
          <t-form-item name="source_reference" :label="t('build.workspaces.create.application')">
            <t-select
              v-model="form.source_reference"
              :options="applicationOptions"
              :loading="applicationLoading"
              :disabled="applicationLoading || applicationOptions.length === 0"
              :placeholder="t('build.workspaces.create.applicationPlaceholder')"
              clearable
            />
            <template #help>
              <span>{{ t('build.workspaces.create.applicationHelp') }}</span>
            </template>
          </t-form-item>
          <t-alert v-if="applicationError" theme="warning" :message="applicationError">
            <template #operation>
              <t-button size="small" variant="outline" @click="openApplications">
                {{ t('build.workspaces.create.manageApplications') }}
              </t-button>
            </template>
          </t-alert>
          <t-alert v-if="message" :theme="messageTheme" :message="message" />
          <div class="build-workspace-create-page__actions">
            <t-button variant="outline" :disabled="submitting" @click="returnToList">
              {{ t('build.workspaces.create.back') }}
            </t-button>
            <t-button v-permission="BUILD_PERMISSION_CODE.CREATE" theme="primary" type="submit" :loading="submitting">
              <template #icon><save-icon /></template>
              {{ t('build.workspaces.create.submit') }}
            </t-button>
          </div>
        </t-form>
      </t-card>
    </management-page-content>
  </section>
</template>
<script setup lang="ts">
// 创建页只登记 Build 对既有 Application workspace 的引用，不复制或编辑应用源码。
import { SaveIcon } from 'tdesign-icons-vue-next';
import type { SubmitContext } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getApplications } from '@/modules/project/api/project';
import { APPLICATION_ROUTE_PATH } from '@/modules/project/contract/paths';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { createBuildWorkspace } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import { BUILD_PERMISSION_CODE } from '../../contract/permissions';
import type { BuildWorkspaceCreateRequest } from '../../types/build';

const { t } = useI18n();
const router = useRouter();
const form = ref<BuildWorkspaceCreateRequest>({ name: '', source_kind: 'application_workspace', source_reference: '' });
const applications = ref<Awaited<ReturnType<typeof getApplications>>['items']>([]);
const applicationLoading = ref(false);
const applicationError = ref('');
const submitting = ref(false);
const message = ref('');
const messageTheme = ref<'success' | 'error'>('success');

const applicationOptions = computed(() =>
  applications.value.map((application) => ({ label: application.display_name, value: application.application_id })),
);
const rules = computed(() => ({
  name: [{ required: true, message: t('build.workspaces.create.nameRequired') }],
  source_reference: [{ required: true, message: t('build.workspaces.create.applicationRequired') }],
}));

async function loadApplications() {
  applicationLoading.value = true;
  applicationError.value = '';
  try {
    const response = await getApplications({ limit: 100, offset: 0 });
    applications.value = response.items ?? [];
    if (applications.value.length === 0) applicationError.value = t('build.workspaces.create.applicationEmpty');
  } catch (error) {
    applications.value = [];
    applicationError.value = resolveLocalizedErrorMessage(t, error, t('build.workspaces.create.applicationLoadFailed'));
  } finally {
    applicationLoading.value = false;
  }
}

function openApplications() {
  void router.push(APPLICATION_ROUTE_PATH.LIST);
}

function returnToList() {
  void router.push(BUILD_ROUTE_PATH.WORKSPACES);
}

async function submit({ validateResult }: SubmitContext) {
  if (validateResult !== true) return;
  submitting.value = true;
  message.value = '';
  try {
    await createBuildWorkspace(form.value);
    await router.push(BUILD_ROUTE_PATH.WORKSPACES);
  } catch (error) {
    messageTheme.value = 'error';
    message.value = resolveLocalizedErrorMessage(t, error, t('build.workspaces.create.submitFailed'));
  } finally {
    submitting.value = false;
  }
}

onMounted(() => void loadApplications());
</script>
<style scoped lang="less">
.build-workspace-create-page {
  min-width: 0;
  width: 100%;
}

.build-workspace-create-page__surface {
  margin-inline: auto;
  max-width: 760px;
  width: 100%;
}

.build-workspace-create-page__actions {
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
  margin-top: var(--graft-density-gap-20);
  padding: var(--graft-density-gap-20) 0 0;
}

@media (width <= 768px) {
  .build-workspace-create-page__actions {
    align-items: stretch;
    flex-direction: column-reverse;
  }

  .build-workspace-create-page__actions :deep(.t-button) {
    width: 100%;
  }
}
</style>
