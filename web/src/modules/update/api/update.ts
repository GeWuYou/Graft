import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import type { UpdateStatus } from '../types/update';

type UpdateStatusEnvelope =
  paths[typeof UPDATE_API_PATH.STATUS]['get']['responses'][200]['content']['application/json'];
type UpdateStatusData = NonNullable<UpdateStatusEnvelope['data']>;

type CheckForUpdatesEnvelope =
  paths[typeof UPDATE_API_PATH.CHECK]['post']['responses'][200]['content']['application/json'];
type CheckForUpdatesData = NonNullable<CheckForUpdatesEnvelope['data']>;

export function getUpdateStatus() {
  return request.get<UpdateStatusData>({ url: UPDATE_API_PATH.STATUS }) as Promise<UpdateStatus>;
}

export function checkForUpdates() {
  return request.post<CheckForUpdatesData>({ url: UPDATE_API_PATH.CHECK }) as Promise<UpdateStatus>;
}
