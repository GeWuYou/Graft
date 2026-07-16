import { nextTick, readonly, ref } from 'vue';

export const ROUTE_LOADING_MIN_MS = 150;
export const ROUTE_LOADING_MAX_MS = 5000;

const loading = ref(false);
let loadingStartedAt = 0;
let loadingToken = 0;
let minTimer: ReturnType<typeof setTimeout> | undefined;
let maxTimer: ReturnType<typeof setTimeout> | undefined;

export const routeLoading = readonly(loading);

/**
 * 清理所有待处理的路由加载计时器。
 */
function clearTimers() {
  if (minTimer) {
    clearTimeout(minTimer);
    minTimer = undefined;
  }

  if (maxTimer) {
    clearTimeout(maxTimer);
    maxTimer = undefined;
  }
}

/**
 * 等待下一帧，以确保路由页面完成一次可见渲染。
 *
 * @returns 下一帧解析；没有 `requestAnimationFrame` 时退化到下一个宏任务。
 */
function requestNextFrame() {
  return new Promise<void>((resolve) => {
    if (typeof requestAnimationFrame === 'function') {
      requestAnimationFrame(() => resolve());
      return;
    }

    setTimeout(resolve, 0);
  });
}

/**
 * 立即停止路由加载指示器。
 */
function stopRouteLoadingNow() {
  clearTimers();
  loading.value = false;
}

/**
 * 启动路由加载指示器，并在达到最大时长后自动停止。
 */
export function startRouteLoading() {
  loadingToken += 1;
  loadingStartedAt = Date.now();
  clearTimers();
  loading.value = true;
  maxTimer = setTimeout(stopRouteLoadingNow, ROUTE_LOADING_MAX_MS);
}

/**
 * 在下一次渲染完成后结束路由加载，同时保持最小展示时长。
 */
export async function finishRouteLoadingAfterRender() {
  const token = loadingToken;
  await nextTick();
  await requestNextFrame();

  if (token !== loadingToken) {
    return;
  }

  const remaining = ROUTE_LOADING_MIN_MS - (Date.now() - loadingStartedAt);
  if (remaining <= 0) {
    stopRouteLoadingNow();
    return;
  }

  minTimer = setTimeout(stopRouteLoadingNow, remaining);
}

/**
 * 停止路由加载指示器并取消待处理的完成操作。
 */
export function hideRouteLoading() {
  loadingToken += 1;
  stopRouteLoadingNow();
}
