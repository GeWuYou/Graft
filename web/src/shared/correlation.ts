import { localizedApiErrorMessage } from '@/modules/shared/localized-api-error';
import type { ApiRequestError } from '@/types/axios';
import { readGlobalLoggerContext } from '@/utils/logger';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export type CorrelationSnapshot = {
  requestId: string;
  traceId: string;
};

function normalizeText(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function readLatestCorrelation(): CorrelationSnapshot {
  const context = readGlobalLoggerContext();
  const requestId = normalizeText(context.requestId);
  const traceId = normalizeText(context.traceId);

  return {
    requestId,
    traceId,
  };
}

function correlationHintText(translate: Translate, snapshot?: Partial<CorrelationSnapshot>) {
  const correlation = snapshot ?? readLatestCorrelation();
  const requestId = normalizeText(correlation.requestId);
  const traceId = normalizeText(correlation.traceId);

  if (requestId && traceId && requestId !== traceId) {
    return translate('audit.correlation.hintRequestAndTrace', { requestId, traceId });
  }
  if (requestId) {
    return translate('audit.correlation.hintRequestOnly', { requestId });
  }
  if (traceId) {
    return translate('audit.correlation.hintTraceOnly', { traceId });
  }
  return '';
}

export function describeCorrelationId(translate: Translate, value: string) {
  return translate('audit.correlation.idLabel', { id: value });
}

export function formatHintedMessage(
  translate: Translate,
  baseMessage: string,
  snapshot?: Partial<CorrelationSnapshot>,
) {
  return formatMessageWithCorrelation(baseMessage, correlationHintText(translate, snapshot));
}

export function formatMessageWithCorrelation(baseMessage: string, correlationHint: string) {
  const trimmedBase = baseMessage.trim();
  const trimmedHint = correlationHint.trim();

  if (!trimmedHint) {
    return trimmedBase;
  }
  if (!trimmedBase) {
    return trimmedHint;
  }

  return `${trimmedBase} ${trimmedHint}`;
}

export function resolveErrorMessageWithCorrelation(
  translate: Translate,
  error: unknown,
  fallbackMessage: string,
  fallbackCorrelation?: Partial<CorrelationSnapshot>,
) {
  const baseMessage = isApiRequestError(error)
    ? localizedApiErrorMessage(translate, error.messageKey, error.message) || fallbackMessage
    : fallbackMessage;

  const correlationHint = correlationHintText(
    translate,
    isApiRequestError(error)
      ? {
          requestId: error.traceId,
          traceId: error.traceId,
        }
      : fallbackCorrelation,
  );

  return formatMessageWithCorrelation(baseMessage, correlationHint);
}

function isApiRequestError(error: unknown): error is ApiRequestError {
  return Boolean(error && typeof error === 'object' && (error as ApiRequestError).isApiRequestError === true);
}
