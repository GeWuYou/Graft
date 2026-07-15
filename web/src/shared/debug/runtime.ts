import { useDebugStore } from '@/store/modules/debug';
import { store } from '@/store/pinia';
import { createLogger } from '@/utils/logger';

import { DEBUG_FLAG_REGISTRY } from './registry';

type FlatDebugDetail = Record<string, unknown>;

const logger = createLogger('debug.runtime');

/**
 * 判断值是否为普通对象。
 *
 * @param value - 要检查的值
 * @returns `true` if `value` is a plain object, `false` otherwise.
 */
function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Object.prototype.toString.call(value) === '[object Object]';
}

/**
 * 将调试值转换为紧凑的字符串表示。
 *
 * @returns 调试值对应的字符串表示。
 */
function stringifyDebugValue(value: unknown): string {
  if (value === null) {
    return 'null';
  }

  if (value === undefined) {
    return 'undefined';
  }

  if (value instanceof Error) {
    return `${value.name}:${value.message}`;
  }

  if (typeof value === 'string') {
    return value;
  }

  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }

  if (Array.isArray(value)) {
    return value.map((item) => stringifyDebugValue(item)).join(',');
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (isPlainObject(value)) {
    return Object.entries(value)
      .slice(0, 6)
      .map(([key, item]) => `${key}:${stringifyDebugValue(item)}`)
      .join(',');
  }

  return String(value);
}

/**
 * 格式化一条调试日志行。
 *
 * @param flagId - 调试标志标识
 * @param event - 事件名称
 * @param detail - 需要附加到日志中的结构化信息
 * @returns 格式化后的调试日志字符串
 */
export function formatDebugLine(flagId: string, event: string, detail: FlatDebugDetail = {}) {
  const detailSummary = Object.entries(detail)
    .map(([key, value]) => `${key}=${stringifyDebugValue(value)}`)
    .join(' ');

  return detailSummary ? `[debug:${flagId}] ${event} ${detailSummary}` : `[debug:${flagId}] ${event}`;
}

/**
 * 初始化全局调试运行时接口。
 *
 * 在浏览器环境下将调试控制对象挂载到 `window.__GRAFT_DEBUG__`，提供调试状态查询、
 * 旗标列表管理以及运行时覆盖控制；同时完成调试状态持久化恢复。
 */
export function initDebugRuntime() {
  const debugStore = useDebugStore(store);
  debugStore.hydrateFromPersistence();

  if (typeof window === 'undefined') {
    return;
  }

  window.__GRAFT_DEBUG__ = {
    state: () => ({
      dangerousCapabilitiesAvailable: debugStore.dangerousCapabilitiesAvailable,
      enabled: debugStore.enabled,
      flags: { ...debugStore.effectiveFlags },
      runtimeOverrides: Object.fromEntries(
        Object.entries(debugStore.runtimeOverrides).filter(
          (entry): entry is [string, boolean] => typeof entry[1] === 'boolean',
        ),
      ),
    }),
    list: () =>
      DEBUG_FLAG_REGISTRY.map((definition) => ({
        ...definition,
        effectiveEnabled: debugStore.isEnabled(definition.flagId),
        runtimeOverride:
          typeof debugStore.runtimeOverrides[definition.flagId] === 'boolean'
            ? Boolean(debugStore.runtimeOverrides[definition.flagId])
            : null,
      })),
    enable: (flagId: string) => debugStore.setRuntimeFlag(flagId, true),
    disable: (flagId: string) => debugStore.setRuntimeFlag(flagId, false),
    set: (flagId: string, value: boolean) => debugStore.setRuntimeFlag(flagId, value),
    clear: (flagId?: string) => debugStore.clearRuntimeFlag(flagId),
    isEnabled: (flagId: string) => debugStore.isEnabled(flagId),
    help: () =>
      [
        'window.__GRAFT_DEBUG__.list()',
        'window.__GRAFT_DEBUG__.state()',
        'window.__GRAFT_DEBUG__.enable("tabs")',
        'window.__GRAFT_DEBUG__.enable("project.logs")',
        'window.__GRAFT_DEBUG__.enable("project.workspace")',
        'window.__GRAFT_DEBUG__.disable("project.monaco")',
        'window.__GRAFT_DEBUG__.clear()',
      ].join('\n'),
  };

  emitDebugLog('project.workspace', 'runtime-ready', {
    enabled: debugStore.isEnabled('project.workspace'),
    runtimeOverride: debugStore.runtimeOverrides['project.workspace'] ?? 'none',
  });
}

/**
 * 判断指定调试标志是否已启用。
 *
 * @param flagId - 调试标志 ID
 * @returns `true` 如果该标志已启用，`false` 否则。
 */
export function isDebugFlagEnabled(flagId: string) {
  return useDebugStore(store).isEnabled(flagId);
}

/**
 * 在调试标志启用时输出调试日志。
 *
 * @param flagId - 调试标志 ID
 * @param event - 调试事件名称
 * @param detail - 要附加到日志中的结构化信息
 */
export function emitDebugLog(flagId: string, event: string, detail: FlatDebugDetail = {}) {
  if (!isDebugFlagEnabled(flagId)) {
    return;
  }

  logger.debug(formatDebugLine(flagId, event, detail));
}
