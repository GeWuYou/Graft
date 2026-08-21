<template>
  <div class="docker-network-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      :title="t('container.networks.title')"
      :description="t('container.networks.description')"
      :source="{ labelKey: 'container.networks.eyebrow', fallback: t('container.networks.eyebrow') }"
    >
      <template #actions>
        <t-space v-if="!isCompactDensity">
          <t-button v-if="canRemove" variant="outline" @click="openCleanup">
            {{ t('container.networks.cleanup.action') }}
          </t-button>
          <t-button v-if="canCreate" theme="primary" @click="openCreateDrawer">
            {{ t('container.networks.create') }}
          </t-button>
        </t-space>
        <t-space v-else>
          <t-dropdown
            v-if="canRemove"
            :options="headerActionOptions"
            placement="bottom-right"
            trigger="click"
            @click="handleHeaderAction"
          >
            <t-tooltip :content="t('container.networks.mobile.moreActions')">
              <t-button shape="square" variant="outline" :aria-label="t('container.networks.mobile.moreActions')">
                <template #icon><ellipsis-icon /></template>
              </t-button>
            </t-tooltip>
          </t-dropdown>
          <t-button v-if="canCreate" theme="primary" @click="openCreateDrawer">
            {{ t('container.networks.create') }}
          </t-button>
        </t-space>
      </template>
    </management-page-header>

    <management-statistics-bar :items="metrics" aria-live="polite" layout="summary" />

    <resource-query-panel
      v-model="resourceQueryState"
      :config="queryConfig"
      :loading="networkQuery.isFetching.value"
      @reset="resetFilters"
      @search="applyQueryState"
    >
      <template #toolbar-actions><saved-query-view-control :controller="savedViews" /></template>
    </resource-query-panel>

    <t-alert
      v-if="networkQuery.isError.value"
      class="docker-network-page__alert"
      theme="error"
      :message="resolveLocalizedErrorMessage(t, networkQuery.error.value, t('container.networks.loadFailed'))"
    />

    <management-paged-table
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :rows="networks"
      :total="networkTotal"
      :columns="allColumns"
      :loading="networkQuery.isFetching.value"
      :selected-row-keys="isCompactDensity ? [] : selectedNetworkIds"
      :cards-visible="true"
      :column-sets="{ comfortable: ['name', 'context', 'status', 'operation'] }"
      density-scope="viewport"
      :footer-summary="paginationSummary"
      :empty-title="t('container.networks.emptyTitle')"
      :empty-description="t('container.networks.emptyDescription')"
      :cell-slot-names="['name', 'context', 'containers', 'status', 'operation']"
      @select-change="handleSelectChange"
      @page-change="handlePageChange"
    >
      <template #toolbar>
        <table-view-toolbar
          :refresh-label="t('container.networks.refresh')"
          :refresh-loading="networkQuery.isFetching.value"
          @refresh="networkQuery.refetch"
        />
      </template>
      <template #batch>
        <management-batch-bar
          v-if="!isCompactDensity && selectedNetworkIds.length"
          :selected-label="t('container.networks.batch.selected', { count: selectedNetworkIds.length })"
          :clear-label="t('container.networks.batch.cancelSelection')"
          @clear="clearSelection"
        >
          <t-button v-if="canRemove" size="small" theme="danger" variant="outline" @click="openBatchRemoveDialog">{{
            t('container.networks.batch.remove')
          }}</t-button>
        </management-batch-bar>
      </template>
      <template #name="{ row }">
        <div class="docker-network-page__identity">
          <t-button class="docker-network-page__name" variant="text" @click="openDetail(row.id)">{{
            row.name
          }}</t-button>
          <span v-if="networkHint(row)" class="docker-network-page__hint">{{ networkHint(row) }}</span>
        </div>
      </template>
      <template #context="{ row }">
        <t-tooltip :content="sourceDescription(row)" placement="top-left">
          <span class="docker-network-page__source">{{ sourceDescription(row) }}</span>
        </t-tooltip>
      </template>
      <template #containers="{ row }">
        <div v-if="row.container_references?.length" class="docker-network-page__container-list">
          <container-reference-list
            :references="row.container_references"
            :title="t('container.networks.connectedContainers')"
            @open="openContainerReference"
          />
        </div>
        <span v-else class="docker-network-page__muted">{{ relationEmptyLabel(row.relationship_status) }}</span>
      </template>
      <template #status="{ row }">
        <t-tag :theme="relationshipPresentation(row.relationship_status).theme" size="small" variant="light">
          {{ relationshipPresentation(row.relationship_status).label }}
        </t-tag>
      </template>
      <template #operation="{ row }">
        <t-space align="center" size="small">
          <t-button size="small" variant="outline" @click="openDetail(row.id)">{{
            t('container.networks.detail')
          }}</t-button>
          <t-button v-if="canRemove" size="small" theme="danger" variant="text" @click="openRemoveDialog(row)">{{
            t('container.networks.remove')
          }}</t-button>
        </t-space>
      </template>
      <template #cards>
        <t-loading :loading="networkQuery.isFetching.value">
          <responsive-card-list v-if="networks.length" class="docker-network-page__mobile-list">
            <article v-for="row in networks" :key="row.id" class="docker-network-page__mobile-card">
              <header class="docker-network-page__mobile-card-head">
                <strong class="docker-network-page__mobile-card-name">{{ row.name }}</strong>
                <t-tag :theme="relationshipPresentation(row.relationship_status).theme" size="small" variant="light">
                  {{ relationshipPresentation(row.relationship_status).label }}
                </t-tag>
              </header>
              <dl class="docker-network-page__mobile-card-details">
                <div>
                  <dt>{{ t('container.resourceContext.source') }}</dt>
                  <dd>{{ sourceDescription(row) }}</dd>
                </div>
                <div>
                  <dt>{{ t('container.resourceContext.containers') }}</dt>
                  <dd v-if="row.container_references?.length" class="docker-network-page__container-list">
                    <container-reference-list
                      :references="row.container_references"
                      :title="t('container.networks.connectedContainers')"
                      @open="openContainerReference"
                    />
                  </dd>
                  <dd v-else class="docker-network-page__muted">{{ relationEmptyLabel(row.relationship_status) }}</dd>
                </div>
              </dl>
              <docker-resource-card-actions
                :detail-label="t('container.networks.detail')"
                :more-actions="networkActionOptions"
                :more-label="t('container.list.actions.more')"
                @detail="openDetail(row.id)"
                @action="handleNetworkAction(row, $event)"
              />
            </article>
          </responsive-card-list>
          <t-empty
            v-else-if="!networkQuery.isFetching.value"
            :title="t('container.networks.emptyTitle')"
            :description="t('container.networks.emptyDescription')"
          />
        </t-loading>
      </template>
      <template #empty>
        <t-empty :title="t('container.networks.emptyTitle')" :description="t('container.networks.emptyDescription')" />
      </template>
    </management-paged-table>

    <t-dialog
      v-model:visible="cleanup.visible.value"
      :header="t('container.networks.cleanup.title')"
      width="760px"
      @confirm="submitCleanup"
    >
      <docker-cleanup-loading-host
        :empty="!cleanup.loading.value && !cleanup.items.value.length"
        :loading="cleanup.loading.value"
      >
        <t-alert v-if="cleanup.items.value.length" theme="warning" :message="t('container.networks.cleanup.warning')" />
        <section v-if="cleanup.items.value.length" class="docker-network-cleanup__preview">
          <div class="docker-network-cleanup__head">
            <h3>{{ t('container.networks.cleanup.candidateTitle', { count: cleanup.items.value.length }) }}</h3>
            <span>{{
              t('container.networks.cleanup.selectedCount', { count: cleanup.selectedIds.value.length })
            }}</span>
          </div>
          <t-table
            row-key="id"
            size="small"
            :columns="cleanupColumns"
            :data="cleanup.previewItems.value"
            :selected-row-keys="cleanup.selectedIds.value"
            @select-change="handleCleanupSelectChange"
          />
          <div v-if="cleanup.pageCount.value > 1" class="docker-network-cleanup__pager">
            <t-button
              shape="circle"
              variant="text"
              :disabled="cleanup.previewPage.value === 1"
              @click="cleanup.previousPage"
              ><arrow-up-icon
            /></t-button>
            <span>{{ cleanup.previewPage.value }} / {{ cleanup.pageCount.value }}</span>
            <t-button
              shape="circle"
              variant="text"
              :disabled="cleanup.previewPage.value === cleanup.pageCount.value"
              @click="cleanup.nextPage"
              ><arrow-down-icon
            /></t-button>
          </div>
        </section>
        <t-empty
          v-if="!cleanup.loading.value && !cleanup.items.value.length"
          :title="t('container.networks.cleanup.empty')"
        />
      </docker-cleanup-loading-host>
      <template #footer
        ><t-button
          theme="danger"
          :disabled="!cleanup.selectedIds.value.length"
          :loading="cleanup.removing.value"
          @click="submitCleanup"
          >{{ t('container.networks.cleanup.removeSelected', { count: cleanup.selectedIds.value.length }) }}</t-button
        ></template
      >
    </t-dialog>

    <t-drawer
      v-model:visible="createDrawerVisible"
      :header="t('container.networks.createTitle')"
      size="600px"
      destroy-on-close
    >
      <t-form :data="createForm" label-align="top">
        <t-form-item :label="t('container.networks.form.name')" required>
          <t-input v-model="createForm.name" :placeholder="t('container.networks.form.namePlaceholder')" />
        </t-form-item>
        <t-form-item :label="t('container.networks.form.driver')" required>
          <t-select v-model="createForm.driver">
            <t-option v-for="driver in drivers" :key="driver" :value="driver" :label="driverLabel(driver)" />
          </t-select>
        </t-form-item>
        <t-form-item :label="t('container.networks.form.options')">
          <t-space direction="vertical">
            <t-switch
              v-model="createForm.internal"
              :label="[t('container.networks.internal'), t('container.networks.internal')]"
            />
            <t-switch
              v-model="createForm.attachable"
              :label="[t('container.networks.attachable'), t('container.networks.attachable')]"
            />
          </t-space>
        </t-form-item>
        <t-form-item :label="t('container.networks.form.subnet')">
          <t-input v-model="createForm.subnet" :placeholder="t('container.networks.form.subnetPlaceholder')" />
        </t-form-item>
        <t-form-item :label="t('container.networks.form.gateway')">
          <t-input v-model="createForm.gateway" :placeholder="t('container.networks.form.gatewayPlaceholder')" />
        </t-form-item>
      </t-form>
      <template #footer>
        <t-space>
          <t-button variant="outline" @click="createDrawerVisible = false">{{
            t('container.networks.cancel')
          }}</t-button>
          <t-button theme="primary" :loading="creating" @click="submitCreate">{{
            t('container.networks.create')
          }}</t-button>
        </t-space>
      </template>
    </t-drawer>

    <resource-detail-layout
      v-model:visible="detailDrawerVisible"
      :title="detailQuery.data.value?.name || t('container.networks.detailTitle')"
      :back-label="t('container.detail.back')"
      size="medium"
    >
      <t-loading :loading="detailQuery.isFetching.value">
        <t-alert
          v-if="detailQuery.isError.value"
          theme="error"
          :message="resolveLocalizedErrorMessage(t, detailQuery.error.value, t('container.networks.detailLoadFailed'))"
        />
        <template v-else-if="detailQuery.data.value">
          <section class="docker-network-page__section">
            <h3>{{ t('container.resourceContext.overview') }}</h3>
            <t-space break-line size="small">
              <t-tag size="small" variant="light-outline">{{ driverLabel(detailQuery.data.value.driver) }}</t-tag>
              <t-tag
                :theme="relationshipPresentation(detailQuery.data.value.relationship_status).theme"
                size="small"
                variant="light"
                >{{ relationshipPresentation(detailQuery.data.value.relationship_status).label }}</t-tag
              >
              <t-tag v-if="detailQuery.data.value.internal" theme="warning" size="small" variant="light">{{
                t('container.networks.internal')
              }}</t-tag>
              <span>{{
                t('container.resourceContext.containerCount', { count: detailQuery.data.value.container_count })
              }}</span>
              <span>{{ formatLocaleDateTime(detailQuery.data.value.created_at, locale) }}</span>
            </t-space>
          </section>
          <docker-resource-context-card :context="detailQuery.data.value.context" resource-kind="network" />
          <section class="docker-network-page__section docker-network-page__section--relations">
            <h3>{{ t('container.resourceContext.relations') }}</h3>
            <div v-if="detailQuery.data.value.container_references?.length" class="docker-network-page__relation-cards">
              <t-button
                v-for="reference in detailQuery.data.value.container_references"
                :key="reference.id"
                class="docker-network-page__relation-card"
                variant="outline"
                @click="openContainerReference(reference.id)"
              >
                <strong>{{ reference.name || reference.id }}</strong>
                <span>{{ t('container.networks.connectedContainers') }}</span>
              </t-button>
            </div>
            <span v-else class="docker-network-page__muted">{{
              relationEmptyLabel(detailQuery.data.value.relationship_status)
            }}</span>
          </section>
          <section class="docker-network-page__section">
            <h3>{{ t('container.resourceContext.configuration') }}</h3>
            <dl class="docker-network-page__detail-fields">
              <div>
                <dt>{{ t('container.networks.fields.driver') }}</dt>
                <dd>{{ driverLabel(detailQuery.data.value.driver) }}</dd>
              </div>
              <div>
                <dt>{{ t('container.networks.fields.scope') }}</dt>
                <dd>{{ scopeLabel(detailQuery.data.value.scope) }}</dd>
              </div>
            </dl>
            <template v-if="detailQuery.data.value.ipam?.driver || detailQuery.data.value.ipam?.config?.length">
              <h4>{{ t('container.networks.ipam') }}</h4>
              <dl class="docker-network-page__detail-fields">
                <div>
                  <dt>{{ t('container.networks.fields.driver') }}</dt>
                  <dd>{{ detailQuery.data.value.ipam.driver || '-' }}</dd>
                </div>
                <div>
                  <dt>{{ t('container.networks.form.subnet') }}</dt>
                  <dd>{{ detailQuery.data.value.ipam.config?.[0]?.subnet || '-' }}</dd>
                </div>
                <div>
                  <dt>{{ t('container.networks.form.gateway') }}</dt>
                  <dd>{{ detailQuery.data.value.ipam.config?.[0]?.gateway || '-' }}</dd>
                </div>
              </dl>
            </template>
          </section>
          <t-collapse
            v-if="Object.keys(detailQuery.data.value.labels ?? {}).length"
            class="docker-network-page__section"
          >
            <t-collapse-panel :header="t('container.resourceContext.metadata')" value="metadata">
              <dl class="docker-network-page__metadata-list">
                <div v-for="(value, key) in detailQuery.data.value.labels" :key="key">
                  <dt>{{ key }}</dt>
                  <dd>{{ value }}</dd>
                </div>
              </dl>
              <p class="docker-network-page__metadata-id">
                {{ t('container.networks.fields.id') }}: {{ detailQuery.data.value.id }}
              </p>
            </t-collapse-panel>
          </t-collapse>
          <container-danger-zone
            v-if="canRemove"
            class="docker-network-page__danger-zone"
            :action-label="t('container.networks.remove')"
            :description="t('container.networks.removeRisk')"
            @action="openRemoveDialog(detailQuery.data.value)"
          />
        </template>
        <div v-else-if="!detailQuery.isFetching.value" class="docker-network-page__detail-state">
          <t-empty
            size="small"
            :title="t('container.networks.detailEmptyTitle')"
            :description="t('container.networks.detailEmptyDescription')"
          />
        </div>
      </t-loading>
    </resource-detail-layout>

    <t-dialog
      v-model:visible="removeDialogVisible"
      :header="t('container.networks.removeTitle')"
      :confirm-btn="t('container.networks.remove')"
      :confirm-loading="removing"
      theme="danger"
      @confirm="submitRemove"
    >
      <p>{{ t('container.networks.removeDescription', { name: selectedNetwork?.name ?? '' }) }}</p>
      <p class="docker-network-page__remove-warning">{{ t('container.networks.removeRisk') }}</p>
      <t-input v-model="removeConfirmation" :placeholder="selectedNetwork?.name" />
    </t-dialog>

    <t-dialog
      v-model:visible="batchRemoveDialogVisible"
      :header="t('container.networks.batch.removeTitle')"
      :confirm-btn="t('container.networks.batch.remove')"
      :confirm-loading="batchRemoving"
      theme="danger"
      @confirm="submitBatchRemove"
    >
      <p>
        {{
          t('container.networks.batch.removeDescription', {
            count: selectedNetworkIds.length,
            names: selectedNetworkNames,
          })
        }}
      </p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { ArrowDownIcon, ArrowUpIcon, EllipsisIcon } from 'tdesign-icons-vue-next';
import type { DropdownProps, TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import type { components } from '@/contracts/openapi/generated/schema';
import {
  ManagementBatchBar,
  ManagementPageHeader,
  ManagementStatisticsBar,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import {
  type ResourceQueryConfig,
  ResourceQueryPanel,
  type ResourceQueryState,
  SavedQueryViewControl,
} from '@/shared/components/query-list';
import ResourceDetailLayout from '@/shared/components/responsive/ResourceDetailLayout.vue';
import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import { useViewportResponsiveVariant } from '@/shared/composables';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability/time';
import { usePermissionStore } from '@/store';

import {
  createDockerNetwork,
  deleteDockerNetworkSavedView,
  type DockerNetworkListQuery,
  getDockerNetworks,
  getDockerNetworkSavedViews,
  postDockerNetworkSavedView,
  putDockerNetworkSavedView,
  removeDockerNetwork,
} from '../../api/container';
import ContainerDangerZone from '../../components/ContainerDangerZone.vue';
import DockerResourceCardActions from '../../components/DockerResourceCardActions.vue';
import DockerResourceContextCard from '../../components/DockerResourceContextCard.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import DockerCleanupLoadingHost from '../../shared/cleanup/DockerCleanupLoadingHost.vue';
import { useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';
import ContainerReferenceList from '../../shared/ContainerReferenceList.vue';
import {
  invalidateDockerNetworkQueries,
  useDockerNetworkDetailQuery,
  useDockerNetworkListQuery,
} from '../../shared/docker-network-queries';
import { useDockerResourceSavedViews } from '../../shared/docker-resource-saved-views';
import {
  getDockerResourceRelationEmptyLabel,
  getDockerResourceRelationshipPresentation,
  getDockerResourceSourceDescription,
  getDockerResourceSourceLabel,
} from '../../shared/resource-presentation';
import type { DockerNetwork, DockerNetworkCreateRequest, DockerNetworkDriver } from '../../types/docker-network';

defineOptions({ name: 'DockerNetworkListIndex' });

// 网络列表与详情分别读取，避免列表快照承载连接容器等高基数信息；窄屏仅切换同一列表数据的呈现方式。
const { locale, t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const viewportVariant = useViewportResponsiveVariant();
const pagination = reactive({ current: 1, pageSize: 20 });
type DockerResourceSource = components['schemas']['docker-resource-source'];

const draftFilters = reactive<{
  keyword: string;
  driver: string;
  scope: string;
  usage: string;
  source: DockerResourceSource | '';
  compose_project: string;
}>({ keyword: '', driver: '', scope: '', usage: '', source: '', compose_project: '' });
const appliedFilters = ref({ ...draftFilters });
const networkListQuery = computed<DockerNetworkListQuery>(() => ({
  limit: pagination.pageSize,
  offset: (pagination.current - 1) * pagination.pageSize,
  keyword: appliedFilters.value.keyword || undefined,
  driver: appliedFilters.value.driver || undefined,
  scope: appliedFilters.value.scope || undefined,
  usage: appliedFilters.value.usage ? (appliedFilters.value.usage as 'used' | 'unused') : undefined,
  source: appliedFilters.value.source || undefined,
  compose_project: appliedFilters.value.compose_project || undefined,
}));
const networkQuery = useDockerNetworkListQuery(networkListQuery);
const selectedNetworkId = ref('');
const detailQuery = useDockerNetworkDetailQuery(selectedNetworkId);
const createDrawerVisible = ref(false);
const detailDrawerVisible = ref(false);
const removeDialogVisible = ref(false);
const batchRemoveDialogVisible = ref(false);
const creating = ref(false);
const removing = ref(false);
const batchRemoving = ref(false);
const selectedNetworkIds = ref<string[]>([]);
const selectedNetworkNamesByID = ref<Record<string, string>>({});
const selectedNetworkNames = computed(() =>
  selectedNetworkIds.value
    .map((id) => selectedNetworkNamesByID.value[id])
    .filter(Boolean)
    .join(', '),
);
const selectedNetwork = ref<DockerNetwork | null>(null);
const removeConfirmation = ref('');
const drivers: DockerNetworkDriver[] = ['bridge', 'overlay', 'macvlan', 'ipvlan', 'none'];
const createForm = reactive({
  name: '',
  driver: 'bridge' as DockerNetworkDriver,
  internal: false,
  attachable: false,
  subnet: '',
  gateway: '',
});
const queryConfig = computed<ResourceQueryConfig>(() => ({
  resource: 'docker-network.list',
  search: true,
  filterBuilder: { enabled: true },
  placeholder: t('container.networks.filters.search'),
  savedView: true,
  filters: [
    {
      key: 'driver',
      label: t('container.networks.filters.driver'),
      type: 'select',
      options: drivers.map((value) => ({ value, label: driverLabel(value) })),
    },
    {
      key: 'scope',
      label: t('container.networks.filters.scope'),
      type: 'select',
      options: ['local', 'swarm', 'global'].map((value) => ({ value, label: scopeLabel(value) })),
    },
    {
      key: 'usage',
      label: t('container.networks.filters.usage'),
      type: 'select',
      options: [
        { value: 'used', label: t('container.networks.filters.inUse') },
        { value: 'unused', label: t('container.networks.filters.unused') },
      ],
    },
    {
      key: 'source',
      label: t('container.resourceContext.source'),
      type: 'select',
      options: ['compose', 'docker_default', 'docker', 'managed', 'imported', 'unknown'].map((value) => ({
        value,
        label: getDockerResourceSourceLabel(t, value as DockerResourceSource),
      })),
    },
    { key: 'compose_project', label: t('container.resourceContext.project'), type: 'input' },
  ],
}));
const resourceQueryState = computed<ResourceQueryState>({
  get: () => ({
    keyword: draftFilters.keyword,
    filters: { ...draftFilters },
    page: pagination.current,
    pageSize: pagination.pageSize,
  }),
  set: (value) => {
    Object.assign(draftFilters, {
      keyword: value.keyword,
      driver: value.filters.driver ?? '',
      scope: value.filters.scope ?? '',
      usage: value.filters.usage ?? '',
      source: value.filters.source ?? '',
      compose_project: value.filters.compose_project ?? '',
    });
    pagination.current = value.page;
    pagination.pageSize = value.pageSize;
  },
});
const savedViews = useDockerResourceSavedViews({
  api: {
    list: getDockerNetworkSavedViews,
    create: postDockerNetworkSavedView,
    update: putDockerNetworkSavedView,
    remove: deleteDockerNetworkSavedView,
  },
  applyState: (state) => {
    resourceQueryState.value = state.queryState;
    applyFilters();
  },
  getState: () => ({ pageSize: pagination.pageSize, queryState: resourceQueryState.value, visibleColumns: [] }),
  onError: (error: unknown) =>
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.networks.loadFailed'))),
});

const networks = computed(() => networkQuery.data.value?.items ?? []);
const networkTotal = computed(() => networkQuery.data.value?.total ?? 0);
const networkSummary = computed(() => networkQuery.data.value?.summary);
const metrics = computed(() => [
  { label: t('container.networks.summary.totalLabel'), value: networkSummary.value?.total ?? '--' },
  { label: t('container.networks.summary.inUseLabel'), value: networkSummary.value?.in_use ?? '--' },
  { label: t('container.networks.summary.unusedLabel'), value: networkSummary.value?.unused ?? '--' },
]);
const paginationSummary = computed(() =>
  networkTotal.value
    ? t('container.networks.pagination.summary', {
        start: (pagination.current - 1) * pagination.pageSize + 1,
        end: Math.min(pagination.current * pagination.pageSize, networkTotal.value),
        total: networkTotal.value,
      })
    : t('container.networks.pagination.empty'),
);
const canCreate = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.NETWORK_CREATE));
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.NETWORK_REMOVE));
const isCompactDensity = computed(() => viewportVariant.value.density === 'compact');
const headerActionOptions = computed(() => [{ content: t('container.networks.cleanup.action'), value: 'cleanup' }]);
const networkActionOptions = computed(() =>
  canRemove.value ? [{ danger: true, label: t('container.networks.remove'), value: 'remove' }] : [],
);
let cleanup: ReturnType<typeof useDockerCleanup<DockerNetwork>>;
cleanup = useDockerCleanup<DockerNetwork>({
  fetchCandidates: fetchCleanupCandidates,
  execute: removeCleanupCandidates,
});
const allColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.networks.fields.name'), minWidth: 280, ellipsis: true },
  { colKey: 'context', title: t('container.resourceContext.source'), minWidth: 190, ellipsis: true },
  { colKey: 'containers', title: t('container.resourceContext.containers'), minWidth: 240 },
  { colKey: 'status', title: t('container.networks.fields.status'), width: 104 },
  { colKey: 'operation', title: t('container.networks.operation'), width: 144, fixed: 'right' as const },
]);
const cleanupColumns = computed<TableProps['columns']>(() =>
  (allColumns.value ?? []).filter((column) => ['row-select', 'name', 'status'].includes(String(column.colKey))),
);
onMounted(() => void savedViews.load());
async function fetchCleanupCandidates() {
  const first = await getDockerNetworks({ limit: 100, offset: 0, usage: 'unused' });
  const all = [...first.items];
  for (let offset = first.items.length; offset < first.total; offset += 100) {
    const page = await getDockerNetworks({ limit: 100, offset, usage: 'unused' });
    all.push(...page.items);
  }
  return all.filter((network) => network.removable !== false);
}
async function removeCleanupCandidates(
  ids: string[],
): Promise<{ items: Array<{ id: string; success: boolean }>; unknownResponseIds: string[] }> {
  const selected: DockerNetwork[] = cleanup.items.value.filter((network: DockerNetwork) => ids.includes(network.id));
  const results = await Promise.allSettled(
    selected.map((network: DockerNetwork) => removeDockerNetwork(network.id, { confirm_network_name: network.name })),
  );
  return {
    items: results.map((result, index) => ({ id: selected[index].id, success: result.status === 'fulfilled' })),
    unknownResponseIds: [],
  };
}
async function openCleanup() {
  try {
    await cleanup.open();
  } catch {
    MessagePlugin.error(t('container.networks.cleanup.loadFailed'));
  }
}
async function submitCleanup() {
  await cleanup.submit();
  await invalidateDockerNetworkQueries();
  if (!cleanup.selectedIds.value.length) cleanup.visible.value = false;
}
const handleHeaderAction: NonNullable<DropdownProps['onClick']> = (payload) => {
  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (action === 'cleanup') void openCleanup();
};
function handleNetworkAction(network: DockerNetwork, payload: { value?: unknown } | string | number) {
  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (action === 'remove' && canRemove.value) openRemoveDialog(network);
}
function handleCleanupSelectChange(keys: Array<string | number>) {
  cleanup.select(keys);
}

function driverLabel(driver: string) {
  return t(`container.networks.drivers.${driver}`, driver);
}
function scopeLabel(scope: string) {
  return t(`container.networks.scopes.${scope}`, scope);
}
function sourceDescription(network: DockerNetwork) {
  return getDockerResourceSourceDescription(t, network.context);
}
function networkHint(network: DockerNetwork) {
  return [
    network.internal ? t('container.networks.internal') : '',
    network.ingress ? t('container.networks.ingress') : '',
  ]
    .filter(Boolean)
    .join(' · ');
}
const relationshipPresentation = (status: DockerNetwork['relationship_status']) =>
  getDockerResourceRelationshipPresentation(t, status);
const relationEmptyLabel = (status: DockerNetwork['relationship_status']) =>
  getDockerResourceRelationEmptyLabel(t, status);
function openContainerReference(containerId: string) {
  void router.push({
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: containerId },
    query: { tab: 'network' },
  });
}
function openCreateDrawer() {
  Object.assign(createForm, {
    name: '',
    driver: 'bridge',
    internal: false,
    attachable: false,
    subnet: '',
    gateway: '',
  });
  createDrawerVisible.value = true;
}
function openDetail(networkId: string) {
  selectedNetworkId.value = networkId;
  detailDrawerVisible.value = true;
}
function openRemoveDialog(network: DockerNetwork) {
  selectedNetwork.value = network;
  removeConfirmation.value = '';
  removeDialogVisible.value = true;
}
function clearSelection() {
  selectedNetworkIds.value = [];
  selectedNetworkNamesByID.value = {};
}
function openBatchRemoveDialog() {
  if (canRemove.value && selectedNetworkIds.value.length) batchRemoveDialogVisible.value = true;
}
function handlePageChange(pageInfo: { current: number; pageSize: number }) {
  pagination.current = pageInfo.current;
  pagination.pageSize = pageInfo.pageSize;
}
function handleSelectChange(keys: Array<string | number>) {
  selectedNetworkIds.value = keys.map(String);
  const selectedIDs = new Set(selectedNetworkIds.value);
  const names = { ...selectedNetworkNamesByID.value };
  for (const network of networks.value) {
    if (selectedIDs.has(network.id)) names[network.id] = network.name;
  }
  for (const id of Object.keys(names)) {
    if (!selectedIDs.has(id)) delete names[id];
  }
  selectedNetworkNamesByID.value = names;
}
function applyFilters() {
  appliedFilters.value = {
    keyword: draftFilters.keyword.trim(),
    driver: draftFilters.driver,
    scope: draftFilters.scope,
    usage: draftFilters.usage,
    source: draftFilters.source,
    compose_project: draftFilters.compose_project.trim(),
  };
  pagination.current = 1;
}
function applyQueryState(value: ResourceQueryState) {
  resourceQueryState.value = value;
  applyFilters();
}
function resetFilters() {
  Object.assign(draftFilters, { keyword: '', driver: '', scope: '', usage: '', source: '', compose_project: '' });
  applyFilters();
}
async function submitCreate() {
  if (!createForm.name.trim()) {
    MessagePlugin.warning(t('container.networks.form.nameRequired'));
    return;
  }
  const ipam =
    createForm.subnet.trim() || createForm.gateway.trim()
      ? { subnet: createForm.subnet.trim() || undefined, gateway: createForm.gateway.trim() || undefined }
      : undefined;
  const body: DockerNetworkCreateRequest = {
    name: createForm.name.trim(),
    driver: createForm.driver,
    internal: createForm.internal,
    attachable: createForm.attachable,
    ipam,
  };
  creating.value = true;
  try {
    const receipt = await createDockerNetwork(body);
    void receipt.task_id;
    createDrawerVisible.value = false;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.networks.createFailed')));
  } finally {
    creating.value = false;
  }
}
async function submitRemove() {
  if (!selectedNetwork.value || removeConfirmation.value !== selectedNetwork.value.name) {
    MessagePlugin.warning(t('container.networks.removeConfirmationMismatch'));
    return;
  }
  removing.value = true;
  try {
    const receipt = await removeDockerNetwork(selectedNetwork.value.id, {
      confirm_network_name: removeConfirmation.value,
    });
    void receipt.task_id;
    removeDialogVisible.value = false;
    detailDrawerVisible.value = false;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.networks.removeFailed')));
  } finally {
    removing.value = false;
  }
}
async function submitBatchRemove() {
  if (!canRemove.value || !selectedNetworkIds.value.length) return;
  const ids = [...selectedNetworkIds.value];
  batchRemoving.value = true;
  const results = await Promise.allSettled(
    ids.map((id) => removeDockerNetwork(id, { confirm_network_name: selectedNetworkNamesByID.value[id] ?? '' })),
  );
  const succeeded = results.filter((result) => result.status === 'fulfilled').length;
  const failed = results.length - succeeded;
  clearSelection();
  batchRemoveDialogVisible.value = false;
  if (failed > 0 && succeeded === 0) MessagePlugin.error(t('container.networks.batch.removeFailed'));
  batchRemoving.value = false;
}
</script>
<style scoped>
.docker-network-page__section {
  margin-top: var(--td-comp-margin-xl);
}

.docker-network-page__detail-state {
  align-items: center;
  display: flex;
  justify-content: center;
  min-height: 240px;
  padding: var(--graft-density-gap-24) var(--graft-density-gap-16);
}

.docker-network-page__section h3 {
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--td-comp-margin-m);
}

.docker-network-page__muted {
  color: var(--td-text-color-placeholder);
}

.docker-network-page__remove-warning {
  color: var(--td-error-color);
}

.docker-network-page__identity {
  display: grid;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.docker-network-page__name {
  justify-content: flex-start;
  max-width: 100%;
}

.docker-network-page__hint,
.docker-network-page__source {
  color: var(--td-text-color-secondary);
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-network-page__hint {
  color: var(--td-text-color-placeholder);
  font-size: var(--td-font-size-body-small);
}

.docker-network-page__container-list {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-xs);
}

.docker-network-page__container-badge {
  cursor: pointer;
  max-width: 148px;
}

.docker-network-page__container-badge--overflow {
  color: var(--td-text-color-secondary);
}

.docker-network-page__container-overflow-trigger {
  background: transparent;
  border: 0;
  cursor: pointer;
  display: inline-flex;
  padding: 0;
}

.docker-network-page__container-overflow-trigger:focus-visible,
.docker-network-page__container-badge:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.docker-network-page__container-overflow {
  display: grid;
  gap: var(--td-comp-margin-s);
  max-height: min(40vh, 240px);
  max-width: 320px;
  overflow: auto;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.docker-network-page__container-overflow-title {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.docker-network-page__section h4 {
  font-size: var(--td-font-size-body-medium);
  margin: var(--td-comp-margin-l) 0 var(--td-comp-margin-m);
}

.docker-network-page__metadata-id {
  color: var(--td-text-color-secondary);
  margin: var(--td-comp-margin-m) 0 0;
  overflow-wrap: anywhere;
}

.docker-network-page__danger-zone {
  margin-top: var(--td-comp-margin-xl);
}

.docker-network-page__detail-fields,
.docker-network-page__metadata-list {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.docker-network-page__detail-fields dt,
.docker-network-page__metadata-list dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-bottom: var(--graft-density-gap-4);
}

.docker-network-page__detail-fields dd,
.docker-network-page__metadata-list dd {
  margin: 0;
  overflow-wrap: anywhere;
}

.docker-network-page__metadata-list {
  grid-template-columns: 1fr;
}

.docker-network-page__relation-cards {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.docker-network-page__relation-card {
  align-items: flex-start;
  block-size: auto;
  display: grid;
  justify-content: start;
  min-inline-size: 0;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-m);
  text-align: start;
}

.docker-network-page__relation-card strong {
  overflow-wrap: anywhere;
}

.docker-network-page__relation-card span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-network-page__mobile-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  box-shadow: var(--td-shadow-1);
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.docker-network-page__mobile-card-head {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.docker-network-page__mobile-card-name {
  color: var(--td-text-color-primary);
  min-width: 0;
  overflow-wrap: anywhere;
}

.docker-network-page__mobile-card-head :deep(.t-tag) {
  flex: 0 0 auto;
}

.docker-network-page__mobile-card-details {
  display: grid;
  gap: var(--graft-density-gap-16);
  margin: 0;
}

.docker-network-page__mobile-card-details > div {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.docker-network-page__mobile-card-details dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-network-page__mobile-card-details dd {
  color: var(--td-text-color-primary);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

@media (width < 768px) {
  .docker-network-page__detail-fields,
  .docker-network-page__relation-cards {
    grid-template-columns: 1fr;
  }

  .docker-network-page__danger-zone :deep(.t-button) {
    width: 100%;
  }
}
</style>
