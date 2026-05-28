import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';
import { USER_PERMISSION_CODE } from '@/modules/user/contract/permissions';

import { ACCESS_CONTROL_ROUTE_PATH } from '../../contract/bootstrap';
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

const dashboardShellStub = defineComponent({
  name: 'GovernanceDashboardShellStub',
  setup(_, { slots }) {
    return () =>
      h('section', [
        h('div', slots.eyebrow?.()),
        h('div', slots.actions?.()),
        h('div', slots.summary?.()),
        h('div', slots.default?.()),
      ]);
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
      directives: {
        permission: {
          mounted(el, binding) {
            const value = binding.value;
            const allowed =
              typeof value === 'string'
                ? permissionStoreMocks.hasPermission(value)
                : Array.isArray(value)
                  ? value.every((code: string) => permissionStoreMocks.hasPermission(code))
                  : value && typeof value === 'object'
                    ? (Array.isArray(value.allOf) ? value.allOf : []).every((code: string) =>
                        permissionStoreMocks.hasPermission(code),
                      ) &&
                      ((Array.isArray(value.anyOf) ? value.anyOf : []).length === 0 ||
                        (Array.isArray(value.anyOf) ? value.anyOf : []).some((code: string) =>
                          permissionStoreMocks.hasPermission(code),
                        ))
                    : false;
            if (!allowed) {
              el.remove();
            }
          },
        },
      },
      stubs: {
        'management-empty-state': passthroughStub,
        'governance-dashboard-shell': dashboardShellStub,
        'governance-summary-card': passthroughStub,
        'governance-section': passthroughStub,
        'governance-action-panel': passthroughStub,
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
    await flushPromises();

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: ACCESS_CONTROL_ROUTE_PATH.PERMISSIONS,
    });
  });

  it('hides the permission entry when permission.read is missing', async () => {
    permissionStoreMocks.hasPermission.mockReturnValue(false);

    const wrapper = mountOverview();
    await flushPromises();

    expect(wrapper.text()).not.toContain('accessControl.overview.actions.viewPermissions');
    expect(wrapper.find('[data-testid="access-control-quick-link-permissions"]').exists()).toBe(false);
  });

  it('does not call guarded overview APIs when the matching read permission is missing', async () => {
    permissionStoreMocks.hasPermission.mockImplementation(() => false);

    mountOverview();
    await flushPromises();

    expect(userApiMocks.getUsers).not.toHaveBeenCalled();
    expect(userRoleApiMocks.getRoles).not.toHaveBeenCalled();
    expect(rbacApiMocks.getPermissions).not.toHaveBeenCalled();
    expect(userRoleApiMocks.getUserRoleBindings).not.toHaveBeenCalled();
  });

  it('skips user role binding requests when user role read permission is missing', async () => {
    permissionStoreMocks.hasPermission.mockImplementation(
      (code: string) =>
        code === USER_PERMISSION_CODE.READ ||
        code === RBAC_PERMISSION_CODE.ROLE_READ ||
        code === RBAC_PERMISSION_CODE.PERMISSION_READ,
    );

    mountOverview();
    await flushPromises();

    expect(userApiMocks.getUsers).toHaveBeenCalledTimes(1);
    expect(userRoleApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getPermissions).toHaveBeenCalledTimes(1);
    expect(userRoleApiMocks.getUserRoleBindings).not.toHaveBeenCalled();
  });
});
