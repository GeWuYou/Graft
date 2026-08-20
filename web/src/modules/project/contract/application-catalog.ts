import type { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';

import { getApplications } from '../api/project';

type ApplicationListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getApplications]['get'];
type ApplicationListQuery = NonNullable<ApplicationListOperation['parameters']['query']>;
type ApplicationListResponse = components['schemas']['ApplicationListResponse'];

export type ApplicationCatalogQuery = Pick<ApplicationListQuery, 'keyword' | 'limit' | 'offset'>;
export type ApplicationCatalogItem = Pick<
  components['schemas']['ApplicationListItem'],
  'application_id' | 'display_name'
>;
export type ApplicationCatalogResult = Omit<ApplicationListResponse, 'items'> & {
  items: ApplicationCatalogItem[];
};

/** getApplicationCatalog 为跨模块选择器提供 Project-owned 的只读 Application 目录边界。 */
export function getApplicationCatalog(query?: ApplicationCatalogQuery): Promise<ApplicationCatalogResult> {
  return getApplications(query);
}
