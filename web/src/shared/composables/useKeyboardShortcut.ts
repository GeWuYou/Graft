import { tinykeys } from 'tinykeys';
import type { MaybeRefOrGetter } from 'vue';
import { onBeforeUnmount, onMounted, toValue, watch } from 'vue';

export interface UseKeyboardShortcutOptions {
  enabled?: MaybeRefOrGetter<boolean>;
  ignoreRepeat?: boolean;
  preventDefault?: boolean;
  target?: MaybeRefOrGetter<EventTarget | null>;
}

function resolveDefaultTarget(): EventTarget | null {
  return typeof window === 'undefined' ? null : window;
}

function expandPlatformModifier(shortcut: string): string[] {
  if (!shortcut.includes('$mod')) {
    return [shortcut];
  }

  return [shortcut.replaceAll('$mod', 'Control'), shortcut.replaceAll('$mod', 'Meta')];
}

function shouldIgnoreShortcutEvent(event: KeyboardEvent): boolean {
  return event.isComposing;
}

/**
 * Binds one tinykeys shortcut for the lifecycle of the current Vue component.
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
