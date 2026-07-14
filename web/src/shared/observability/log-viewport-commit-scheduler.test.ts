import { afterEach, describe, expect, it, vi } from 'vitest';

import { LogViewportCommitScheduler } from './log-viewport-commit-scheduler';

describe('LogViewportCommitScheduler', () => {
  afterEach(() => vi.useRealTimers());

  it('commits the newest snapshot once per flush window', () => {
    vi.useFakeTimers();
    const commits: string[] = [];
    const scheduler = new LogViewportCommitScheduler({
      onCommit: (value: string) => commits.push(value),
      requestFrame: (callback) => {
        callback(0);
        return 1;
      },
      cancelFrame: () => undefined,
    });

    scheduler.publish('first');
    scheduler.publish('latest');
    vi.advanceTimersByTime(100);

    expect(commits).toEqual(['latest']);
  });

  it('defers visible commits until a viewport interaction becomes idle', () => {
    vi.useFakeTimers();
    const commits: string[] = [];
    const scheduler = new LogViewportCommitScheduler({
      onCommit: (value: string) => commits.push(value),
      requestFrame: (callback) => {
        callback(0);
        return 1;
      },
      cancelFrame: () => undefined,
    });

    scheduler.beginInteraction();
    scheduler.publish('during-drag');
    vi.advanceTimersByTime(1_000);
    expect(commits).toEqual([]);

    scheduler.endInteraction();
    vi.advanceTimersByTime(119);
    expect(commits).toEqual([]);
    vi.advanceTimersByTime(1);
    vi.advanceTimersByTime(100);

    expect(commits).toEqual(['during-drag']);
  });

  it('commits the initial snapshot synchronously', () => {
    const commits: string[] = [];
    const scheduler = new LogViewportCommitScheduler({ onCommit: (value: string) => commits.push(value) });

    scheduler.publish('initial', true);

    expect(commits).toEqual(['initial']);
  });

  it('drops pending work after destruction', () => {
    vi.useFakeTimers();
    const onCommit = vi.fn();
    const scheduler = new LogViewportCommitScheduler({ onCommit });

    scheduler.publish('pending');
    scheduler.destroy();
    vi.runAllTimers();

    expect(onCommit).not.toHaveBeenCalled();
  });
});
