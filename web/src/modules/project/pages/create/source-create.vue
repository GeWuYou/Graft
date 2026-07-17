<template>
  <div class="template-catalog" data-page-type="catalog-discovery">
    <management-page-content>
      <management-page-header
        title-key="project.route.createTemplate.title"
        :description="t('project.templateCatalog.description')"
        :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
      >
        <template #actions>
          <t-space size="small">
            <t-button variant="outline" @click="goToSource">{{ t('project.create.actions.backToSource') }}</t-button>
            <t-button :loading="loading" @click="loadCatalog">{{ t('project.create.actions.refresh') }}</t-button>
          </t-space>
        </template>
      </management-page-header>

      <t-card bordered class="template-catalog__filters">
        <div class="template-catalog__filter-grid">
          <t-input
            v-model="keyword"
            clearable
            :placeholder="t('project.templateCatalog.searchPlaceholder')"
            @enter="applyFilters"
          >
            <template #prefix-icon><search-icon /></template>
          </t-input>
          <t-select v-model="sort" :options="sortOptions" @change="applyFilters" />
        </div>
        <div class="template-catalog__categories" role="group" :aria-label="t('project.templateCatalog.category')">
          <t-button
            v-for="option in categoryOptions"
            :key="option.value"
            size="small"
            :theme="category === option.value ? 'primary' : 'default'"
            :variant="category === option.value ? 'base' : 'outline'"
            @click="selectCategory(option.value as ApplicationTemplateCategory | '')"
            >{{ option.label }}</t-button
          >
        </div>
      </t-card>

      <t-alert v-if="loadError" theme="error" :message="loadError" class="template-catalog__feedback">
        <template #operation
          ><t-button theme="danger" variant="text" @click="loadCatalog">{{
            t('project.templates.retry')
          }}</t-button></template
        >
      </t-alert>
      <t-loading :loading="loading" class="template-catalog__loading">
        <div v-if="items.length" class="template-catalog__grid">
          <t-card v-for="item in items" :key="item.template_id" bordered class="template-catalog__card">
            <template #header>
              <div class="template-catalog__card-header">
                <div>
                  <button class="template-catalog__title" type="button" @click="openDetail(item)">
                    {{ item.display_name }}
                  </button>
                  <p>{{ item.description || t('project.sourceCreate.noDescription') }}</p>
                </div>
                <t-tag variant="light-outline">{{ categoryLabel(item.category) }}</t-tag>
              </div>
            </template>
            <div class="template-catalog__metadata">
              <span>{{ item.deployment_adapter_kind }}</span>
              <span>{{ t('project.templateCatalog.updatedAt', { value: formatTime(item.updated_at) }) }}</span>
            </div>
            <template #footer>
              <t-space size="small">
                <t-button variant="text" @click="openDetail(item)">{{
                  t('project.templateCatalog.viewDetail')
                }}</t-button>
                <t-button theme="primary" :loading="selecting === item.template_id" @click="selectTemplate(item)">{{
                  t('project.sourceCreate.useTemplate')
                }}</t-button>
              </t-space>
            </template>
          </t-card>
        </div>
        <t-empty v-else-if="!loadError" :description="t('project.templateCatalog.empty')" />
      </t-loading>
      <div v-if="items.length" class="template-catalog__pagination">
        <t-pagination
          :current="page"
          :page-size="pageSize"
          :total="paginationTotal"
          :show-page-size="false"
          @current-change="changePage"
        />
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { SearchIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getApplicationTemplateCatalog } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { navigateToApplicationCreateSource } from '../../shared/navigation';
import type { ApplicationTemplateCatalogItem, ApplicationTemplateCategory } from '../../types/project';

// 模板目录只消费发布摘要并把筛选状态写回 URL；完整定义始终通过详情或版本快照接口读取。
defineOptions({ name: 'ApplicationTemplateCatalog' });

const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const items = ref<ApplicationTemplateCatalogItem[]>([]);
const loading = ref(false);
const selecting = ref('');
const loadError = ref('');
const keyword = ref(queryString('q'));
const category = ref<ApplicationTemplateCategory | ''>(validCategory(queryString('category')));
const sort = ref<'updated_desc' | 'name_asc'>(queryString('sort') === 'name_asc' ? 'name_asc' : 'updated_desc');
const page = ref(Math.max(1, Number(queryString('page')) || 1));
const pageSize = 24;
const hasMore = ref(false);

const categoryOptions = computed(() => [
  { value: '', label: t('project.templateCatalog.allCategories') },
  ...(['database', 'cache', 'mq', 'proxy', 'storage', 'monitoring', 'logging', 'cicd', 'ai', 'other'] as const).map(
    (value) => ({ value, label: categoryLabel(value) }),
  ),
]);
const sortOptions = computed(() => [
  { value: 'updated_desc', label: t('project.templateCatalog.sortUpdated') },
  { value: 'name_asc', label: t('project.templateCatalog.sortName') },
]);
const paginationTotal = computed(() =>
  hasMore.value ? page.value * pageSize + 1 : (page.value - 1) * pageSize + items.value.length,
);

onMounted(() => void loadCatalog());
watch(() => route.query, syncFromRoute);

async function loadCatalog() {
  loading.value = true;
  loadError.value = '';
  try {
    const response = await getApplicationTemplateCatalog({
      deployment_adapter_kind: 'compose',
      q: keyword.value.trim() || undefined,
      category: category.value || undefined,
      sort: sort.value,
      page: page.value,
      page_size: pageSize,
    });
    items.value = response.items;
    hasMore.value = response.has_more;
  } catch (error) {
    items.value = [];
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.sourceCreate.templatesLoadFailed'));
  } finally {
    loading.value = false;
  }
}

function applyFilters() {
  page.value = 1;
  void replaceQuery();
}
function selectCategory(value: ApplicationTemplateCategory | '') {
  category.value = value;
  applyFilters();
}
function changePage(value: number) {
  page.value = value;
  void replaceQuery();
}
async function replaceQuery() {
  await router.replace({
    query: {
      ...route.query,
      ...(keyword.value.trim() ? { q: keyword.value.trim() } : { q: undefined }),
      ...(category.value ? { category: category.value } : { category: undefined }),
      ...(sort.value === 'updated_desc' ? { sort: undefined } : { sort: sort.value }),
      ...(page.value > 1 ? { page: String(page.value) } : { page: undefined }),
    },
  });
}
function syncFromRoute() {
  keyword.value = queryString('q');
  category.value = validCategory(queryString('category'));
  sort.value = queryString('sort') === 'name_asc' ? 'name_asc' : 'updated_desc';
  page.value = Math.max(1, Number(queryString('page')) || 1);
  void loadCatalog();
}
function openDetail(item: ApplicationTemplateCatalogItem) {
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE_DETAIL.pageRouteName,
    params: { templateId: item.template_id },
    query: route.query,
  });
}
async function selectTemplate(item: ApplicationTemplateCatalogItem) {
  if (route.query.deployment !== 'compose' || !/^\d+$/.test(String(route.query.runtime_target_id || ''))) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    goToSource();
    return;
  }
  selecting.value = item.template_id;
  await router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK.pageRouteName,
    query: { ...route.query, template_id: item.template_id, template_version_id: item.version.template_version_id },
  });
  selecting.value = '';
}
function goToSource() {
  navigateToApplicationCreateSource(router, route.query);
}
function categoryLabel(value: ApplicationTemplateCategory) {
  return t(`project.templateCatalog.categories.${value}`);
}
function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(new Date(value));
}
function queryString(key: string) {
  return typeof route.query[key] === 'string' ? route.query[key] : '';
}
function validCategory(value: string): ApplicationTemplateCategory | '' {
  return ['database', 'cache', 'mq', 'proxy', 'storage', 'monitoring', 'logging', 'cicd', 'ai', 'other'].includes(value)
    ? (value as ApplicationTemplateCategory)
    : '';
}
</script>
<style scoped>
.template-catalog__filters,
.template-catalog__feedback,
.template-catalog__loading {
  margin-top: var(--graft-density-gap-16);
}

.template-catalog__filter-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(0, 1fr) 180px;
}

.template-catalog__categories {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-12);
}

.template-catalog__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
}

.template-catalog__card {
  display: flex;
  flex-direction: column;
  min-height: 210px;
}

.template-catalog__card-header {
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.template-catalog__title {
  background: transparent;
  border: 0;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: var(--td-font-title-medium);
  padding: 0;
  text-align: left;
}

.template-catalog__card-header p,
.template-catalog__metadata {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-6) 0 0;
}

.template-catalog__metadata {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.template-catalog__pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--graft-density-gap-16);
}

@media (width <= 720px) {
  .template-catalog__filter-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .template-catalog__pagination {
    justify-content: center;
  }
}
</style>
