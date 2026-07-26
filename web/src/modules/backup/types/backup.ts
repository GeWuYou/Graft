import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';

type BackupListPath = typeof OPENAPI_RUNTIME_PATH.listPlatformBackups;
type BackupListOperation = paths[BackupListPath]['get'];
type BackupListEnvelope = BackupListOperation['responses'][200]['content']['application/json'];

type BackupDetailPath = typeof OPENAPI_RUNTIME_PATH.getPlatformBackup;
type BackupDetailOperation = paths[BackupDetailPath]['get'];
type BackupDetailEnvelope = BackupDetailOperation['responses'][200]['content']['application/json'];

type CreateBackupPath = typeof OPENAPI_RUNTIME_PATH.postPlatformBackup;
type CreateBackupOperation = paths[CreateBackupPath]['post'];

export type BackupListResponse = NonNullable<BackupListEnvelope['data']>;
export type BackupListQuery = NonNullable<BackupListOperation['parameters']['query']>;
export type BackupSummary = BackupListResponse['items'][number];
export type BackupDetail = NonNullable<BackupDetailEnvelope['data']>;
export type BackupDetailID = BackupDetailOperation['parameters']['path']['id'];
export type CreateBackupRequest = NonNullable<CreateBackupOperation['requestBody']>['content']['application/json'];
export type CreateBackupReceipt = NonNullable<
  CreateBackupOperation['responses'][202]['content']['application/json']['data']
>;
export type BackupRetention = CreateBackupRequest['retention'];
