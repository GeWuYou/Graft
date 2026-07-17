import type { components } from '@/contracts/openapi/generated/schema';

export type DockerNetwork = components['schemas']['docker-network'];
export type DockerNetworkDetail = components['schemas']['docker-network-detail'];
export type DockerNetworkCreateRequest = components['schemas']['docker-network-create-request'];
export type DockerNetworkRemoveRequest = components['schemas']['docker-network-remove-request'];
export type DockerNetworkDriver = DockerNetworkCreateRequest['driver'];
