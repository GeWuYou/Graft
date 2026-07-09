import { defineStore } from 'pinia';

import { STORAGE_KEY } from '@/contracts/storage/keys';
import { DEBUG_FLAG_MAP, DEBUG_FLAG_REGISTRY, type DebugFlagId } from '@/shared/debug/registry';

type DebugRuntimeOverrideMap = Partial<Record<DebugFlagId, boolean>>;
type DebugFlagStateMap = Record<DebugFlagId, boolean>;

/**
 * 判断值是否表示启用状态。
 *
 * @param value - 待判断的值
 * @returns 当值为 `true`、`'true'`、`'1'` 或 `1` 时为 `true`，否则为 `false`。
 */
function isEnabledValue(value: unknown) {
  return value === true || value === 'true' || value === '1' || value === 1;
}

/**
 * 构建所有调试旗标的默认状态映射。
 *
 * @returns 每个已注册旗标对应其默认启用状态的完整映射
 */
function createDefaultFlags() {
  return DEBUG_FLAG_REGISTRY.reduce((accumulator: DebugFlagStateMap, definition) => {
    accumulator[definition.flagId] = definition.defaultEnabled;
    return accumulator;
  }, {} as DebugFlagStateMap);
}

/**
 * 解析环境变量中的调试标志值。
 *
 * @param envKeys - 按顺序检查的环境变量键
 * @returns 找到可用环境变量时返回其布尔值；未提供键或未找到可用值时返回 `null`
 */
function readEnvFlag(envKeys?: readonly (keyof ImportMetaEnv)[]) {
  if (!envKeys || envKeys.length === 0) {
    return null;
  }

  for (const envKey of envKeys) {
    const rawValue = import.meta.env[envKey];
    if (rawValue === undefined) {
      continue;
    }

    return isEnabledValue(rawValue);
  }

  return null;
}

/**
 * 解析指定调试旗标的最终启用状态。
 *
 * @param flagId - 要解析的旗标
 * @param runtimeOverrides - 运行时覆盖值映射
 * @param visited - 已访问旗标集合，用于防止循环依赖
 * @returns 该旗标的最终启用状态；在未知旗标或检测到循环依赖时返回 `false`
 */
function resolveFlagEnabled(
  flagId: DebugFlagId,
  runtimeOverrides: DebugRuntimeOverrideMap,
  visited = new Set<DebugFlagId>(),
): boolean {
  if (visited.has(flagId)) {
    return false;
  }

  visited.add(flagId);
  const definition = DEBUG_FLAG_MAP.get(flagId);
  if (!definition) {
    return false;
  }

  const runtimeValue = runtimeOverrides[flagId];
  if (typeof runtimeValue === 'boolean') {
    return runtimeValue;
  }

  const envValue = readEnvFlag(definition.envKeys);
  if (typeof envValue === 'boolean') {
    return envValue;
  }

  if (definition.parentFlagId) {
    return resolveFlagEnabled(definition.parentFlagId, runtimeOverrides, visited);
  }

  return definition.defaultEnabled;
}

/**
 * 计算所有调试旗标的最终启用状态。
 *
 * @param runtimeOverrides - 运行时覆盖值映射
 * @returns 所有已知调试旗标的布尔状态映射
 */
function computeFlags(runtimeOverrides: DebugRuntimeOverrideMap): DebugFlagStateMap {
  return DEBUG_FLAG_REGISTRY.reduce((accumulator: DebugFlagStateMap, definition) => {
    accumulator[definition.flagId] = resolveFlagEnabled(definition.flagId, runtimeOverrides);
    return accumulator;
  }, createDefaultFlags());
}

export const useDebugStore = defineStore('debug-runtime', {
  state: () => ({
    enabled: false,
    dangerousCapabilitiesAvailable: import.meta.env.DEV,
    runtimeOverrides: {} as DebugRuntimeOverrideMap,
    flags: createDefaultFlags(),
  }),
  getters: {
    effectiveFlags(state): DebugFlagStateMap {
      return state.flags;
    },
  },
  actions: {
    hydrateFromPersistence() {
      if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
        this.recompute();
        return;
      }

      try {
        const raw = window.localStorage.getItem(STORAGE_KEY.DEBUG_FLAGS);
        if (!raw) {
          this.recompute();
          return;
        }

        const parsed = JSON.parse(raw) as unknown;
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          this.recompute();
          return;
        }

        const overrides: DebugRuntimeOverrideMap = {};
        Object.entries(parsed as Record<string, unknown>).forEach(([flagId, value]) => {
          if (!DEBUG_FLAG_MAP.has(flagId as DebugFlagId) || typeof value !== 'boolean') {
            return;
          }

          overrides[flagId as DebugFlagId] = value;
        });
        this.runtimeOverrides = overrides;
      } catch {
        this.runtimeOverrides = {};
      }

      this.recompute();
    },
    persistRuntimeOverrides() {
      if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
        return;
      }

      try {
        if (Object.keys(this.runtimeOverrides).length === 0) {
          window.localStorage.removeItem(STORAGE_KEY.DEBUG_FLAGS);
          return;
        }

        window.localStorage.setItem(STORAGE_KEY.DEBUG_FLAGS, JSON.stringify(this.runtimeOverrides));
      } catch {
        // 调试运行时持久化失败时静默降级，不影响主业务链路。
      }
    },
    recompute() {
      this.flags = computeFlags(this.runtimeOverrides);
      this.enabled = Object.values(this.flags).some(Boolean);
    },
    setRuntimeFlag(flagId: string, value: boolean) {
      if (!DEBUG_FLAG_MAP.has(flagId as DebugFlagId)) {
        return false;
      }

      this.runtimeOverrides = {
        ...this.runtimeOverrides,
        [flagId]: value,
      };
      this.persistRuntimeOverrides();
      this.recompute();
      return this.flags[flagId as DebugFlagId];
    },
    clearRuntimeFlag(flagId?: string) {
      if (!flagId) {
        this.runtimeOverrides = {};
      } else if (DEBUG_FLAG_MAP.has(flagId as DebugFlagId)) {
        const nextOverrides = { ...this.runtimeOverrides };
        delete nextOverrides[flagId as DebugFlagId];
        this.runtimeOverrides = nextOverrides;
      }

      this.persistRuntimeOverrides();
      this.recompute();
      return this.flags;
    },
    isEnabled(flagId: string) {
      if (!DEBUG_FLAG_MAP.has(flagId as DebugFlagId)) {
        return false;
      }

      return this.flags[flagId as DebugFlagId];
    },
  },
});
