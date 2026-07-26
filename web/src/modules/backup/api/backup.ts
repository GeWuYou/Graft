import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  BackupDetail,
  BackupListResponse,
  BackupSummary,
  CreateBackupReceipt,
  CreateBackupRequest,
} from '../types/backup';

type ListBackupsPath = typeof OPENAPI_RUNTIME_PATH.listPlatformBackups;
type ListBackupsOperation = paths[ListBackupsPath]['get'];
type ListBackupsData = NonNullable<ListBackupsOperation['responses'][200]['content']['application/json']['data']>;

type GetBackupPath = typeof OPENAPI_RUNTIME_PATH.getPlatformBackup;
type GetBackupOperation = paths[GetBackupPath]['get'];
type GetBackupData = NonNullable<GetBackupOperation['responses'][200]['content']['application/json']['data']>;
type GetBackupPathParams = GetBackupOperation['parameters']['path'];

type CreateBackupPath = typeof OPENAPI_RUNTIME_PATH.postPlatformBackup;
type CreateBackupOperation = paths[CreateBackupPath]['post'];
type CreateBackupData = NonNullable<CreateBackupOperation['responses'][202]['content']['application/json']['data']>;
type CreateBackupRequestData = NonNullable<CreateBackupOperation['requestBody']>['content']['application/json'];

export type BackupListQuery = NonNullable<ListBackupsOperation['parameters']['query']>;

export function listBackups(query?: BackupListQuery) {
  return request.get<ListBackupsData>({
    url: OPENAPI_RUNTIME_PATH.listPlatformBackups,
    params: query,
  }) as Promise<BackupListResponse>;
}

export function getBackup(id: GetBackupPathParams['id']) {
  return request.get<GetBackupData>({
    url: buildOpenApiRuntimePath('getPlatformBackup', { id }),
  }) as Promise<BackupDetail>;
}

export function submitBackup(data: CreateBackupRequest, idempotencyKey: string) {
  return request.post<CreateBackupData>({
    url: OPENAPI_RUNTIME_PATH.postPlatformBackup,
    data: data as CreateBackupRequestData,
    headers: { 'Idempotency-Key': idempotencyKey },
  }) as Promise<CreateBackupReceipt>;
}

export type { BackupSummary };
