import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import type { UpdateStatus } from '../types/update';

export function getUpdateStatus() {
  return request.get<UpdateStatus>({ url: UPDATE_API_PATH.STATUS }) as Promise<UpdateStatus>;
}

export function checkForUpdates() {
  return request.post<UpdateStatus>({ url: UPDATE_API_PATH.CHECK }) as Promise<UpdateStatus>;
}
