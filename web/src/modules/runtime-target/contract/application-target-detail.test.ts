import { describe, expect, it, vi } from 'vitest';

import { getApplicationRuntimeTargetDetail } from './application-target-detail';

const api = vi.hoisted(() => ({ getRuntimeTarget: vi.fn() }));
vi.mock('../api/runtime-target', () => api);

describe('Application runtime target public projection', () => {
  const target = {
    displayName: 'Local Docker',
    connection: { endpoint: 'unix:///var/run/docker.sock' },
    health: { status: 'healthy' },
    runtime: {
      provider: 'docker',
      type: 'container_runtime',
      version: '29.9.3',
      hostName: 'host',
      operatingSystem: 'Linux',
    },
    agent: { status: 'not_enrolled', capabilities: [] },
  };

  it('preserves server-owned local execution readiness for application consumers', async () => {
    const execution = {
      mode: 'local',
      ready: true,
      diagnostic_code: 'execution_ready',
      capabilities: ['container_execution', 'compose_execution'],
    };
    api.getRuntimeTarget.mockResolvedValue({ ...target, execution });
    const projection = await getApplicationRuntimeTargetDetail(7);
    expect(projection.execution).toEqual(execution);
    expect(projection.agent.capabilities).toEqual([]);
    expect(api.getRuntimeTarget).toHaveBeenCalledWith(7);
  });

  it('keeps older responses readable without inferring execution readiness', async () => {
    api.getRuntimeTarget.mockResolvedValue(target);
    const projection = await getApplicationRuntimeTargetDetail(7);
    expect(projection.execution).toBeUndefined();
    expect(projection.agent).toEqual(target.agent);
  });
});
