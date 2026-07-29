<template>
  <t-dropdown trigger="click" placement="bottom-right">
    <t-button
      :aria-label="compact ? t('scheduledTask.list.more') : undefined"
      :title="compact ? t('scheduledTask.list.more') : undefined"
      :shape="compact ? 'square' : undefined"
      size="small"
      theme="default"
      :variant="compact ? 'text' : 'outline'"
    >
      <template #icon><ellipsis-icon /></template>
      <template v-if="!compact">{{ t('scheduledTask.list.more') }}</template>
    </t-button>
    <t-dropdown-menu>
      <t-dropdown-item v-permission="permissionCodes.RUN" :disabled="runDisabled" @click="emit('run', task)">
        <template #prefix-icon><play-icon /></template>
        {{ t('scheduledTask.list.run') }}
      </t-dropdown-item>
      <t-dropdown-item v-permission="permissionCodes.UPDATE" @click="emit('edit', task)">
        <template #prefix-icon><edit-icon /></template>
        {{ t('scheduledTask.list.edit') }}
      </t-dropdown-item>
      <t-dropdown-item v-permission="permissionCodes.ENABLE" :disabled="lifecyclePending" @click="emit('toggle', task)">
        <template #prefix-icon>
          <pause-icon v-if="task.enabled" />
          <play-icon v-else />
        </template>
        {{ task.enabled ? t('scheduledTask.list.disable') : t('scheduledTask.list.enable') }}
      </t-dropdown-item>
      <t-dropdown-item
        v-permission="permissionCodes.DELETE"
        :disabled="deleteDisabled"
        theme="error"
        @click="emit('delete', task)"
      >
        <template #prefix-icon><delete-icon /></template>
        {{ t('scheduledTask.list.delete') }}
      </t-dropdown-item>
    </t-dropdown-menu>
  </t-dropdown>
</template>
<script setup lang="ts">
import { DeleteIcon, EditIcon, EllipsisIcon, PauseIcon, PlayIcon } from 'tdesign-icons-vue-next';
import { useI18n } from 'vue-i18n';

import { SCHEDULED_TASK_PERMISSION_CODE } from '../contract/permissions';
import type { ScheduledTaskItem } from '../types/scheduled-task';

// 该组件统一列表与卡片的任务操作菜单，页面继续拥有操作的对话框、请求和生命周期状态。
withDefaults(
  defineProps<{
    task: ScheduledTaskItem;
    compact?: boolean;
    runDisabled: boolean;
    lifecyclePending: boolean;
    deleteDisabled: boolean;
  }>(),
  { compact: false },
);

const emit = defineEmits<{
  run: [task: ScheduledTaskItem];
  edit: [task: ScheduledTaskItem];
  toggle: [task: ScheduledTaskItem];
  delete: [task: ScheduledTaskItem];
}>();

const { t } = useI18n();
const permissionCodes = SCHEDULED_TASK_PERMISSION_CODE;
</script>
