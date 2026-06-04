import type { Ref } from 'vue';

export function restartLogListQuery(config: {
  activePreset: Ref<string>;
  pagination: Ref<{ current: number; pageSize: number }>;
  preset?: string;
  updateRouteQuery: () => Promise<unknown> | unknown;
}) {
  config.activePreset.value = config.preset ?? 'all';
  config.pagination.value.current = 1;
  void config.updateRouteQuery();
}

async function openLogDetailRecord<Row, Detail>(config: {
  fetchDetail: (id: number) => Promise<Detail>;
  onError: (error: unknown) => void;
  record: Ref<Detail | null>;
  row: Row & { id: number | string };
  visible: Ref<boolean>;
}) {
  try {
    config.record.value = await config.fetchDetail(Number(config.row.id));
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
