import { MessagePlugin } from 'tdesign-vue-next';
import type { Ref } from 'vue';

function handleLogListLoadError(config: {
  error: unknown;
  fallbackMessage: string;
  logMessage: string;
  logger: { error: (message: string, error?: unknown) => void };
  resolveMessage: (error: unknown, fallback: string) => string;
  setListError: (message: string) => void;
  setRowsEmpty: () => void;
  setTotalEmpty: () => void;
}) {
  config.setRowsEmpty();
  config.setTotalEmpty();
  config.logger.error(config.logMessage, config.error);
  const message = config.resolveMessage(config.error, config.fallbackMessage);
  config.setListError(message);
  MessagePlugin.error(message);
}

function showLogDetailLoadError(config: {
  error: unknown;
  fallbackMessage: string;
  resolveMessage: (error: unknown, fallback: string) => string;
}) {
  MessagePlugin.error(config.resolveMessage(config.error, config.fallbackMessage));
}

export function createLogListErrorReporter<Row>(config: {
  fallbackMessage: () => string;
  listError: Ref<string>;
  logger: { error: (message: string, error?: unknown) => void };
  logMessage: string;
  resolveMessage: (error: unknown, fallback: string) => string;
  rows: Ref<Row[]>;
  total: Ref<number>;
}) {
  return (error: unknown) =>
    handleLogListLoadError({
      error,
      fallbackMessage: config.fallbackMessage(),
      logger: config.logger,
      logMessage: config.logMessage,
      resolveMessage: config.resolveMessage,
      setListError: (message) => {
        config.listError.value = message;
      },
      setRowsEmpty: () => {
        config.rows.value = [];
      },
      setTotalEmpty: () => {
        config.total.value = 0;
      },
    });
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
