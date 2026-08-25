import { inject, type InjectionKey, provide, type Ref, shallowRef } from 'vue';

import type { ScrollEdgeActionsController } from './useScrollEdgeActions';

/** 壳层共享的滚动控制注册面；最后注册的 controller 获得悬浮操作的控制权。 */
export interface ScrollEdgeActionsContext {
  readonly activeController: Readonly<Ref<ScrollEdgeActionsController | null>>;
  register(controller: ScrollEdgeActionsController): () => void;
}

/** Vue 注入键；仅用于壳层滚动控制注册面，不承载业务页面状态。 */
export const scrollEdgeActionsContextKey: InjectionKey<ScrollEdgeActionsContext> = Symbol('scroll-edge-actions');

/**
 * 创建壳层滚动控制注册面；注册顺序决定当前悬浮操作所绑定的滚动视口。
 */
export function createScrollEdgeActionsContext(): ScrollEdgeActionsContext {
  const registrations: Array<{ controller: ScrollEdgeActionsController; token: symbol }> = [];
  const activeController = shallowRef<ScrollEdgeActionsController | null>(null);

  const refreshActiveController = () => {
    activeController.value = registrations[registrations.length - 1]?.controller ?? null;
  };

  const register = (controller: ScrollEdgeActionsController) => {
    const token = Symbol('scroll-edge-actions-registration');
    registrations.push({ controller, token });
    refreshActiveController();

    let active = true;
    return () => {
      if (!active) return;
      active = false;
      const index = registrations.findIndex((registration) => registration.token === token);
      if (index < 0) return;
      registrations.splice(index, 1);
      refreshActiveController();
    };
  };

  return {
    activeController,
    register,
  };
}

/**
 * 在应用壳层提供滚动控制注册面，页面可通过同一接口覆盖默认 PageContainer 目标。
 */
export function provideScrollEdgeActionsContext(): ScrollEdgeActionsContext {
  const context = createScrollEdgeActionsContext();
  provide(scrollEdgeActionsContextKey, context);
  return context;
}

/** 获取最近的壳层滚动控制注册面；未在应用壳下使用时返回 undefined。 */
export function useScrollEdgeActionsContext(): ScrollEdgeActionsContext | undefined {
  return inject(scrollEdgeActionsContextKey, undefined);
}
