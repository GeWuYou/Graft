/**
 * 向其他功能模块暴露的稳定任务观察边界。
 */
export type { TaskObserver, TaskObserverOptions } from '../task-observer';
export { isTerminalTaskStatus, observeTask } from '../task-observer';
