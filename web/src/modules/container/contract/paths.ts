export const CONTAINER_ROUTE_PATH = {
  LIST: '/infrastructure/docker/containers',
  DETAIL: '/infrastructure/docker/containers/:id',
  RESOURCES: '/infrastructure/docker/containers/resources',
  IMAGES: '/infrastructure/images',
  VOLUMES: '/infrastructure/docker/volumes',
  VOLUME_DETAIL: '/infrastructure/docker/volumes/:id',
} as const;
