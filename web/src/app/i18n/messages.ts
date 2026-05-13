export const DEFAULT_LOCALE = 'zh-CN';
export const DEFAULT_FALLBACK_LOCALE = 'zh-CN';

interface MessageTree {
  [key: string]: string | MessageTree;
}

const messageCatalog: Record<string, MessageTree> = {
  'zh-CN': {
    app: {
      name: 'Graft',
      shellName: 'Admin Shell',
    },
    common: {
      actions: {
        login: '登录',
        logout: '退出登录',
        goToAccessiblePage: '前往可访问页面',
        switchAccount: '切换账号',
        backToDashboard: '返回仪表盘',
        backToLogin: '返回登录页',
      },
      status: {
        mvpShell: 'MVP Shell',
      },
    },
    navigation: {
      dashboard: '仪表盘',
    },
    routes: {
      login: '登录',
      unauthorized: '无权限访问',
      workspace: '工作台',
      notFound: '页面不存在',
    },
    layouts: {
      auth: {
        title: '插件式后台平台',
        description: '先提供稳定后台壳，再让业务模块沿着菜单、路由、权限和 API 的固定路径接入。',
      },
      basic: {
        permissionHint: '静态权限会在接入后端后替换',
      },
    },
    login: {
      title: '登录 Graft',
      description: '当前阶段使用静态登录态模拟后台返回的 token、用户信息和权限集合。',
      fields: {
        userName: '用户名',
        userNamePlaceholder: '请输入用户名',
        password: '密码',
        passwordPlaceholder: '请输入密码',
      },
      tips: {
        recommendedUser: '建议账号：admin',
        recommendedPassword: '建议密码：任意非空值',
      },
      messages: {
        missingCredentials: '请输入用户名和密码',
        success: '登录成功',
      },
    },
    dashboard: {
      hero: {
        eyebrow: 'Graft Platform',
        title: '欢迎回来，{userName}',
        defaultUser: '管理员',
        summary: '当前前端壳已经具备登录、静态路由、基础导航和权限占位，后续模块可以沿着 `menu + route + page + api + permission` 的路径接入。',
      },
      tags: {
        stack: 'Vue 3 + TypeScript',
        ui: 'TDesign Vue Next',
        permission: 'Static Permission Stub',
      },
      metrics: {
        routes: {
          title: '静态路由',
          note: '登录页、仪表盘和 404 已接通基础守卫。',
        },
        navigation: {
          title: '导航项',
          note: '当前只保留 MVP 所需的仪表盘入口。',
        },
        permissions: {
          title: '权限码',
          note: '通过 route meta 与 auth store 建立最小约束。',
        },
      },
      sections: {
        capabilities: {
          title: '当前壳能力',
          items: {
            layout: '登录页使用 `AuthLayout`，后台页面统一挂在 `BasicLayout`。',
            guard: '静态路由已经接入全局守卫，未登录访问会被重定向回登录页。',
            menu: '导航 store 保留 `plugin` 和 `permissionCode` 字段，为后端动态菜单留出契约位。',
          },
        },
        nextSteps: {
          title: '下一步接入建议',
          items: {
            session: '登录成功后改为拉取用户信息、权限集合和菜单树。',
            router: '将动态菜单结果装配到 `router` 与 `navigation` store。',
            modules: '在 `web/src/modules` 下按插件维度接入用户、角色和审计模块。',
          },
        },
      },
    },
    unauthorized: {
      title: '当前账号无权访问此页面',
      description: '登录态仍然有效，但目标路由要求的权限不在当前会话内。后续接入后端菜单与权限数据后，这里会继续作为显式授权兜底页。',
    },
    notFound: {
      title: '页面不存在',
      description: '当前地址没有匹配的静态路由，后续动态模块接入后也会复用同一套兜底页。',
    },
  },
};

function readMessageNode(messageTree: MessageTree | undefined, key: string): string | null {
  if (!messageTree) {
    return null;
  }

  const value = key.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') {
      return null;
    }

    return (current as Record<string, unknown>)[segment];
  }, messageTree);

  return typeof value === 'string' ? value : null;
}

export function normalizeLocale(locale: string | null | undefined): string {
  if (!locale) {
    return DEFAULT_LOCALE;
  }

  return locale in messageCatalog ? locale : DEFAULT_LOCALE;
}

export function resolveMessageTemplate(locale: string, key: string): string | null {
  return readMessageNode(messageCatalog[locale], key);
}
