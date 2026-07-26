import type { paths } from '@/contracts/openapi/generated/schema';

type BackupListOperation = paths['/api/platform/backups']['get'];
type BackupListEnvelope = BackupListOperation['responses'][200]['content']['application/json'];

type BackupDetailOperation = paths['/api/platform/backups/{id}']['get'];
type BackupDetailEnvelope = BackupDetailOperation['responses'][200]['content']['application/json'];

type CreateBackupOperation = paths['/api/platform/backups']['post'];

export type BackupListResponse = NonNullable<BackupListEnvelope['data']>;
export type BackupSummary = BackupListResponse['items'][number];
export type BackupDetail = NonNullable<BackupDetailEnvelope['data']>;
export type CreateBackupRequest = NonNullable<CreateBackupOperation['requestBody']>['content']['application/json'];
export type CreateBackupReceipt = NonNullable<
  CreateBackupOperation['responses'][202]['content']['application/json']['data']
>;
export type BackupRetention = CreateBackupRequest['retention'];
