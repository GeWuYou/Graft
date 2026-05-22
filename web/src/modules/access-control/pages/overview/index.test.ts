import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';

import OverviewPage from './index.vue';

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

const permissionStoreMocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
}));

const userApiMocks = vi.hoisted(() => ({
  getUsers: vi.fn(),
}));

const userRoleApiMocks = vi.hoisted(() => ({
  getRoles: vi.fn(),
  getUserRoleBindings: vi.fn(),
}));

const rbacApiMocks = vi.hoisted(() => ({
  getPermissions: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerMocks.push,
  }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => permissionStoreMocks,
}));

vi.mock('@/modules/user/api/users', () => ({
  getUsers: userApiMocks.getUsers,
}));

vi.mock('@/modules/user/api/user-roles', () => ({
  getRoles: userRoleApiMocks.getRoles,
  getUserRoleBindings: userRoleApiMocks.getUserRoleBindings,
}));

vi.mock('@/modules/rbac/api/rbac', () => ({
  getPermissions: rbacApiMocks.getPermissions,
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_, { slots, attrs }) {
    return () => h('div', attrs, slots.default?.());
  },
});

const pageHeaderStub = defineComponent({
  name: 'ManagementPageHeaderStub',
  setup(_, { slots }) {
    return () => h('section', [h('div', slots.eyebrow?.()), h('div', slots.actions?.()), h('div', slots.default?.())]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots, attrs }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

function mountOverview() {
  return mount(OverviewPage, {
    global: {
      stubs: {
        'management-empty-state': passthroughStub,
        'management-page-content': passthroughStub,
        'management-page-header': pageHeaderStub,
        'management-stats-grid': passthroughStub,
        'management-table-card': passthroughStub,
        't-button': buttonStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('AccessControlOverviewPage', () => {
  beforeEach(() => {
    routerMocks.push.mockReset();
    permissionStoreMocks.hasPermission.mockReset();
    userApiMocks.getUsers.mockReset();
    userRoleApiMocks.getRoles.mockReset();
    userRoleApiMocks.getUserRoleBindings.mockReset();
    rbacApiMocks.getPermissions.mockReset();

    permissionStoreMocks.hasPermission.mockImplementation(
      (code: string) => code === RBAC_PERMISSION_CODE.PERMISSION_READ,
    );
    userApiMocks.getUsers.mockResolvedValue({
      items: [{ id: 1, status: 'enabled' }],
    });
    userRoleApiMocks.getRoles.mockResolvedValue({
      items: [{ id: 1, builtin: true, permission_count: 2, updated_at: '2026-05-23T10:00:00Z' }],
    });
    userRoleApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [1],
    });
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [{ id: 1 }],
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('shows the permission quick entry and navigates to the permission page', async () => {
    const wrapper = mountOverview();
    await flushPromises();

    expect(wrapper.text()).toContain('accessControl.overview.actions.viewPermissions');
    expect(wrapper.get('[data-testid="access-control-quick-link-permissions"]').text()).toContain(
      'accessControl.overview.quickLinks.permissions.title',
    );

    await wrapper.get('[data-testid="access-control-quick-link-permissions"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: '/access-control/permissions',
    });
  });

  it('hides the permission entry when permission.read is missing', async () => {
    permissionStoreMocks.hasPermission.mockReturnValue(false);

    const wrapper = mountOverview();
    await flushPromises();

    expect(wrapper.text()).not.toContain('accessControl.overview.actions.viewPermissions');
    expect(wrapper.find('[data-testid="access-control-quick-link-permissions"]').exists()).toBe(false);
  });
});
