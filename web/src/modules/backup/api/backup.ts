import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import type {
  BackupDetail,
  BackupDetailID,
  BackupListQuery,
  BackupListResponse,
  CreateBackupReceipt,
  CreateBackupRequest,
} from '../types/backup';

export function listBackups(query?: BackupListQuery) {
  return request.get<BackupListResponse>({
    url: OPENAPI_RUNTIME_PATH.listPlatformBackups,
    params: query,
  });
}

export function getBackup(id: BackupDetailID) {
  return request.get<BackupDetail>({
    url: buildOpenApiRuntimePath('getPlatformBackup', { id }),
  });
}

export function submitBackup(data: CreateBackupRequest, idempotencyKey: string) {
  return request.post<CreateBackupReceipt>({
    url: OPENAPI_RUNTIME_PATH.postPlatformBackup,
    data,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}
