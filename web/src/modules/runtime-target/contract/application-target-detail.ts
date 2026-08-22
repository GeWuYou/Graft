import { getRuntimeTarget, type RuntimeTargetDetail } from '../api/runtime-target';

export type ApplicationRuntimeTargetDetail = {
  displayName: string;
  endpoint: string;
  healthStatus: string;
  hostName: string;
  operatingSystem: string;
  provider: string;
  runtimeType: string;
  version: string;
  agent: RuntimeTargetDetail['agent'];
};

/**
 * 为应用概览提供运行目标的稳定展示投影，避免消费者依赖运行目标模块的私有 API 类型。
 */
export async function getApplicationRuntimeTargetDetail(id: number): Promise<ApplicationRuntimeTargetDetail> {
  const target = await getRuntimeTarget(id);
  const runtime = target.runtime as typeof target.runtime & Record<string, unknown>;

  return {
    displayName: target.displayName,
    endpoint: target.connection.endpoint,
    healthStatus: target.health.status,
    hostName: readText(runtime.hostName),
    operatingSystem: readText(runtime.operatingSystem),
    provider: readText(runtime.provider),
    runtimeType: readText(runtime.type),
    version: readText(runtime.version),
    agent: target.agent,
  };
}

function readText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}
