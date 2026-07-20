export const CONTAINER_ROUTE_PATH = {
  LIST: '/infrastructure/docker/containers',
  DETAIL: '/infrastructure/docker/containers/:id',
  RESOURCES: '/infrastructure/docker/containers/resources',
  NETWORKS: '/infrastructure/docker/networks',
  IMAGES: '/infrastructure/images',
  VOLUMES: '/infrastructure/docker/volumes',
} as const;
