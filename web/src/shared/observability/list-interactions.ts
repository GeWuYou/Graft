import type { Ref } from 'vue';

export function restartLogListQuery(config: {
  activePreset: Ref<string>;
  pagination: Ref<{ current: number; pageSize: number }>;
  preset?: string;
  updateRouteQuery: () => Promise<unknown> | unknown;
}) {
  config.activePreset.value = config.preset ?? 'all';
  config.pagination.value.current = 1;
  return config.updateRouteQuery();
}

/**
 * 打开日志详情记录。
 *
 * 当日志 ID 无效或小于等于 0 时，会通过 `onError` 报错并停止处理；否则会拉取详情并显示记录。
 *
 * @param config - 详情加载与显示所需配置
 */
async function openLogDetailRecord<Row, Detail>(config: {
  fetchDetail: (id: number) => Promise<Detail>;
  onError: (error: unknown) => void;
  record: Ref<Detail | null>;
  row: Row & { id: number | string };
  visible: Ref<boolean>;
}) {
  try {
    const id = Number(config.row.id);
    if (!Number.isFinite(id) || id <= 0) {
      config.onError(new Error(`Invalid log id: ${String(config.row.id)}`));
      return;
    }
    config.record.value = await config.fetchDetail(id);
    config.visible.value = true;
  } catch (error) {
    config.onError(error);
  }
}

export async function openLogDetailRow<Row extends { id: number | string }, Detail>(
  row: Row,
  fetchDetail: (id: number) => Promise<Detail>,
  record: Ref<Detail | null>,
  visible: Ref<boolean>,
  onError: (error: unknown) => void,
) {
  await openLogDetailRecord({ fetchDetail, onError, record, row, visible });
}
