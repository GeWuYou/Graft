import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const source = readFileSync(join(process.cwd(), 'src/modules/network/pages/outbound/index.vue'), 'utf8');

describe('outbound network settings page', () => {
  it('keeps the policy form, effective policy, and fixed diagnostic target in one settings surface', () => {
    expect(source).toContain('data-page-type="settings"');
    expect(source).toContain("t('network.outbound.resetToDefault')");
    expect(source).toContain("t('network.outbound.effectivePolicy')");
    expect(source).toContain('t(diagnosticTarget.value.title_key)');
    expect(source).toContain('diagnoseOutboundNetwork');
    expect(source).not.toContain('platform-update-release');
    expect(source).not.toContain('diagnosticUrl');
  });

  it('places the Docker boundary notice below the page header and above the action toolbar', () => {
    const noticeIndex = source.indexOf('class="outbound-network-page__notice"');
    const toolbarIndex = source.indexOf('<management-toolbar>');

    expect(noticeIndex).toBeGreaterThan(source.indexOf('<page-header'));
    expect(noticeIndex).toBeLessThan(toolbarIndex);
  });
});
