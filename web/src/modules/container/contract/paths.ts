export const CONTAINER_ROUTE_PATH = {
  LIST: '/infrastructure/docker/containers',
  DETAIL: '/infrastructure/docker/containers/:id',
  NETWORKS: '/infrastructure/docker/networks',
  IMAGES: '/infrastructure/images',
  VOLUMES: '/infrastructure/docker/volumes',
  VOLUME_DETAIL: '/infrastructure/docker/volumes/:name',
} as const;
