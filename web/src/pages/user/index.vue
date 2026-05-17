<template>
  <div class="user-page">
    <t-row :gutter="[24, 24]">
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.userList.listTitle')">
          <div class="summary-metric">
            <span class="summary-metric__label">{{ t('pages.userList.countLabel') }}</span>
            <span class="summary-metric__value">{{ users.length }}</span>
          </div>
          <div class="summary-hint">{{ t('pages.userList.hint') }}</div>
        </t-card>
      </t-col>
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.userList.apiTitle')">
          <div class="summary-meta">
            <span
              >{{ t('pages.userList.endpointLabel') }}<code>{{ t('pages.userList.endpointValue') }}</code></span
            >
            <span
              >{{ t('pages.userList.fieldsLabel') }}<code>{{ t('pages.userList.fieldsValue') }}</code></span
            >
          </div>
          <div class="summary-actions">
            <t-button theme="primary" variant="outline" :loading="loading" @click="fetchUsers">
              {{ t('pages.userList.refresh') }}
            </t-button>
            <t-button v-permission="permissionCodes.CREATE" theme="default" variant="base" disabled>
              {{ t('pages.listBase.create') }}
            </t-button>
          </div>
        </t-card>
      </t-col>
    </t-row>

    <t-card class="table-card" :bordered="false" :title="t('pages.userList.dataTitle')">
      <t-table
        row-key="id"
        :data="users"
        :columns="columns"
        :loading="loading"
        size="medium"
        :table-layout="showOperationColumn ? 'fixed' : 'auto'"
      >
        <template #operation>
          <div class="operation-cell">
            <t-button v-permission="permissionCodes.UPDATE" variant="text" theme="primary" disabled>
              {{ t('components.commonTable.detail') }}
            </t-button>
            <t-button v-permission="permissionCodes.DISABLE" variant="text" theme="danger" disabled>
              {{ t('components.manage') }}
            </t-button>
          </div>
        </template>
        <template #empty>
          <t-empty :description="t('pages.userList.empty')" />
        </template>
      </t-table>
    </t-card>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { UserListItem } from '@/api/model/userModel';
import { getUsers } from '@/api/user';
import { USER_PERMISSION_CODE } from '@/contracts/user/permissions';
import { usePermissionStore } from '@/store';

defineOptions({
  name: 'UsersIndex',
});

const { t, locale } = useI18n();
const permissionStore = usePermissionStore();
const users = ref<UserListItem[]>([]);
const loading = ref(false);
const permissionCodes = USER_PERMISSION_CODE;

const showOperationColumn = computed(() =>
  permissionStore.hasAnyPermission([permissionCodes.UPDATE, permissionCodes.DISABLE]),
);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  void showOperationColumn.value;

  const baseColumns: TdBaseTableProps['columns'] = [
    {
      title: t('pages.userList.columns.id'),
      colKey: 'id',
      width: 100,
    },
    {
      title: t('pages.userList.columns.username'),
      colKey: 'username',
      minWidth: 180,
    },
    {
      title: t('pages.userList.columns.display'),
      colKey: 'display',
      minWidth: 180,
    },
    {
      title: t('pages.userList.columns.createdAt'),
      colKey: 'created_at',
      minWidth: 220,
    },
    {
      title: t('pages.userList.columns.updatedAt'),
      colKey: 'updated_at',
      minWidth: 220,
    },
  ];

  if (showOperationColumn.value) {
    baseColumns.push({
      title: t('components.commonTable.operation'),
      colKey: 'operation',
      width: 220,
      fixed: 'right',
    });
  }

  return baseColumns;
});

async function fetchUsers() {
  loading.value = true;
  try {
    // 后端契约假设：GET /api/users 始终返回 { items: UserListItem[] }，且时间字段已可直接展示。
    const response = await getUsers();
    users.value = response.items;
  } catch (error) {
    users.value = [];
    MessagePlugin.error(error instanceof Error ? error.message : '用户列表加载失败');
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  fetchUsers();
});
</script>
<style lang="less" scoped>
@import './index.less';

.summary-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.operation-cell {
  display: flex;
  gap: var(--td-comp-margin-xs);
  justify-content: flex-start;
}
</style>
