import { createPinia, type Pinia, setActivePinia } from 'pinia';
import type { App as VueApp, Component } from 'vue';
import { defineComponent, h } from 'vue';
import { createMemoryHistory, createRouter, type Router } from 'vue-router';

import { setupI18n } from '@/app/i18n';

function createPassthroughComponent(name: string): Component {
  return defineComponent({
    inheritAttrs: false,
    name,
    setup(_, { attrs, slots }) {
      return () =>
        h(
          'div',
          {
            ...attrs,
            'data-stub': name,
          },
          [
            h('span', slots.icon ? slots.icon() : []),
            h('div', slots.default ? slots.default() : []),
          ],
        );
    },
  });
}

export function createTestingPinia(): Pinia {
  const pinia = createPinia();

  setActivePinia(pinia);

  return pinia;
}

export function createTestingRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/login',
        name: 'login',
        component: defineComponent({
          name: 'LoginRouteStub',
          setup: () => () => h('div', 'login'),
        }),
      },
      {
        path: '/dashboard',
        name: 'dashboard',
        component: defineComponent({
          name: 'DashboardRouteStub',
          setup: () => () => h('div', 'dashboard'),
        }),
        meta: {
          title: '仪表盘',
          titleKey: 'navigation.dashboard',
          requiresAuth: true,
        },
      },
      {
        path: '/unauthorized',
        name: 'unauthorized',
        component: defineComponent({
          name: 'UnauthorizedRouteStub',
          setup: () => () => h('div', 'unauthorized'),
        }),
      },
      {
        path: '/:pathMatch(.*)*',
        name: 'not-found',
        component: defineComponent({
          name: 'NotFoundRouteStub',
          setup: () => () => h('div', 'not-found'),
        }),
      },
    ],
  });
}

export async function pushRouter(router: Router, path: string) {
  await router.push(path);
  await router.isReady();
}

export function createI18nPlugin(pinia: Pinia) {
  return {
    install(app: VueApp<Element>) {
      setupI18n(app, pinia);
    },
  };
}

export function createTDesignStubs() {
  return {
    't-aside': createPassthroughComponent('t-aside'),
    't-avatar': createPassthroughComponent('t-avatar'),
    't-breadcrumb': createPassthroughComponent('t-breadcrumb'),
    't-breadcrumb-item': createPassthroughComponent('t-breadcrumb-item'),
    't-card': createPassthroughComponent('t-card'),
    't-content': createPassthroughComponent('t-content'),
    't-header': createPassthroughComponent('t-header'),
    't-layout': createPassthroughComponent('t-layout'),
    't-space': createPassthroughComponent('t-space'),
    't-tag': createPassthroughComponent('t-tag'),
    't-button': defineComponent({
      inheritAttrs: false,
      name: 'TButtonStub',
      emits: ['click'],
      setup(_, { attrs, emit, slots }) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              type: attrs.type === 'submit' ? 'submit' : 'button',
              onClick: (event: Event) => emit('click', event),
            },
            slots.default ? slots.default() : [],
          );
      },
    }),
    't-input': defineComponent({
      inheritAttrs: false,
      name: 'TInputStub',
      props: {
        modelValue: {
          default: '',
        },
      },
      emits: ['update:modelValue'],
      setup(props, { attrs, emit }) {
        return () =>
          h('input', {
            ...attrs,
            value: props.modelValue,
            onInput: (event: Event) =>
              emit(
                'update:modelValue',
                (event.target as HTMLInputElement).value,
              ),
          });
      },
    }),
    't-menu': defineComponent({
      inheritAttrs: false,
      name: 'TMenuStub',
      props: {
        value: {
          default: '',
        },
      },
      emits: ['change'],
      setup(_, { attrs, emit, slots }) {
        return () =>
          h(
            'div',
            {
              ...attrs,
              'data-stub': 't-menu',
              onClick: (event: Event) => {
                const target = event.target as HTMLElement | null;
                const trigger = target?.closest(
                  '[data-menu-value]',
                ) as HTMLElement | null;
                const value = trigger?.getAttribute('data-menu-value');
                if (value) {
                  emit('change', value);
                }
              },
            },
            slots.default ? slots.default() : [],
          );
      },
    }),
    't-menu-item': defineComponent({
      inheritAttrs: false,
      name: 'TMenuItemStub',
      props: {
        value: {
          default: '',
        },
      },
      emits: ['click'],
      setup(props, { attrs, emit, slots }) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              'data-menu-value': props.value,
              type: 'button',
              onClick: (event: Event) => emit('click', event),
            },
            [
              h('span', slots.icon ? slots.icon() : []),
              ...(slots.default ? slots.default() : []),
            ],
          );
      },
    }),
  };
}
