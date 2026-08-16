import type { components } from '@/contracts/openapi/generated/schema';

export type BuildBuilderSnapshot = components['schemas']['build-builder-snapshot'];
export type BuildTemplateRef = components['schemas']['build-job-create-request']['template_ref'];
export type BuildDriverRef = components['schemas']['build-job-create-request']['driver'];
export type BuildExecutionProjection = components['schemas']['build-task-execution'];
export type BuildStatusFilter = components['schemas']['build-status-filter'];
export type BuildJobSummary = components['schemas']['build-job-summary'];
export type BuildJobDetail = components['schemas']['build-job-detail'];
export type BuildJobCreateRequest = components['schemas']['build-job-create-request'];
export type BuildJobListResponse = components['schemas']['build-job-list'];
export type BuildArtifact = components['schemas']['build-artifact'];
export type BuildArtifactListResponse = components['schemas']['build-artifact-list'];
export type BuildArtifactPromotionCreateRequest = components['schemas']['build-artifact-promotion-create-request'];
export type BuildBuilderPool = components['schemas']['build-builder-pool'];

// 这些引用由 OpenAPI enum 派生，避免创建表单与提交契约各自维护一份可发布意图。
export const BUILD_TEMPLATE_REF = 'oci-dockerfile/default@v1' satisfies BuildTemplateRef;
export const BUILD_DRIVER_REF = 'docker-engine@v1' satisfies BuildDriverRef;
export const BUILD_MULTI_PLATFORM_DRIVER_REF = 'docker-buildx@v1' satisfies BuildDriverRef;
export const BUILD_PLATFORM_OPTIONS = ['linux/amd64', 'linux/arm64'] as const;
