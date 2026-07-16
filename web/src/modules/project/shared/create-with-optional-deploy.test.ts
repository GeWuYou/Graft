import { describe, expect, it, vi } from 'vitest';

import { createWithOptionalDeploy } from './create-with-optional-deploy';

describe('createWithOptionalDeploy', () => {
  it('keeps deployment off by default', async () => {
    const create = vi.fn().mockResolvedValue({ application_id: 'app_41' });
    const deploy = vi.fn();

    const result = await createWithOptionalDeploy({ create, deploy, deployAfterCreate: false });

    expect(result).toEqual({ created: { application_id: 'app_41' }, deployment: { status: 'not-requested' } });
    expect(deploy).not.toHaveBeenCalled();
  });

  it('deploys only after a successful creation', async () => {
    const create = vi.fn().mockResolvedValue({ application_id: 'app_42' });
    const deploy = vi.fn().mockResolvedValue(undefined);

    const result = await createWithOptionalDeploy({ create, deploy, deployAfterCreate: true });

    expect(result.deployment.status).toBe('succeeded');
    expect(create).toHaveBeenCalledTimes(1);
    expect(deploy).toHaveBeenCalledWith('app_42');
  });

  it('preserves successful creation when deployment fails', async () => {
    const create = vi.fn().mockResolvedValue({ application_id: 'app_43' });
    const failure = new Error('deploy unavailable');
    const deploy = vi.fn().mockRejectedValue(failure);

    const result = await createWithOptionalDeploy({ create, deploy, deployAfterCreate: true });

    expect(result.created.application_id).toBe('app_43');
    expect(result.deployment).toEqual({ status: 'failed', error: failure });
  });
});
