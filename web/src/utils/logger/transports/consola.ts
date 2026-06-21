import { createConsola } from 'consola';

import type { LogEvent, LoggerTransport } from '@/utils/logger/types';

const logger = createConsola();

function createPayload(event: LogEvent) {
  return {
    timestamp: event.timestamp,
    ...(event.meta === undefined ? {} : { meta: event.meta }),
    ...(event.error === undefined ? {} : { error: event.error }),
  };
}

export function createConsolaTransport(): LoggerTransport {
  return {
    log(event) {
      const taggedLogger = logger.withTag(event.moduleName);
      const payload = createPayload(event);

      switch (event.level) {
        case 'debug':
          taggedLogger.debug(event.message, payload);
          return;
        case 'info':
          taggedLogger.info(event.message, payload);
          return;
        case 'warn':
          taggedLogger.warn(event.message, payload);
          return;
        case 'error':
          taggedLogger.error(event.message, payload);
          return;
        case 'silent':
          return;
      }
    },
  };
}
