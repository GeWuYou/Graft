export const CONTAINER_ROUTE_PATH = {
  LIST: '/infrastructure/docker/containers',
  DETAIL: '/infrastructure/docker/containers/:id',
  RESOURCES: '/infrastructure/docker/containers/resources',
  IMAGES: '/infrastructure/images',
  VOLUMES: '/infrastructure/docker/volumes',
  VOLUME_DETAIL: '/infrastructure/docker/volumes/:id',
} as const;

export const CONTAINER_API_PATH = {
  LIST: '/api/ops/containers',
  DETAIL: '/api/ops/containers/{id}',
  RESOURCES: '/api/ops/docker/resources',
  DOCKER_VOLUMES: '/api/ops/docker/volumes',
  DOCKER_VOLUME_DETAIL: '/api/ops/docker/volumes/{id}',
  DOCKER_VOLUME_REMOVE: '/api/ops/docker/volumes/{id}/remove',
} as const;
