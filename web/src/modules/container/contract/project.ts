import { batchContainerActions, getContainers } from '../api/container';
import type {
  ContainerBatchActionItem,
  ContainerBatchActionRequest,
  ContainerBatchActionResponse,
  ContainerSummaryRecord,
} from '../types/container';

export { batchContainerActions, getContainers };

export type ProjectContainerAction = Extract<ContainerBatchActionRequest['action'], 'start' | 'stop' | 'restart'>;
export type ProjectContainerActionSubmission = Pick<ContainerBatchActionRequest, 'action' | 'force' | 'ids'>;
export type ProjectContainerActionResult = ContainerBatchActionResponse;
export type ProjectContainerActionResultItem = ContainerBatchActionItem;
export type ProjectContainerSummary = ContainerSummaryRecord;
