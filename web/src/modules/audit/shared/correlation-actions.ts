import { Button } from 'tdesign-vue-next/es/button';
import { NotifyPlugin } from 'tdesign-vue-next/es/notification';
import { Space } from 'tdesign-vue-next/es/space';
import { h } from 'vue';
import type { Router } from 'vue-router';

import { buildAuditRequestLocation } from '@/modules/audit/contract/deep-link';
import { copyText } from '@/shared/observability';

type Translate = (key: string, params?: Record<string, unknown>) => string;

function normalizeText(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

export type CorrelationActionsOptions = {
  router?: Router;
  title: string;
  message: string;
  requestId?: string;
  translate: Translate;
};

export function requestIdFromError(error: unknown) {
  if (!error || typeof error !== 'object' || !('traceId' in error)) {
    return '';
  }

  return normalizeText((error as { traceId?: unknown }).traceId);
}

async function copyCorrelationRequestId(requestId: string) {
  try {
    return await copyText(requestId);
  } catch {
    return false;
  }
}

export function openCorrelationErrorNotification(options: CorrelationActionsOptions) {
  const requestId = normalizeText(options.requestId);

  return NotifyPlugin.error({
    title: options.title,
    duration: 0,
    closeBtn: true,
    content: () =>
      h('div', [
        h('p', options.message),
        requestId
          ? h('p', [h('strong', `${options.translate('audit.correlation.requestIdLabel')} `), h('span', requestId)])
          : null,
      ]),
    footer: requestId
      ? () =>
          h(Space, { size: 8 }, () => [
            h(
              Button,
              {
                size: 'small',
                theme: 'default',
                variant: 'outline',
                onClick: async () => {
                  const copied = await copyCorrelationRequestId(requestId);
                  if (copied) {
                    void NotifyPlugin.success({
                      title: options.translate('audit.correlation.copyRequestIdSuccessTitle'),
                      content: options.translate('audit.correlation.copyRequestIdSuccessContent'),
                      duration: 2000,
                    });
                    return;
                  }

                  void NotifyPlugin.error({
                    title: options.translate('audit.correlation.copyRequestIdErrorTitle'),
                    content: options.translate('audit.correlation.copyRequestIdErrorContent'),
                    duration: 2500,
                  });
                },
              },
              () => options.translate('audit.correlation.copyRequestIdAction'),
            ),
            options.router
              ? h(
                  Button,
                  {
                    size: 'small',
                    theme: 'primary',
                    variant: 'base',
                    onClick: () => {
                      void options.router?.push(buildAuditRequestLocation(requestId));
                    },
                  },
                  () => options.translate('audit.correlation.viewAuditAction'),
                )
              : null,
          ])
      : undefined,
  });
}
