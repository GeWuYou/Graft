<template>
  <div class="docker-network-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      :title="t('container.networks.title')"
      :description="t('container.networks.description')"
      :source="{ labelKey: 'container.networks.eyebrow', fallback: t('container.networks.eyebrow') }"
    >
      <template #actions>
        <t-space>
          <t-button v-if="canRemove" variant="outline" @click="openCleanup">
            {{ t('container.networks.cleanup.action') }}
          </t-button>
          <t-button v-if="canCreate" theme="primary" @click="openCreateDrawer">
            {{ t('container.networks.create') }}
          </t-button>
        </t-space>
      </template>
    </management-page-header>

    <management-statistics-bar :items="metrics" aria-live="polite" />

    <management-toolbar class="docker-network-page__toolbar">
      <template #filters>
        <t-input
          v-model="draftFilters.keyword"
          class="management-list-search"
          clearable
          :placeholder="t('container.networks.filters.search')"
          @enter="applyFilters"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-select
          v-model="draftFilters.driver"
          class="management-toolbar__select"
          :placeholder="t('container.networks.filters.driver')"
          clearable
        >
          <t-option value="bridge" :label="t('container.networks.drivers.bridge')" />
          <t-option value="overlay" :label="t('container.networks.drivers.overlay')" />
          <t-option value="macvlan" :label="t('container.networks.drivers.macvlan')" />
          <t-option value="ipvlan" :label="t('container.networks.drivers.ipvlan')" />
          <t-option value="none" :label="t('container.networks.drivers.none')" />
        </t-select>
        <t-select
          v-model="draftFilters.scope"
          class="management-toolbar__select"
          :placeholder="t('container.networks.filters.scope')"
          clearable
        >
          <t-option value="local" :label="t('container.networks.scopes.local')" />
          <t-option value="swarm" :label="t('container.networks.scopes.swarm')" />
          <t-option value="global" :label="t('container.networks.scopes.global')" />
        </t-select>
        <t-select
          v-model="draftFilters.usage"
          class="management-toolbar__select"
          :placeholder="t('container.networks.filters.usage')"
          clearable
        >
          <t-option value="used" :label="t('container.networks.filters.inUse')" />
          <t-option value="unused" :label="t('container.networks.filters.unused')" />
        </t-select>
        <t-button theme="primary" @click="applyFilters">{{ t('container.networks.filters.apply') }}</t-button>
        <t-button variant="text" @click="resetFilters">{{ t('container.networks.filters.reset') }}</t-button>
      </template>
    </management-toolbar>

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
      :columns="columns"
      :loading="networkQuery.isFetching.value"
      :selected-row-keys="selectedNetworkIds"
      :footer-summary="paginationSummary"
      :empty-title="t('container.networks.emptyTitle')"
      :empty-description="t('container.networks.emptyDescription')"
      :cell-slot-names="[
        'name',
        'driver',
        'scope',
        'flags',
        'container_count',
        'resource_source',
        'created_at',
        'operation',
      ]"
      @select-change="handleSelectChange"
      @page-change="handlePageChange"
    >
      <template #toolbar>
        <table-view-toolbar
          :refresh-label="t('container.networks.refresh')"
          :refresh-loading="networkQuery.isFetching.value"
          :column-settings-label="t('container.list.columnSettings')"
          @refresh="networkQuery.refetch"
          @column-settings="columnDrawerVisible = true"
        />
      </template>
      <template #batch>
        <management-batch-bar
          v-if="selectedNetworkIds.length"
          :selected-label="t('container.networks.batch.selected', { count: selectedNetworkIds.length })"
          :clear-label="t('container.networks.batch.cancelSelection')"
          @clear="clearSelection"
        >
          <t-button v-if="canRemove" theme="danger" variant="outline" @click="openBatchRemoveDialog">{{
            t('container.networks.batch.remove')
          }}</t-button>
        </management-batch-bar>
      </template>
      <template #name="{ row }">
        <t-button variant="text" @click="openDetail(row.id)">{{ row.name }}</t-button>
      </template>
      <template #driver="{ row }"
        ><t-tag size="small" variant="light">{{ driverLabel(row.driver) }}</t-tag></template
      >
      <template #scope="{ row }">{{ scopeLabel(row.scope) }}</template>
      <template #flags="{ row }">
        <t-space v-if="row.internal || row.attachable || row.ingress" size="small" break-line>
          <t-tag v-if="row.internal" size="small" theme="warning" variant="light">{{
            t('container.networks.internal')
          }}</t-tag>
          <t-tag v-if="row.attachable" size="small" theme="primary" variant="light">{{
            t('container.networks.attachable')
          }}</t-tag>
          <t-tag v-if="row.ingress" size="small" theme="success" variant="light">{{
            t('container.networks.ingress')
          }}</t-tag>
        </t-space>
        <span v-else class="docker-network-page__muted">{{ t('container.networks.noAttributes') }}</span>
      </template>
      <template #container_count="{ row }">
        <t-tag :theme="row.container_count > 0 ? 'primary' : 'default'" size="small" variant="light">
          {{ row.container_count }}
        </t-tag>
      </template>
      <template #resource_source="{ row }">
        <t-popup attach="body" destroy-on-close show-arrow trigger="click" placement="bottom-left">
          <template #content>
            <div class="docker-network-resource-source-popup">
              <section class="docker-network-resource-source-popup__section">
                <h4>{{ t('container.networks.resourceSource') }}</h4>
                <t-descriptions :column="1" size="small" table-layout="auto">
                  <t-descriptions-item :label="t('container.networks.source.fields.kind')">
                    {{ sourceKindLabel(resourceSource(row).kind) }}
                  </t-descriptions-item>
                  <t-descriptions-item
                    v-for="field in resourceSource(row).sourceFields"
                    :key="field.key"
                    :label="t(`container.networks.source.fields.${field.key}`)"
                  >
                    <span class="docker-network-page__value">{{ field.value }}</span>
                  </t-descriptions-item>
                </t-descriptions>
              </section>
              <resource-label-group
                :title="t('container.networks.systemLabels')"
                :labels="resourceSource(row).systemLabels"
              />
              <resource-label-group
                :title="t('container.networks.userLabels')"
                :labels="resourceSource(row).userLabels"
              />
            </div>
          </template>
          <button
            type="button"
            class="docker-network-resource-source-trigger"
            :aria-label="
              t('container.networks.source.openDetails', { source: sourceKindLabel(resourceSource(row).kind) })
            "
          >
            <span>{{ sourceIdentityLabel(resourceSource(row)) }}</span>
            <span class="docker-network-resource-source-trigger__labels">{{
              userLabelSummary(resourceSource(row))
            }}</span>
          </button>
        </t-popup>
      </template>
      <template #created_at="{ row }">
        <t-tooltip :content="formatLocaleDateTime(row.created_at, locale)">
          <span>{{ formatLocaleDateTime(row.created_at, locale) }}</span>
        </t-tooltip>
      </template>
      <template #operation="{ row }">
        <t-space size="small">
          <t-button variant="text" @click="openDetail(row.id)">{{ t('container.networks.detail') }}</t-button>
          <t-button v-if="canRemove" theme="danger" variant="text" @click="openRemoveDialog(row)">
            {{ t('container.networks.remove') }}
          </t-button>
        </t-space>
      </template>
      <template #empty>
        <t-empty :title="t('container.networks.emptyTitle')" :description="t('container.networks.emptyDescription')" />
      </template>
    </management-paged-table>

    <t-drawer v-model:visible="columnDrawerVisible" :header="t('container.list.columnSettings')" size="360px">
      <t-checkbox-group v-model="visibleColumnKeys" :options="columnOptions" />
    </t-drawer>

    <t-dialog
      v-model:visible="cleanup.visible.value"
      :header="t('container.networks.cleanup.title')"
      width="760px"
      @confirm="submitCleanup"
    >
      <t-loading :loading="cleanup.loading.value">
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
      </t-loading>
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
        <t-form-item :label="t('container.networks.form.labels')">
          <t-textarea
            v-model="createForm.labelsText"
            :autosize="{ minRows: 3, maxRows: 6 }"
            :placeholder="t('container.networks.form.labelsHint')"
          />
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

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('container.networks.detailTitle')"
      size="720px"
      destroy-on-close
      :footer="false"
    >
      <t-loading :loading="detailQuery.isFetching.value">
        <template v-if="detailQuery.data.value">
          <t-descriptions bordered :column="2" table-layout="auto">
            <t-descriptions-item :label="t('container.networks.fields.name')">{{
              detailQuery.data.value.name
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('container.networks.fields.id')">{{
              detailQuery.data.value.id
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('container.networks.fields.driver')">{{
              detailQuery.data.value.driver
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('container.networks.fields.scope')">{{
              detailQuery.data.value.scope
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('container.networks.fields.createdAt')">{{
              formatLocaleDateTime(detailQuery.data.value.created_at, locale)
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('container.networks.fields.containers')">{{
              detailQuery.data.value.container_count
            }}</t-descriptions-item>
          </t-descriptions>
          <section class="docker-network-page__section">
            <h3>{{ t('container.networks.ipam') }}</h3>
            <t-descriptions v-if="detailQuery.data.value.ipam" bordered :column="2">
              <t-descriptions-item :label="t('container.networks.fields.driver')">{{
                detailQuery.data.value.ipam.driver || '-'
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.networks.form.subnet')">{{
                detailQuery.data.value.ipam.config?.[0]?.subnet || '-'
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.networks.form.gateway')">{{
                detailQuery.data.value.ipam.config?.[0]?.gateway || '-'
              }}</t-descriptions-item>
            </t-descriptions>
            <t-empty v-else size="small" :description="t('container.networks.none')" />
          </section>
          <section class="docker-network-page__section">
            <h3>{{ t('container.networks.resourceSource') }}</h3>
            <t-descriptions bordered :column="1" size="small" table-layout="auto">
              <t-descriptions-item :label="t('container.networks.source.fields.kind')">
                {{ sourceKindLabel(resourceSource(detailQuery.data.value).kind) }}
              </t-descriptions-item>
              <t-descriptions-item
                v-for="field in resourceSource(detailQuery.data.value).sourceFields"
                :key="field.key"
                :label="t(`container.networks.source.fields.${field.key}`)"
              >
                <span class="docker-network-page__value">{{ field.value }}</span>
              </t-descriptions-item>
            </t-descriptions>
          </section>
          <resource-label-group
            :title="t('container.networks.systemLabels')"
            :labels="resourceSource(detailQuery.data.value).systemLabels"
            detail
          />
          <resource-label-group
            :title="t('container.networks.userLabels')"
            :labels="resourceSource(detailQuery.data.value).userLabels"
            detail
          />
          <section class="docker-network-page__section">
            <h3>{{ t('container.networks.connectedContainers') }}</h3>
            <t-table
              row-key="id"
              size="small"
              :data="detailQuery.data.value.containers ?? []"
              :columns="endpointColumns"
            />
          </section>
        </template>
      </t-loading>
    </t-drawer>

    <t-dialog
      v-model:visible="removeDialogVisible"
      :header="t('container.networks.removeTitle')"
      :confirm-btn="t('container.networks.remove')"
      :confirm-loading="removing"
      theme="danger"
      @confirm="submitRemove"
    >
      <p>{{ t('container.networks.removeDescription', { name: selectedNetwork?.name ?? '' }) }}</p>
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
import { ArrowDownIcon, ArrowUpIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, defineComponent, h, reactive, ref, resolveComponent, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import {
  ManagementBatchBar,
  ManagementPagedTable,
  ManagementPageHeader,
  ManagementStatisticsBar,
  ManagementToolbar,
  TableViewToolbar,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability/time';
import { usePermissionStore } from '@/store';

import {
  createDockerNetwork,
  type DockerNetworkListQuery,
  getDockerNetworks,
  removeDockerNetwork,
} from '../../api/container';
import { CONTAINER_NETWORK_COLUMN_STORAGE_KEY } from '../../contract/storage';
import { useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';
import {
  invalidateDockerNetworkQueries,
  useDockerNetworkDetailQuery,
  useDockerNetworkListQuery,
} from '../../shared/docker-network-queries';
import type { DockerNetwork, DockerNetworkCreateRequest, DockerNetworkDriver } from '../../types/docker-network';
import { presentNetworkResourceSource, type ResourceSourcePresentation } from './resource-source-presenter';

defineOptions({ name: 'DockerNetworkListIndex' });

/** 网络页消费服务端规范化的来源与标签分组；原始 labels 只保留在创建请求，避免列表重复推导 Docker 元数据。 */
const { locale, t } = useI18n();
const permissionStore = usePermissionStore();
const pagination = reactive({ current: 1, pageSize: 20 });
const draftFilters = reactive({ keyword: '', driver: '', scope: '', usage: '' });
const appliedFilters = ref({ ...draftFilters });
const networkListQuery = computed<DockerNetworkListQuery>(() => ({
  limit: pagination.pageSize,
  offset: (pagination.current - 1) * pagination.pageSize,
  keyword: appliedFilters.value.keyword || undefined,
  driver: appliedFilters.value.driver || undefined,
  scope: appliedFilters.value.scope || undefined,
  usage: appliedFilters.value.usage ? (appliedFilters.value.usage as 'used' | 'unused') : undefined,
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
  labelsText: '',
  subnet: '',
  gateway: '',
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
let cleanup: ReturnType<typeof useDockerCleanup<DockerNetwork>>;
cleanup = useDockerCleanup<DockerNetwork>({
  fetchCandidates: fetchCleanupCandidates,
  execute: removeCleanupCandidates,
});
const columnDrawerVisible = ref(false);
const columnOptions = [
  { label: t('container.networks.fields.driver'), value: 'driver' },
  { label: t('container.networks.fields.scope'), value: 'scope' },
  { label: t('container.networks.fields.flags'), value: 'flags' },
  { label: t('container.networks.fields.containers'), value: 'container_count' },
  { label: t('container.networks.resourceSource'), value: 'resource_source' },
];
const defaultColumnKeys = columnOptions.map((item) => item.value);
const visibleColumnKeys = ref<string[]>(
  typeof localStorage === 'undefined' ? defaultColumnKeys : readVisibleColumnKeys(),
);
function readVisibleColumnKeys() {
  try {
    const stored = JSON.parse(localStorage.getItem(CONTAINER_NETWORK_COLUMN_STORAGE_KEY) ?? 'null');
    return Array.isArray(stored) ? stored.filter((key): key is string => typeof key === 'string') : defaultColumnKeys;
  } catch {
    return defaultColumnKeys;
  }
}
watch(visibleColumnKeys, (value) => localStorage.setItem(CONTAINER_NETWORK_COLUMN_STORAGE_KEY, JSON.stringify(value)), {
  deep: true,
});
const allColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.networks.fields.name'), width: 360, ellipsis: true },
  { colKey: 'driver', title: t('container.networks.fields.driver'), width: 120, ellipsis: true },
  { colKey: 'scope', title: t('container.networks.fields.scope'), width: 120, ellipsis: true },
  { colKey: 'flags', title: t('container.networks.fields.flags'), width: 180 },
  { colKey: 'container_count', title: t('container.networks.fields.containers'), width: 100 },
  { colKey: 'resource_source', title: t('container.networks.resourceSource'), width: 260 },
  { colKey: 'created_at', title: t('container.networks.fields.createdAt'), width: 180, ellipsis: true },
  { colKey: 'operation', title: t('container.networks.operation'), width: 150, fixed: 'right' as const },
]);
const columns = computed<TableProps['columns']>(() =>
  (allColumns.value ?? []).filter(
    (column) =>
      ['row-select', 'name', 'operation', 'created_at'].includes(String(column.colKey)) ||
      visibleColumnKeys.value.includes(String(column.colKey)),
  ),
);
const cleanupColumns = computed<TableProps['columns']>(() =>
  (allColumns.value ?? []).filter((column) =>
    ['row-select', 'name', 'driver', 'scope', 'container_count'].includes(String(column.colKey)),
  ),
);
const endpointColumns: TableProps['columns'] = [
  { colKey: 'name', title: t('container.networks.fields.name') },
  { colKey: 'id', title: t('container.networks.fields.id'), ellipsis: true },
  { colKey: 'ipv4_address', title: t('container.networks.fields.ipv4') },
  { colKey: 'mac_address', title: 'MAC' },
];
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
function handleCleanupSelectChange(keys: Array<string | number>) {
  cleanup.select(keys);
}

function driverLabel(driver: string) {
  return t(`container.networks.drivers.${driver}`, driver);
}
function scopeLabel(scope: string) {
  return t(`container.networks.scopes.${scope}`, scope);
}
function resourceSource(network: DockerNetwork): ResourceSourcePresentation {
  return presentNetworkResourceSource(network);
}
function sourceKindLabel(kind: ResourceSourcePresentation['kind']) {
  return t(`container.networks.source.kinds.${kind}`);
}
function sourceIdentityLabel(source: ResourceSourcePresentation) {
  const kind = sourceKindLabel(source.kind);
  return source.identity ? `${kind} · ${source.identity}` : kind;
}
function userLabelSummary(source: ResourceSourcePresentation) {
  if (!source.userLabel) return t('container.networks.source.noUserLabels');
  return source.remainingUserLabelCount ? `${source.userLabel} +${source.remainingUserLabelCount}` : source.userLabel;
}
const ResourceLabelGroup = defineComponent({
  name: 'ResourceLabelGroup',
  props: {
    title: { type: String, required: true },
    labels: { type: Array as () => Array<[string, string]>, required: true },
    detail: Boolean,
  },
  setup(props) {
    const descriptions = resolveComponent('t-descriptions');
    const descriptionsItem = resolveComponent('t-descriptions-item');
    return () =>
      h('section', { class: ['docker-network-label-group', { 'docker-network-label-group--detail': props.detail }] }, [
        h('h3', props.title),
        props.labels.length
          ? h(
              descriptions,
              { bordered: props.detail, column: 1, size: 'small', tableLayout: 'auto' },
              {
                default: () =>
                  props.labels.map(([key, value]) =>
                    h(
                      descriptionsItem,
                      { label: key },
                      { default: () => h('span', { class: 'docker-network-page__value', title: value }, value) },
                    ),
                  ),
              },
            )
          : h('span', { class: 'docker-network-page__muted' }, t('container.networks.none')),
      ]);
  },
});
function openCreateDrawer() {
  Object.assign(createForm, {
    name: '',
    driver: 'bridge',
    internal: false,
    attachable: false,
    labelsText: '',
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
  };
  pagination.current = 1;
}
function resetFilters() {
  Object.assign(draftFilters, { keyword: '', driver: '', scope: '', usage: '' });
  applyFilters();
}
function parseLabels(source: string) {
  const labels = Object.fromEntries(
    source
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const separator = line.indexOf('=');
        return separator < 0 ? [line, ''] : [line.slice(0, separator).trim(), line.slice(separator + 1).trim()];
      }),
  );
  return Object.keys(labels).length ? labels : undefined;
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
    labels: parseLabels(createForm.labelsText),
    ipam,
  };
  creating.value = true;
  try {
    await createDockerNetwork(body);
    await invalidateDockerNetworkQueries();
    createDrawerVisible.value = false;
    MessagePlugin.success(t('container.networks.createSuccess'));
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
    await removeDockerNetwork(selectedNetwork.value.id, { confirm_network_name: removeConfirmation.value });
    await invalidateDockerNetworkQueries();
    removeDialogVisible.value = false;
    detailDrawerVisible.value = false;
    MessagePlugin.success(t('container.networks.removeSuccess'));
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
  await invalidateDockerNetworkQueries();
  batchRemoveDialogVisible.value = false;
  if (failed === 0) MessagePlugin.success(t('container.networks.batch.removeSuccess', { count: succeeded }));
  else if (succeeded > 0) MessagePlugin.warning(t('container.networks.batch.removePartial', { succeeded, failed }));
  else MessagePlugin.error(t('container.networks.batch.removeFailed'));
  batchRemoving.value = false;
}
</script>
<style scoped>
.docker-network-page__toolbar {
  margin-bottom: var(--td-comp-margin-l);
}

.docker-network-page__toolbar :deep(.management-list-search) {
  flex-basis: clamp(220px, 22vw, 280px);
  width: clamp(220px, 22vw, 280px);
}

.docker-network-page__toolbar :deep(.management-toolbar__select) {
  flex-basis: clamp(140px, 15vw, 180px);
  width: clamp(140px, 15vw, 180px);
}

.docker-network-page__batch {
  margin-bottom: var(--td-comp-margin-m);
}

.docker-network-page__section {
  margin-top: var(--td-comp-margin-xl);
}

.docker-network-page__section h3 {
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--td-comp-margin-m);
}

.docker-network-page__muted {
  color: var(--td-text-color-placeholder);
}

.docker-network-page__value {
  display: block;
  overflow-wrap: anywhere;
}

.docker-network-resource-source-trigger {
  background: transparent;
  border: 0;
  color: var(--td-text-color-primary);
  cursor: pointer;
  display: grid;
  font: inherit;
  gap: var(--td-comp-margin-xs);
  max-width: 100%;
  overflow: hidden;
  padding: 0;
  text-align: left;
}

.docker-network-resource-source-trigger:hover,
.docker-network-resource-source-trigger:focus-visible {
  color: var(--td-brand-color);
  outline: none;
}

.docker-network-resource-source-trigger:focus-visible {
  border-radius: var(--td-radius-small);
  box-shadow: 0 0 0 2px var(--td-brand-color-focus);
}

.docker-network-resource-source-trigger > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-network-resource-source-trigger__labels {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.docker-network-resource-source-popup {
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-m);
  width: min(420px, calc(100vw - var(--td-comp-margin-xxl)));
}

.docker-network-resource-source-popup__section,
.docker-network-label-group {
  margin-top: var(--td-comp-margin-m);
}

.docker-network-resource-source-popup__section:first-child {
  margin-top: 0;
}

.docker-network-resource-source-popup h4,
.docker-network-label-group h3 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-body-medium);
  font-weight: var(--td-font-weight-medium);
  margin: 0 0 var(--td-comp-margin-s);
}
</style>
