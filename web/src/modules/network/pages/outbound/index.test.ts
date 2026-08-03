import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const source = readFileSync(join(process.cwd(), 'src/modules/network/pages/outbound/index.vue'), 'utf8');

describe('outbound network settings page', () => {
  it('organizes overview, runtime, routing, and persisted diagnostics into one settings surface', () => {
    expect(source).toContain('data-page-type="settings"');
    expect(source).toContain("t('network.outbound.resetToDefault')");
    expect(source).toContain("t('network.outbound.overview.kicker')");
    expect(source).toContain("t('network.outbound.runtime.title')");
    expect(source).toContain("t('network.outbound.routing.title')");
    expect(source).toContain("t('network.outbound.diagnostics.history')");
    expect(source).toContain('diagnoseOutboundNetwork');
    expect(source).toContain('getOutboundNetworkDiagnosticHistory');
    expect(source).not.toContain('platform-update-release');
    expect(source).not.toContain('diagnosticUrl');
  });

  it('keeps policy scope beneath the page header and places runtime beside configuration', () => {
    const scopeIndex = source.indexOf('class="outbound-network-page__scope"');
    const workspaceIndex = source.indexOf('class="outbound-network-page__workspace"');

    expect(scopeIndex).toBeGreaterThan(source.indexOf('<page-header'));
    expect(scopeIndex).toBeLessThan(workspaceIndex);
    expect(source).toContain('grid-template-columns: minmax(0, 8fr) minmax(19rem, 4fr)');
  });
});
