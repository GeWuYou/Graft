import { tinykeys } from 'tinykeys';
import type { MaybeRefOrGetter } from 'vue';
import { onBeforeUnmount, onMounted, toValue, watch } from 'vue';

export interface UseKeyboardShortcutOptions {
  enabled?: MaybeRefOrGetter<boolean>;
  ignoreRepeat?: boolean;
  preventDefault?: boolean;
  target?: MaybeRefOrGetter<EventTarget | null>;
}

/**
 * 获取浏览器环境中的默认键盘事件目标。
 *
 * @returns 浏览器环境中的 `window`，非浏览器环境中返回 `null`
 */
function resolveDefaultTarget(): EventTarget | null {
  return typeof window === 'undefined' ? null : window;
}

/**
 * 将快捷键中的 `$mod` 展开为平台对应的修饰键组合。
 *
 * @param shortcut - 包含可选 `$mod` 修饰符的快捷键字符串
 * @returns 未包含 `$mod` 时返回原快捷键；否则返回分别使用 `Control` 和 `Meta` 的快捷键
 */
function expandPlatformModifier(shortcut: string): string[] {
  if (!shortcut.includes('$mod')) {
    return [shortcut];
  }

  return [shortcut.replaceAll('$mod', 'Control'), shortcut.replaceAll('$mod', 'Meta')];
}

/**
 * 判断是否应忽略快捷键事件。
 *
 * @param event - 要检查的键盘事件
 * @returns `true` 如果事件处于输入法组合状态，否则返回 `false`
 */
function shouldIgnoreShortcutEvent(event: KeyboardEvent): boolean {
  return event.isComposing;
}

/**
 * 在当前 Vue 组件的生命周期内绑定键盘快捷键。
 *
 * @param shortcut - 要绑定的 `tinykeys` 快捷键字符串
 * @param callback - 快捷键触发时接收键盘事件的回调
 * @param options - 控制快捷键启用状态、默认行为、重复事件及事件目标的选项
 */
export function useKeyboardShortcut(
  shortcut: string,
  callback: (event: KeyboardEvent) => void,
  options: UseKeyboardShortcutOptions = {},
) {
  const enabled = options.enabled ?? true;
  const preventDefault = options.preventDefault ?? false;
  const ignoreRepeat = options.ignoreRepeat ?? false;
  const resolveTarget = () => (options.target === undefined ? resolveDefaultTarget() : toValue(options.target));

  let unsubscribe: (() => void) | undefined;
  let stopWatch: (() => void) | undefined;

  const unbind = () => {
    unsubscribe?.();
    unsubscribe = undefined;
  };

  const bind = () => {
    unbind();

    const target = resolveTarget();
    if (!target || !toValue(enabled)) {
      return;
    }

    const handler = (event: KeyboardEvent) => {
      if (!toValue(enabled)) {
        return;
      }

      if (preventDefault) {
        event.preventDefault();
      }

      if (ignoreRepeat && event.repeat) {
        return;
      }

      callback(event);
    };
    const keybindings = Object.fromEntries(expandPlatformModifier(shortcut).map((key) => [key, handler]));

    unsubscribe = tinykeys(target as Window | HTMLElement, keybindings, { ignore: shouldIgnoreShortcutEvent });
  };

  onMounted(() => {
    stopWatch = watch([resolveTarget, () => toValue(enabled)], bind, { immediate: true });
  });

  onBeforeUnmount(() => {
    stopWatch?.();
    unbind();
  });
}
