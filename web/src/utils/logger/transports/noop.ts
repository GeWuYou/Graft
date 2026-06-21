import type { LoggerTransport } from '@/utils/logger/types';

export const noopTransport: LoggerTransport = {
  log() {},
};
