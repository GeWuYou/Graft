<template>
  <div class="docker-network-page" data-page-type="list-form-detail">
    <management-page-header
      :title="t('container.networks.title')"
      :description="t('container.networks.description')"
      :source="{ labelKey: 'container.networks.eyebrow', fallback: t('container.networks.eyebrow') }"
    >
      <template #meta>
        <t-space break-line size="small">
          <t-tag variant="light-outline">{{ t('container.networks.summary.total', { count: networks.length }) }}</t-tag>
          <t-tag theme="primary" variant="light-outline">
            {{ t('container.networks.summary.inUse', { count: inUseCount }) }}
          </t-tag>
          <t-tag theme="default" variant="light-outline">
            {{ t('container.networks.summary.unused', { count: unusedCount }) }}
          </t-tag>
        </t-space>
      </template>
    </management-page-header>

    <management-toolbar class="docker-network-page__toolbar">
      <template #filters>
        <t-input v-model="filters.keyword" clearable :placeholder="t('container.networks.filters.search')">
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-select v-model="filters.driver" :placeholder="t('container.networks.filters.driver')" clearable>
          <t-option value="bridge" :label="t('container.networks.drivers.bridge')" />
          <t-option value="overlay" :label="t('container.networks.drivers.overlay')" />
          <t-option value="macvlan" :label="t('container.networks.drivers.macvlan')" />
          <t-option value="ipvlan" :label="t('container.networks.drivers.ipvlan')" />
          <t-option value="none" :label="t('container.networks.drivers.none')" />
        </t-select>
        <t-select v-model="filters.scope" :placeholder="t('container.networks.filters.scope')" clearable>
          <t-option value="local" :label="t('container.networks.scopes.local')" />
          <t-option value="swarm" :label="t('container.networks.scopes.swarm')" />
          <t-option value="global" :label="t('container.networks.scopes.global')" />
        </t-select>
        <t-select v-model="filters.usage" :placeholder="t('container.networks.filters.usage')" clearable>
          <t-option value="in-use" :label="t('container.networks.filters.inUse')" />
          <t-option value="unused" :label="t('container.networks.filters.unused')" />
        </t-select>
      </template>
      <template #actions>
        <t-space size="small">
          <t-button variant="outline" :loading="networkQuery.isFetching.value" @click="networkQuery.refetch()">
            {{ t('container.networks.refresh') }}
          </t-button>
          <t-button v-if="canCreate" theme="primary" @click="openCreateDrawer">
            {{ t('container.networks.create') }}
          </t-button>
        </t-space>
      </template>
    </management-toolbar>

    <management-batch-bar
      v-if="selectedNetworkIds.length"
      class="docker-network-page__batch"
      :selected-label="t('container.networks.batch.selected', { count: selectedNetworkIds.length })"
      :clear-label="t('container.networks.batch.cancelSelection')"
      clear-test-id="network-batch-clear"
      @clear="clearSelection"
    >
      <t-button
        data-testid="network-batch-remove"
        size="small"
        theme="danger"
        variant="outline"
        :disabled="!canRemove || batchRemoving"
        :loading="batchRemoving"
        @click="openBatchRemoveDialog"
      >
        {{ t('container.networks.batch.remove') }}
      </t-button>
    </management-batch-bar>

    <t-table
      v-model:selected-row-keys="selectedNetworkIds"
      row-key="id"
      :data="filteredNetworks"
      :columns="columns"
      :loading="networkQuery.isFetching.value"
      :pagination="paginationConfig"
      @page-change="handlePageChange"
    >
      <template #name="{ row }">
        <t-button variant="text" @click="openDetail(row.id)">{{ row.name }}</t-button>
      </template>
      <template #driver="{ row }"
        ><t-tag size="small" variant="light">{{ driverLabel(row.driver) }}</t-tag></template
      >
      <template #scope="{ row }">{{ scopeLabel(row.scope) }}</template>
      <template #flags="{ row }">
        <t-space size="small" break-line>
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
      </template>
      <template #container_count="{ row }">
        <t-tag :theme="row.container_count > 0 ? 'primary' : 'default'" size="small" variant="light">
          {{ row.container_count }}
        </t-tag>
      </template>
      <template #labels="{ row }">
        <span>{{ Object.keys(row.labels ?? {}).length }}</span>
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
    </t-table>

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
              detailQuery.data.value.created_at
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
            <h3>{{ t('container.networks.labels') }}</h3>
            <t-space v-if="Object.keys(detailQuery.data.value.labels ?? {}).length" break-line size="small">
              <t-tag v-for="(value, key) in detailQuery.data.value.labels" :key="key" variant="light"
                >{{ key }}={{ value }}</t-tag
              >
            </t-space>
            <t-empty v-else size="small" :description="t('container.networks.none')" />
          </section>
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
      <p>{{ t('container.networks.batch.removeDescription', { count: selectedNetworkIds.length }) }}</p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import { ManagementBatchBar, ManagementPageHeader, ManagementToolbar } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';

import { createDockerNetwork, removeDockerNetwork } from '../../api/container';
import {
  invalidateDockerNetworkQueries,
  useDockerNetworkDetailQuery,
  useDockerNetworkListQuery,
} from '../../shared/docker-network-queries';
import type { DockerNetwork, DockerNetworkCreateRequest, DockerNetworkDriver } from '../../types/docker-network';

defineOptions({ name: 'DockerNetworkListIndex' });

// 本页复用模块 Docker 网络列表缓存；详情单独读取，避免列表快照承载连接容器等高基数信息。
const { t } = useI18n();
const permissionStore = usePermissionStore();
const networkQuery = useDockerNetworkListQuery();
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
const selectedNetwork = ref<DockerNetwork | null>(null);
const removeConfirmation = ref('');
const drivers: DockerNetworkDriver[] = ['bridge', 'overlay', 'macvlan', 'ipvlan', 'none'];
const filters = reactive({ keyword: '', driver: '', scope: '', usage: '' });
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
const filteredNetworks = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase();
  return networks.value.filter((network) => {
    if (keyword && !`${network.name} ${network.id}`.toLowerCase().includes(keyword)) return false;
    if (filters.driver && network.driver !== filters.driver) return false;
    if (filters.scope && network.scope !== filters.scope) return false;
    return !filters.usage || (filters.usage === 'in-use' ? network.container_count > 0 : network.container_count === 0);
  });
});
const inUseCount = computed(() => networks.value.filter((network) => network.container_count > 0).length);
const unusedCount = computed(() => networks.value.length - inUseCount.value);
const canCreate = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.NETWORK_CREATE));
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.NETWORK_REMOVE));
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const paginationConfig = computed(() => ({ ...pagination, total: filteredNetworks.value.length }));
const columns: TableProps['columns'] = [
  { colKey: 'row-select', type: 'multiple', width: 48 },
  { colKey: 'name', title: t('container.networks.fields.name'), ellipsis: true },
  { colKey: 'driver', title: t('container.networks.fields.driver') },
  { colKey: 'scope', title: t('container.networks.fields.scope') },
  { colKey: 'flags', title: t('container.networks.fields.flags'), width: 180 },
  { colKey: 'container_count', title: t('container.networks.fields.containers'), width: 100 },
  { colKey: 'labels', title: t('container.networks.labels'), width: 100 },
  { colKey: 'created_at', title: t('container.networks.fields.createdAt'), ellipsis: true },
  { colKey: 'operation', title: t('container.networks.operation'), width: 150, fixed: 'right' },
];
const endpointColumns: TableProps['columns'] = [
  { colKey: 'name', title: t('container.networks.fields.name') },
  { colKey: 'id', title: t('container.networks.fields.id'), ellipsis: true },
  { colKey: 'ipv4_address', title: t('container.networks.fields.ipv4') },
  { colKey: 'mac_address', title: 'MAC' },
];

function driverLabel(driver: string) {
  return t(`container.networks.drivers.${driver}`);
}
function scopeLabel(scope: string) {
  return t(`container.networks.scopes.${scope}`, scope);
}
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
}
function openBatchRemoveDialog() {
  if (canRemove.value && selectedNetworkIds.value.length) batchRemoveDialogVisible.value = true;
}
function handlePageChange(pageInfo: { current: number; pageSize: number }) {
  pagination.current = pageInfo.current;
  pagination.pageSize = pageInfo.pageSize;
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
    ids.map((id) => {
      const network = networks.value.find((item) => item.id === id);
      return removeDockerNetwork(id, { confirm_network_name: network?.name ?? '' });
    }),
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
</style>
