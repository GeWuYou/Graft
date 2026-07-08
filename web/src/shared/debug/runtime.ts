import { useDebugStore } from '@/store/modules/debug';
import { store } from '@/store/pinia';

import { DEBUG_FLAG_REGISTRY } from './registry';

type FlatDebugDetail = Record<string, unknown>;

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Object.prototype.toString.call(value) === '[object Object]';
}

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

export function formatDebugLine(flagId: string, event: string, detail: FlatDebugDetail = {}) {
  const detailSummary = Object.entries(detail)
    .map(([key, value]) => `${key}=${stringifyDebugValue(value)}`)
    .join(' ');

  return detailSummary ? `[debug:${flagId}] ${event} ${detailSummary}` : `[debug:${flagId}] ${event}`;
}

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
        'window.__GRAFT_DEBUG__.disable("project.monaco")',
        'window.__GRAFT_DEBUG__.clear()',
      ].join('\n'),
  };
}

export function isDebugFlagEnabled(flagId: string) {
  return useDebugStore(store).isEnabled(flagId);
}

export function emitDebugLog(flagId: string, event: string, detail: FlatDebugDetail = {}) {
  if (!isDebugFlagEnabled(flagId)) {
    return;
  }

  console.debug(formatDebugLine(flagId, event, detail));
}
