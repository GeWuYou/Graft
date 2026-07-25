export const CONTAINER_TASK_TYPE = {
  DOCKER_IMAGE_PULL: 'container.docker-image-pull.v1',
} as const;

export type ContainerTaskType = (typeof CONTAINER_TASK_TYPE)[keyof typeof CONTAINER_TASK_TYPE];
