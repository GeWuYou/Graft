import { describe, expect, it } from 'vitest';

import type { ProjectLogEntry, ProjectLogResponse } from '../types/project';
import { ProjectLogRealtimeBatcher } from './project-log-realtime-batcher';

function entry(line: string, occurredAt: string, containerID = line): ProjectLogEntry {
  return {
    container_id: containerID,
    container_name: `container-${containerID}`,
    line,
    occurred_at: occurredAt,
    service_name: 'app',
    source: {
      container_id: containerID,
      container_name: `container-${containerID}`,
      service_name: 'app',
    },
    stream: 'stdout',
  };
}

function snapshot(entries: ProjectLogEntry[], tail = 3): ProjectLogResponse {
  return {
    canonical_project_name: 'demo',
    entries,
    project_id: 7,
    stderr: true,
    stdout: true,
    tail,
    timestamps: true,
    truncated: false,
  };
}

describe('ProjectLogRealtimeBatcher', () => {
  it('keeps only the newest bounded entries in chronological order', () => {
    const commits: ProjectLogResponse[] = [];
    const batcher = new ProjectLogRealtimeBatcher({
      lineLimit: 3,
      onCommit: (next) => commits.push(next),
    });

    batcher.seed(
      snapshot([
        entry('middle', '2026-07-10T10:00:01Z'),
        entry('oldest', '2026-07-10T10:00:00Z'),
        entry('latest', '2026-07-10T10:00:02Z'),
      ]),
    );
    batcher.enqueue(entry('newest', '2026-07-10T10:00:03Z'));
    batcher.flush();

    expect(commits.at(-1)?.entries.map((item) => item.line)).toEqual(['middle', 'latest', 'newest']);
    expect(commits.at(-1)?.truncated).toBe(true);
  });

  it('preserves realtime entries received while the HTTP snapshot is pending', () => {
    const commits: ProjectLogResponse[] = [];
    const batcher = new ProjectLogRealtimeBatcher({
      lineLimit: 2,
      onCommit: (next) => commits.push(next),
    });

    batcher.beginSnapshot(2);
    batcher.enqueue(entry('during-load', '2026-07-10T10:00:02Z'));
    batcher.flush();
    batcher.seed(snapshot([entry('snapshot', '2026-07-10T10:00:01Z')], 2));

    expect(commits).toHaveLength(1);
    expect(commits[0]?.entries.map((item) => item.line)).toEqual(['snapshot', 'during-load']);
  });

  it('deduplicates replayed entries before publishing', () => {
    const commits: ProjectLogResponse[] = [];
    const repeated = entry('same', '2026-07-10T10:00:01Z');
    const batcher = new ProjectLogRealtimeBatcher({
      lineLimit: 3,
      onCommit: (next) => commits.push(next),
    });

    batcher.seed(snapshot([repeated]));
    batcher.enqueue(repeated);
    batcher.flush();

    expect(commits.at(-1)?.entries).toHaveLength(1);
  });

  it('retains snapshot and realtime entries when the server reports truncation', () => {
    const commits: ProjectLogResponse[] = [];
    const batcher = new ProjectLogRealtimeBatcher({
      lineLimit: 3,
      onCommit: (next) => commits.push(next),
    });

    batcher.seed({
      ...snapshot([entry('oldest', '2026-07-10T10:00:00Z'), entry('latest', '2026-07-10T10:00:01Z')]),
      truncated: true,
    });
    batcher.enqueue(entry('realtime', '2026-07-10T10:00:02Z'));
    batcher.flush();

    expect(commits.at(-1)?.entries.map((item) => item.line)).toEqual(['oldest', 'latest', 'realtime']);
    expect(commits.at(-1)?.truncated).toBe(true);
  });
});
