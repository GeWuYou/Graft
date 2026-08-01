import type { GraftQueryBarConfig, GraftQueryState } from '../graft-query-bar';

/** 资源列表查询的最小稳定状态；页面可在此基础上保留领域专属筛选状态。 */
export type ResourceQueryState = GraftQueryState;

export type ResourceQueryConfig = GraftQueryBarConfig & {
  resource: string;
  search?: boolean;
  filterBuilder?: { enabled: boolean };
  savedView?: boolean;
  timeRange?: boolean;
  sorting?: boolean;
  columnSetting?: boolean;
};
