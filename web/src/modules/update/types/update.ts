import type { components } from '@/contracts/openapi/generated/schema';

/** 平台更新状态由 OpenAPI 生成契约拥有，模块只导出页面需要的稳定别名。 */
type UpdateSchemas = components['schemas'];

export type UpdateStatus = UpdateSchemas['platform-update-status'];
export type UpdateChannel = UpdateStatus['channel'];
export type UpdateCapability = UpdateStatus['installation_profile']['capability'];
export type InstallationProfile = UpdateStatus['installation_profile'];
export type UpdateRelease = NonNullable<UpdateStatus['latest']>;
export type UpdateOperation = UpdateSchemas['platform-update-operation'];
export type CreateUpdateOperationRequest = UpdateSchemas['create-platform-update-operation-request'];
