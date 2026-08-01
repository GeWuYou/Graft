import type { components } from '@/contracts/openapi/generated/schema';

/** 平台更新状态由 OpenAPI 生成契约拥有，模块只导出页面需要的稳定别名。 */
type UpdateSchemas = components['schemas'];

export type UpdateStatus = UpdateSchemas['platform-update-status'];
export type UpdateChannel = UpdateStatus['channel'];
export type DeploymentStrategy = UpdateStatus['deployment_strategy'];
export type VerifiedUpdateRelease = NonNullable<UpdateStatus['available_releases']>[number];
export type UpdateCapability = UpdateStatus['installation_profile']['capability'];
export type InstallationProfile = UpdateStatus['installation_profile'];
export type UpdateRelease = NonNullable<UpdateStatus['latest']>;
export type UpdateReadiness = NonNullable<UpdateStatus['readiness']>;
export type UpdateReadinessCheck = UpdateReadiness['checks'][number];
export type UpdateReadinessAction = NonNullable<UpdateReadiness['next_action']>;
export type UpdateReadinessEvidence = UpdateReadinessCheck['evidence'][number];
export type UpdateOperation = UpdateSchemas['platform-update-operation'];
/** runner 接管后，创建请求只确认已接受的操作与执行器身份，真实阶段必须再读取状态快照。 */
export type UpdateOperationLaunchAcknowledgement = Pick<UpdateOperation, 'operation_id' | 'runner_id'>;
/** runner 状态中的阶段是当前执行事实，不能由页面自行推导或补写。 */
export type UpdateOperationPhase = UpdateOperation['phase'];
/** 更新启动失败诊断只由受权限保护的诊断接口返回，不进入通用错误响应。 */
export type UpdateFailureDiagnostic = UpdateSchemas['platform-update-failure-diagnostic'];
export type CreateUpdateOperationRequest = UpdateSchemas['create-platform-update-operation-request'];
