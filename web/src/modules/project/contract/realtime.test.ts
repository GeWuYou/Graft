import { describe, expect, it } from 'vitest';

import {
  parseProjectDetailRealtimePayload,
  parseProjectListSummaryRealtimePayload,
  parseProjectLogsRealtimePayload,
} from './realtime';

describe('project realtime payload parsers', () => {
  it('parses valid project list summary payloads', () => {
    const payload = parseProjectListSummaryRealtimePayload(
      JSON.stringify({
        data: {
          topic: 'project.list.summary',
          published_at: '2026-07-06T00:00:00Z',
          items: [
            {
              project_id: 7,
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
      topic: 'project.list.summary',
      published_at: '2026-07-06T00:00:00Z',
      items: [{ project_id: 7, runtime_status: 'running' }],
    });
  });

  it('rejects invalid project list summary payloads', () => {
    expect(
      parseProjectListSummaryRealtimePayload(JSON.stringify({ data: { topic: 'project.list.summary', items: [] } })),
    ).toBeNull();
    expect(
      parseProjectListSummaryRealtimePayload(
        JSON.stringify({
          data: {
            topic: 'project.list.summary',
            published_at: '2026-07-06T00:00:00Z',
            items: [{ project_id: '7' }],
          },
        }),
      ),
    ).toBeNull();
  });

  it('parses valid project detail payloads', () => {
    const payload = parseProjectDetailRealtimePayload(
      JSON.stringify({
        data: {
          topic: 'project.detail:7',
          project_id: 7,
          published_at: '2026-07-06T00:00:00Z',
          detail: { id: 7 },
          overview: { project_id: 7 },
          services: { items: [] },
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: 'project.detail:7',
      project_id: 7,
      published_at: '2026-07-06T00:00:00Z',
    });
  });

  it('rejects invalid project detail payloads', () => {
    expect(parseProjectDetailRealtimePayload(JSON.stringify({ data: { topic: 'project.detail:7' } }))).toBeNull();
    expect(parseProjectDetailRealtimePayload('not-json')).toBeNull();
  });

  it('parses valid project log payloads', () => {
    const payload = parseProjectLogsRealtimePayload(
      JSON.stringify({
        data: {
          topic: 'project.logs:7',
          entry: {
            line: 'hello',
            stream: 'stdout',
          },
        },
      }),
    );

    expect(payload).toMatchObject({
      topic: 'project.logs:7',
      entry: {
        line: 'hello',
        stream: 'stdout',
      },
    });
  });

  it('rejects invalid project log payloads', () => {
    expect(
      parseProjectLogsRealtimePayload(JSON.stringify({ data: { topic: 'project.logs:7', entry: {} } })),
    ).toBeNull();
    expect(parseProjectLogsRealtimePayload({ data: {} })).toBeNull();
  });
});
