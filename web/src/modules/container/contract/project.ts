import { batchContainerActions, getContainers } from '../api/container';
import type {
  ContainerBatchActionItem,
  ContainerBatchActionRequest,
  ContainerBatchActionResponse,
  ContainerSummaryRecord,
} from '../types/container';

export { batchContainerActions, getContainers };

export type ProjectContainerAction = Extract<ContainerBatchActionRequest['action'], 'start' | 'stop' | 'restart'>;
export type ProjectContainerActionSubmission = Omit<ContainerBatchActionRequest, 'action'> & {
  action: ProjectContainerAction;
};
export type ProjectContainerActionResult = ContainerBatchActionResponse;
export type ProjectContainerActionResultItem = ContainerBatchActionItem;
export type ProjectContainerSummary = ContainerSummaryRecord;
