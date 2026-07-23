import type { components } from '@/contracts/openapi/generated/schema';

/** 平台更新状态由 OpenAPI 生成契约拥有，模块只导出页面需要的稳定别名。 */
export type UpdateStatus = components['schemas']['platform-update-status'];
export type UpdateChannel = UpdateStatus['channel'];
