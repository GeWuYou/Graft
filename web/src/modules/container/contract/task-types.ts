export const CONTAINER_TASK_TYPE = {
  DOCKER_IMAGE_PULL: 'container.docker-image-pull.v1',
  LIFECYCLE_REMOVE: 'container.lifecycle.remove.v1',
  LIFECYCLE_REMOVE_BATCH: 'container.lifecycle.remove.batch.v1',
  LIFECYCLE_RESTART: 'container.lifecycle.restart.v1',
  LIFECYCLE_RESTART_BATCH: 'container.lifecycle.restart.batch.v1',
  LIFECYCLE_START: 'container.lifecycle.start.v1',
  LIFECYCLE_START_BATCH: 'container.lifecycle.start.batch.v1',
  LIFECYCLE_STOP: 'container.lifecycle.stop.v1',
  LIFECYCLE_STOP_BATCH: 'container.lifecycle.stop.batch.v1',
} as const;

export type ContainerTaskType = (typeof CONTAINER_TASK_TYPE)[keyof typeof CONTAINER_TASK_TYPE];
