import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';

import UserPage from './index.vue';

const userApiMocks = vi.hoisted(() => ({
  getUsers: vi.fn(),
}));

const rbacApiMocks = vi.hoisted(() => ({
  assignUserRoles: vi.fn(),
  getRoles: vi.fn(),
  getUserRoleBindings: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({
  grantedCodes: [] as string[],
}));

vi.mock('@/modules/user/api/users', () => ({
  getUsers: userApiMocks.getUsers,
}));

vi.mock('@/modules/user/api/user-roles', () => ({
  assignUserRoles: rbacApiMocks.assignUserRoles,
  getRoles: rbacApiMocks.getRoles,
  getUserRoleBindings: rbacApiMocks.getUserRoleBindings,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasAnyPermission: (codes: string[]) => codes.some((code) => permissionState.grantedCodes.includes(code)),
    hasPermission: (code: string) => permissionState.grantedCodes.includes(code),
  }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: {
      value: 'en-US',
    },
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageMocks.error,
    success: messageMocks.success,
  },
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.title, props.description, slots.default?.()]);
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    columns: {
      type: Array,
      default: () => [],
    },
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => {
      if (props.data.length === 0) {
        return h('div', slots.empty?.());
      }

      const hasOperationColumn = props.columns.some(
        (column) =>
          typeof column === 'object' && column !== null && 'colKey' in column && column.colKey === 'operation',
      );

      return h(
        'div',
        props.data.map((row, index) =>
          h('div', { 'data-testid': `user-row-${index}` }, [
            h('span', String((row as { username?: string }).username ?? '')),
            hasOperationColumn ? slots.operation?.({ row }) : null,
          ]),
        ),
      );
    };
  },
});

const dialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () =>
      props.visible
        ? h('section', { 'data-testid': 'user-role-dialog' }, [
            h('h2', props.header),
            slots.body?.(),
            slots.default?.(),
          ])
        : null;
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          'data-loading': String(props.loading),
          onClick: (event: MouseEvent) => {
            if (!props.disabled) {
              emit('click', event);
            }
          },
        },
        slots.default?.(),
      );
  },
});

const checkboxGroupStub = defineComponent({
  name: 'TCheckboxGroupStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    modelValue: {
      type: Array<number>,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'role-checkbox-group',
          'data-disabled': String(props.disabled),
          'data-selected-role-ids': JSON.stringify(props.modelValue),
        },
        slots.default?.(),
      );
  },
});

const checkboxStub = defineComponent({
  name: 'TCheckboxStub',
  props: {
    value: {
      type: Number,
      required: true,
    },
  },
  setup(props, { slots }) {
    return () => h('label', { 'data-role-id': String(props.value) }, slots.default?.());
  },
});

function createUserListResponse() {
  return {
    items: [
      {
        id: 7,
        username: 'alice',
        display: 'Alice',
        created_at: '2026-05-17T00:00:00Z',
        updated_at: '2026-05-17T00:00:00Z',
      },
    ],
  };
}

function createRoleListResponse() {
  return {
    items: [
      {
        id: 2,
        name: 'editor',
        display: 'Editor',
        description: 'Editor role',
        builtin: false,
      },
    ],
  };
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;

  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

function mountUserPage() {
  return mount(UserPage, {
    global: {
      directives: {
        permission: {
          mounted() {},
        },
      },
      stubs: {
        't-button': buttonStub,
        't-card': passthroughStub,
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-col': passthroughStub,
        't-dialog': dialogStub,
        't-empty': passthroughStub,
        't-row': passthroughStub,
        't-table': tableStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('UserPage', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [];
    userApiMocks.getUsers.mockReset();
    rbacApiMocks.assignUserRoles.mockReset();
    rbacApiMocks.getRoles.mockReset();
    rbacApiMocks.getUserRoleBindings.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads users on mount and hides operation controls without permissions', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();

    expect(userApiMocks.getUsers).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('alice');
    expect(wrapper.text()).not.toContain('pages.userList.assignRoles');
  });

  it('opens the role dialog and applies role bindings when permitted', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ, RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({ role_ids: [2] });
    rbacApiMocks.assignUserRoles.mockResolvedValue(null);

    const wrapper = mountUserPage();
    await flushPromises();

    const manageRolesButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('pages.userList.assignRoles'));
    expect(manageRolesButton).toBeDefined();
    await manageRolesButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getUserRoleBindings).toHaveBeenCalledWith(7);
    expect(wrapper.find('[data-testid="user-role-dialog"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="role-checkbox-group"]').attributes('data-selected-role-ids')).toBe('[2]');

    const dialogButtons = wrapper.findAll('[data-testid="user-role-dialog"] button');
    await dialogButtons[1].trigger('click');
    await flushPromises();

    expect(rbacApiMocks.assignUserRoles).toHaveBeenCalledWith(7, { role_ids: [2] });
    expect(messageMocks.success).toHaveBeenCalledWith('pages.userList.assignSuccess');
  });

  it('ignores stale role dialog requests after the dialog is closed', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const rolesDeferred = createDeferred<ReturnType<typeof createRoleListResponse>>();
    const bindingsDeferred = createDeferred<{ role_ids: number[] }>();
    rbacApiMocks.getRoles.mockReturnValue(rolesDeferred.promise);
    rbacApiMocks.getUserRoleBindings.mockReturnValue(bindingsDeferred.promise);

    const wrapper = mountUserPage();
    await flushPromises();

    const manageRolesButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('pages.userList.assignRoles'));
    expect(manageRolesButton).toBeDefined();
    await manageRolesButton!.trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="user-role-dialog"]').exists()).toBe(true);

    const dialogButtons = wrapper.findAll('[data-testid="user-role-dialog"] button');
    await dialogButtons[0].trigger('click');
    await flushPromises();

    rolesDeferred.resolve(createRoleListResponse());
    bindingsDeferred.resolve({ role_ids: [2] });
    await flushPromises();

    expect(wrapper.find('[data-testid="user-role-dialog"]').exists()).toBe(false);
  });
});
