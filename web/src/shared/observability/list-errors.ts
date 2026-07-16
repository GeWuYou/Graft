import { MessagePlugin } from 'tdesign-vue-next/es/message';
function showLogDetailLoadError(config: {
  error: unknown;
  fallbackMessage: string;
  resolveMessage: (error: unknown, fallback: string) => string;
}) {
  MessagePlugin.error(config.resolveMessage(config.error, config.fallbackMessage));
}

export function createLogDetailErrorReporter(config: {
  fallbackMessage: () => string;
  resolveMessage: (error: unknown, fallback: string) => string;
}) {
  return (error: unknown) =>
    showLogDetailLoadError({
      error,
      fallbackMessage: config.fallbackMessage(),
      resolveMessage: config.resolveMessage,
    });
}
