import type { ComputedRef, InjectionKey } from 'vue';

/** 分页表格向标准工具栏提供的列设置与密度控制边界。 */
export type ManagementTableViewTools = {
  columnSettingsLabel: ComputedRef<string>;
  densityLabel: ComputedRef<string>;
  openColumnSettings: () => void;
  toggleDensity: () => void;
};

/** 分页表格通过此上下文补齐列设置与密度行为，工具栏无需了解表格数据或页面请求逻辑。 */
export const managementTableViewToolsKey: InjectionKey<ManagementTableViewTools> =
  Symbol('management-table-view-tools');
