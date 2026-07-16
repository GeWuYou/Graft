import { describe, expect, it } from 'vitest';

import {
  parseApplicationLifecycleConfigRealtimePayload,
  parseApplicationListSummaryRealtimePayload,
  parseApplicationLogsRealtimePayload,
  parseApplicationRuntimeRealtimePayload,
} from './realtime';

const APPLICATION_ID = 'app_01ARZ3NDEKTSV4RRFFQ69G5FAV';

describe('project realtime payload parsers', () => {
  it('parses valid project list summary payloads', () => {
    const payload = parseApplicationListSummaryRealtimePayload(
      JSON.stringify({
        data: {
          topic: 'application.list.summary',
          published_at: '2026-07-06T00:00:00Z',
          items: [
            {
              application_id: APPLICATION_ID,
              runtime_status: 'running',
              service_count: 3,
              container_counts: {
                issue: 0,
                running: 3,
                stopped: 0,
                total: 3,
                transitioning: 0,
              },
              drift_status: 'clean',
            },
          ],
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: 'application.list.summary',
      published_at: '2026-07-06T00:00:00Z',
      items: [{ application_id: APPLICATION_ID, runtime_status: 'running' }],
    });
  });

  it('rejects invalid project list summary payloads', () => {
    expect(
      parseApplicationListSummaryRealtimePayload(
        JSON.stringify({ data: { topic: 'application.list.summary', items: [] } }),
      ),
    ).toBeNull();
    expect(
      parseApplicationListSummaryRealtimePayload(
        JSON.stringify({ data: { topic: 'project.list.other', published_at: '2026-07-06T00:00:00Z', items: [] } }),
      ),
    ).toBeNull();
    expect(
      parseApplicationListSummaryRealtimePayload(
        JSON.stringify({
          data: {
            topic: 'application.list.summary',
            published_at: '2026-07-06T00:00:00Z',
            items: [{ application_id: '7' }],
          },
        }),
      ),
    ).toBeNull();
  });

  it('parses valid project runtime payloads', () => {
    const payload = parseApplicationRuntimeRealtimePayload(
      JSON.stringify({
        data: {
          topic: `application.runtime:${APPLICATION_ID}`,
          application_id: APPLICATION_ID,
          published_at: '2026-07-06T00:00:00Z',
          detail: { id: 7 },
          overview: { application_id: 'app_7' },
          services: { items: [] },
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: `application.runtime:${APPLICATION_ID}`,
      application_id: APPLICATION_ID,
      published_at: '2026-07-06T00:00:00Z',
    });
  });

  it('rejects invalid project runtime payloads', () => {
    expect(
      parseApplicationRuntimeRealtimePayload(
        JSON.stringify({ data: { topic: `application.runtime:${APPLICATION_ID}` } }),
      ),
    ).toBeNull();
    expect(parseApplicationRuntimeRealtimePayload('not-json')).toBeNull();
    expect(
      parseApplicationRuntimeRealtimePayload(
        JSON.stringify({
          data: {
            topic: 'application.runtime:app_01ARZ3NDEKTSV4RRFFQ69G5FAW',
            application_id: 'app_7',
            published_at: '2026-07-06T00:00:00Z',
            detail: { id: 7 },
            overview: { application_id: 'app_7' },
            services: { items: [] },
          },
        }),
      ),
    ).toBeNull();
  });

  it('parses valid lifecycle configuration payloads', () => {
    const payload = parseApplicationLifecycleConfigRealtimePayload(
      JSON.stringify({
        data: {
          topic: `application.lifecycle-config:${APPLICATION_ID}`,
          application_id: APPLICATION_ID,
          published_at: '2026-07-06T00:00:00Z',
          detail: { id: 7, lifecycle_configuration: { wait_after_up: true } },
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: `application.lifecycle-config:${APPLICATION_ID}`,
      application_id: APPLICATION_ID,
      published_at: '2026-07-06T00:00:00Z',
    });
  });

  it('rejects invalid lifecycle configuration payloads', () => {
    expect(
      parseApplicationLifecycleConfigRealtimePayload(
        JSON.stringify({ data: { topic: `application.lifecycle-config:${APPLICATION_ID}` } }),
      ),
    ).toBeNull();
    expect(parseApplicationLifecycleConfigRealtimePayload('not-json')).toBeNull();
    expect(
      parseApplicationLifecycleConfigRealtimePayload(
        JSON.stringify({
          data: {
            topic: 'application.lifecycle-config:app_01ARZ3NDEKTSV4RRFFQ69G5FAW',
            application_id: APPLICATION_ID,
            published_at: '2026-07-06T00:00:00Z',
            detail: { id: 7 },
          },
        }),
      ),
    ).toBeNull();
  });

  it('parses valid project log payloads', () => {
    const payload = parseApplicationLogsRealtimePayload(
      JSON.stringify({
        data: {
          topic: `application.logs:${APPLICATION_ID}`,
          application_id: APPLICATION_ID,
          entry: {
            line: 'hello',
            stream: 'stdout',
          },
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: `application.logs:${APPLICATION_ID}`,
      entry: {
        line: 'hello',
        stream: 'stdout',
      },
    });
  });

  it('rejects invalid project log payloads', () => {
    expect(
      parseApplicationLogsRealtimePayload(
        JSON.stringify({
          data: { topic: `application.logs:${APPLICATION_ID}`, application_id: APPLICATION_ID, entry: {} },
        }),
      ),
    ).toBeNull();
    expect(parseApplicationLogsRealtimePayload({ data: {} })).toBeNull();
    expect(
      parseApplicationLogsRealtimePayload(
        JSON.stringify({
          data: {
            topic: `application.runtime:${APPLICATION_ID}`,
            application_id: APPLICATION_ID,
            entry: { line: 'hello' },
          },
        }),
      ),
    ).toBeNull();
    expect(
      parseApplicationLogsRealtimePayload(
        JSON.stringify({
          data: {
            topic: `application.logs:${APPLICATION_ID}`,
            application_id: 'app_01ARZ3NDEKTSV4RRFFQ69G5FAW',
            entry: { line: 'hello' },
          },
        }),
      ),
    ).toBeNull();
  });
});
