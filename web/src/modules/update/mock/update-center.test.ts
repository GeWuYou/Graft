import { describe, expect, it } from 'vitest';

import { createUpdateCenterPreviewDataSource } from './update-center';

describe('createUpdateCenterPreviewDataSource', () => {
  it('retains the initialized deployment policy when an operation payload omits it', async () => {
    const dataSource = createUpdateCenterPreviewDataSource();

    const acknowledgement = await dataSource.createOperation({ target_version: '0.9.8-beta.3' });
    const [operation] = await dataSource.getOperations();

    expect(acknowledgement.runner_id).toContain('preview-runner');
    expect(operation.target_version).toBe('0.9.8-beta.3');
  });
});
