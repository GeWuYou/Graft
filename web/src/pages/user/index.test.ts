import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '@/contracts/rbac/permissions';

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

vi.mock('@/api/user', () => ({
  getUsers: userApiMocks.getUsers,
}));

vi.mock('@/api/rbac', () => ({
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
      value: 'en_US',
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

function findButtonByText(wrapper: ReturnType<typeof mountUserPage>, text: string) {
  return wrapper.findAll('button').find((button) => button.text().trim() === text);
}

describe('UsersIndex role dialog', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ, RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN];
    userApiMocks.getUsers.mockReset();
    rbacApiMocks.assignUserRoles.mockReset();
    rbacApiMocks.getRoles.mockReset();
    rbacApiMocks.getUserRoleBindings.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps replace-write blocked when the current user-role snapshot cannot be restored', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockRejectedValue(new Error('selection load failed'));

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getUserRoleBindings).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('selection load failed');

    const submitButton = findButtonByText(wrapper, 'pages.userList.roleDialog.confirm');
    expect(submitButton).toBeDefined();
    expect(submitButton!.attributes('disabled')).toBeDefined();

    await submitButton!.trigger('click');
    expect(rbacApiMocks.assignUserRoles).not.toHaveBeenCalled();
  });

  it('retries the dialog load in place after role definitions fail to load', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles
      .mockRejectedValueOnce(new Error('role load failed'))
      .mockResolvedValueOnce(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [2],
    });

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('role load failed');

    const retryButton = findButtonByText(wrapper, 'pages.userList.roleDialog.retry');
    expect(retryButton).toBeDefined();

    await retryButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(2);
    expect(rbacApiMocks.getUserRoleBindings).toHaveBeenCalledWith(7);
    expect(wrapper.text()).not.toContain('role load failed');

    const checkboxGroup = wrapper.get('[data-testid="role-checkbox-group"]');
    expect(checkboxGroup.attributes('data-disabled')).toBe('false');
    expect(checkboxGroup.attributes('data-selected-role-ids')).toBe('[2]');
  });

  it('keeps the role checkbox group disabled when the actor lacks assign permission', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [2],
    });

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const checkboxGroup = wrapper.get('[data-testid="role-checkbox-group"]');
    expect(checkboxGroup.attributes('data-disabled')).toBe('true');

    const submitButton = findButtonByText(wrapper, 'pages.userList.roleDialog.confirm');
    expect(submitButton).toBeDefined();
    expect(submitButton!.attributes('disabled')).toBeDefined();
  });

  it('submits the restored role snapshot for the selected user and closes the dialog on success', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [2],
    });
    rbacApiMocks.assignUserRoles.mockResolvedValue(null);

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const submitButton = findButtonByText(wrapper, 'pages.userList.roleDialog.confirm');
    expect(submitButton).toBeDefined();
    expect(submitButton!.attributes('disabled')).toBeUndefined();

    await submitButton!.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.assignUserRoles).toHaveBeenCalledWith(7, {
      role_ids: [2],
    });
    expect(messageMocks.success).toHaveBeenCalledWith('pages.userList.assignSuccess');
    expect(wrapper.find('[data-testid="user-role-dialog"]').exists()).toBe(false);
  });

  it('shows an error toast and keeps the dialog open when the role assignment submit fails', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [2],
    });
    rbacApiMocks.assignUserRoles.mockRejectedValue(new Error('assign failed'));

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const submitButton = findButtonByText(wrapper, 'pages.userList.roleDialog.confirm');
    expect(submitButton).toBeDefined();

    await submitButton!.trigger('click');
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('assign failed');
    expect(wrapper.find('[data-testid="user-role-dialog"]').exists()).toBe(true);
  });

  it('ignores stale user-role responses after the dialog is closed and reopened', async () => {
    const firstRoleBindingRequest = createDeferred<{ role_ids: number[] }>();

    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getUserRoleBindings
      .mockImplementationOnce(() => firstRoleBindingRequest.promise)
      .mockResolvedValueOnce({
        role_ids: [],
      });

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const cancelButton = findButtonByText(wrapper, 'pages.roleList.form.cancel');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    await flushPromises();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const checkboxGroup = wrapper.get('[data-testid="role-checkbox-group"]');
    expect(checkboxGroup.attributes('data-selected-role-ids')).toBe('[]');

    firstRoleBindingRequest.resolve({
      role_ids: [2],
    });
    await flushPromises();

    expect(wrapper.get('[data-testid="role-checkbox-group"]').attributes('data-selected-role-ids')).toBe('[]');
    expect(rbacApiMocks.getUserRoleBindings).toHaveBeenCalledTimes(2);
  });

  it('ignores stale role-definition responses after the dialog is closed and reopened', async () => {
    const firstRoleListRequest = createDeferred<ReturnType<typeof createRoleListResponse>>();

    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    rbacApiMocks.getRoles
      .mockImplementationOnce(() => firstRoleListRequest.promise)
      .mockResolvedValueOnce({
        items: [],
      });
    rbacApiMocks.getUserRoleBindings.mockResolvedValue({
      role_ids: [],
    });

    const wrapper = mountUserPage();
    await flushPromises();

    const openDialogButton = findButtonByText(wrapper, 'pages.userList.assignRoles');
    expect(openDialogButton).toBeDefined();

    await openDialogButton!.trigger('click');
    await flushPromises();

    const cancelButton = findButtonByText(wrapper, 'pages.roleList.form.cancel');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    await flushPromises();

    await openDialogButton!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('pages.userList.roleDialog.empty');
    expect(rbacApiMocks.getUserRoleBindings).toHaveBeenCalledTimes(1);

    firstRoleListRequest.resolve(createRoleListResponse());
    await flushPromises();

    expect(wrapper.text()).toContain('pages.userList.roleDialog.empty');
    expect(wrapper.text()).not.toContain('Editor');
    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(2);
  });
});
