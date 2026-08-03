import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import OutboundNetworkPage from './index.vue';

const pageSource = readFileSync(join(process.cwd(), 'src/modules/network/pages/outbound/index.vue'), 'utf8');

const apiMocks = vi.hoisted(() => ({
  diagnoseOutboundNetwork: vi.fn(),
  getOutboundNetworkDiagnosticHistory: vi.fn(),
  getOutboundNetworkPolicy: vi.fn(),
  resetOutboundNetworkPolicy: vi.fn(),
  updateOutboundNetworkPolicy: vi.fn(),
}));

vi.mock('../../api/outbound', () => apiMocks);
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string) => key,
  }),
}));
vi.mock('@/shared/components/management', () => ({ formatCompactDateTime: (value: string) => value }));

const pageHeaderStub = defineComponent({
  setup(_, { slots }) {
    return () => h('header', slots.default?.());
  },
});

const policyResponse = () => ({
  policy: {
    config: { enabled: false, http_proxy: '', https_proxy: '', no_proxy: [] },
    source: 'default' as const,
    updated_at: null,
    updated_by_name: null,
  },
  diagnostic_targets: [{ id: 'platform-update', title_key: 'network.diagnosticTargets.platformUpdate' }],
  consumers: [],
});

function mountPage() {
  apiMocks.getOutboundNetworkPolicy.mockResolvedValue({ data: policyResponse(), etag: '"0"' });
  apiMocks.getOutboundNetworkDiagnosticHistory.mockResolvedValue({ items: [] });
  return mount(OutboundNetworkPage, {
    global: {
      stubs: { PageHeader: pageHeaderStub },
    },
  });
}

describe('outbound network settings page', () => {
  it('renders the settings surface and loads empty diagnostic history', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.attributes('data-page-type')).toBe('settings');
    expect(wrapper.text()).toContain('network.outbound.overview.kicker');
    expect(wrapper.html()).toContain('network.outbound.runtime.title');
    expect(wrapper.html()).toContain('network.outbound.routing.title');
    expect(wrapper.text()).toContain('network.outbound.diagnostics.noHistory');
    expect(apiMocks.getOutboundNetworkDiagnosticHistory).toHaveBeenCalledWith('platform-update', 5);
    expect(pageSource).not.toContain('platform-update-release');
    expect(pageSource).not.toContain('diagnosticUrl');
  });

  it('runs a diagnostic from the rendered action', async () => {
    const wrapper = mountPage();
    await flushPromises();
    apiMocks.diagnoseOutboundNetwork.mockResolvedValue({});
    apiMocks.getOutboundNetworkDiagnosticHistory.mockResolvedValue({ items: [] });

    await wrapper.find('.outbound-network-page__diagnostic-action').trigger('click');
    await flushPromises();

    expect(apiMocks.diagnoseOutboundNetwork).toHaveBeenCalledWith('platform-update');
  });

  it('keeps a visible reload action for conditional request failures', () => {
    expect(pageSource).toContain('status === 412');
    expect(pageSource).toContain('status === 428');
    expect(pageSource).toContain('reloadLatestPolicy');
    expect(pageSource).toContain('network.outbound.precondition.message');
  });
});
