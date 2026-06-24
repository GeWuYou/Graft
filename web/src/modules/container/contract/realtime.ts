export const CONTAINER_REALTIME_TOPIC = {
  STATS_PREFIX: 'container.stats:',
} as const;

export type ContainerRealtimeTopicPrefix = (typeof CONTAINER_REALTIME_TOPIC)[keyof typeof CONTAINER_REALTIME_TOPIC];

export function buildContainerStatsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.STATS_PREFIX}${containerId}`;
}
