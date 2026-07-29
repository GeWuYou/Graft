import { describe, expect, it } from 'vitest';

import { createUpdateCenterPreviewDataSource, updateCenterPreviewStatus } from './update-center';

describe('createUpdateCenterPreviewDataSource', () => {
  it('retains the initialized deployment policy when an operation payload omits it', async () => {
    const dataSource = createUpdateCenterPreviewDataSource();

    const operation = await dataSource.createOperation({ target_version: '0.9.8-beta.3' });

    expect(operation.update_policy).toBe(updateCenterPreviewStatus.update_policy);
  });
});
