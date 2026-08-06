import type { components } from '@/contracts/openapi/generated/schema';

export type BuildBuilderSnapshot = components['schemas']['build-builder-snapshot'];
export type BuildExecutionProjection = components['schemas']['build-task-execution'];
export type BuildStatusFilter = components['schemas']['build-status-filter'];
export type BuildJobSummary = components['schemas']['build-job-summary'];
export type BuildJobDetail = components['schemas']['build-job-detail'];
export type BuildJobCreateRequest = components['schemas']['build-job-create-request'];
export type BuildJobListResponse = components['schemas']['build-job-list'];
export type BuildArtifact = components['schemas']['build-artifact'];
export type BuildArtifactListResponse = components['schemas']['build-artifact-list'];
export type BuildArtifactPromotionCreateRequest = components['schemas']['build-artifact-promotion-create-request'];
