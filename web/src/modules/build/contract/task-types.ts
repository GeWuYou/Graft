export const BUILD_TASK_TYPE = {
  DOCKER_IMAGE: 'build.docker-image.v1',
} as const;

export type BuildTaskType = (typeof BUILD_TASK_TYPE)[keyof typeof BUILD_TASK_TYPE];
