export const CONTAINER_PERMISSION_CODE = {
  VIEW: 'ops.container.view',
  DETAIL: 'ops.container.detail',
  EVENTS: 'ops.container.events',
  LOGS: 'ops.container.logs',
  SHELL: 'ops.container.shell',
  START: 'ops.container.start',
  STOP: 'ops.container.stop',
  RESTART: 'ops.container.restart',
  REMOVE: 'ops.container.remove',
  VOLUME_REMOVE: 'ops.container.volume.remove',
} as const;

export type ContainerPermissionCode = (typeof CONTAINER_PERMISSION_CODE)[keyof typeof CONTAINER_PERMISSION_CODE];
