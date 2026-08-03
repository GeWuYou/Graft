import type { components } from '@/contracts/openapi/generated/schema';

/** 平台出站网络策略输入只描述代理与绕过规则，不承载 HTTP 客户端超时或重试等调用行为。 */
export type OutboundNetworkConfig = components['schemas']['platform-network-outbound-config'];
/** 生效策略及其配置来源。 */
export type OutboundNetworkPolicy = components['schemas']['platform-network-outbound-policy'];
export type OutboundNetworkOverview = components['schemas']['platform-network-overview'];
export type OutboundNetworkDiagnostic = components['schemas']['platform-network-diagnostic-result'];
export type OutboundNetworkDiagnosticTarget = components['schemas']['platform-network-diagnostic-target'];
export type OutboundNetworkDiagnosticHistory = components['schemas']['platform-network-diagnostic-history'];
export type OutboundNetworkConsumer = components['schemas']['platform-network-consumer'];
