// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { ScheduledTaskItem, ScheduledTaskJobDefinitionItem } from '../types/scheduled-task';

type Translate = (key: string, params?: Record<string, unknown>) => string;
type TranslationExists = (key: string) => boolean;
type ScheduledTaskJobSummary = NonNullable<ScheduledTaskItem['job']>;
type ScheduledTaskJobDisplay = ScheduledTaskJobDefinitionItem | ScheduledTaskJobSummary;

export type ScheduledTaskPresenterI18n = {
  t: Translate;
  te: TranslationExists;
};

export type ScheduledTaskRowView = {
  taskKey: string;
  taskTitle: string;
  taskDescription?: string;

  jobKey: string;
  jobTitle: string;
  jobShortTitle: string;
  jobCategory: string;
  jobCategoryLabel: string;
  moduleKey: string;
  moduleLabel: string;

  jobDisplayLabel: string;
  jobTooltip: string;

  cronExpression: string;
  nextRunLabel: string;
  nextRunTooltip: string;

  statusLabel: string;
  recentResultLabel: string;
  recentResultTooltip: string;
};

export type PresentScheduledTaskRowInput = {
  task: ScheduledTaskItem;
  job?: ScheduledTaskJobDisplay | null;
  i18n: ScheduledTaskPresenterI18n;
  cronExpression: string;
  nextRunLabel: string;
  nextRunTooltip: string;
  statusLabel: string;
  recentResultLabel: string;
  recentResultTooltip: string;
};

export function presentScheduledTaskRow(input: PresentScheduledTaskRowInput): ScheduledTaskRowView {
  const job = input.job ?? input.task.job;
  const moduleKey = job?.module_key ?? '';
  const category = job?.category ?? 'custom';
  const categoryKey = job?.category_key ?? `scheduler.job.category.${category}`;
  const taskDisplayTitle = taskTitle(input.task, input.i18n);
  const fullJobTitle = job ? jobTitle(job, input.i18n) : input.task.job_key;
  const shortJobTitle = job ? jobShortTitle(job, input.i18n) : input.task.job_key;
  const categoryText =
    localizeDisplayText(input.i18n, categoryKey, '', true) || input.i18n.t(`scheduler.job.category.${category}`);
  const moduleText = moduleKey ? moduleLabel(moduleKey, input.i18n) : '';

  return {
    taskKey: input.task.task_key,
    taskTitle: taskDisplayTitle,
    taskDescription: taskDescription(input.task, input.i18n),
    jobKey: input.task.job_key,
    jobTitle: fullJobTitle,
    jobShortTitle: shortJobTitle,
    jobCategory: category,
    jobCategoryLabel: categoryText,
    moduleKey,
    moduleLabel: moduleText,
    jobDisplayLabel: moduleText ? `${categoryText} · ${moduleText}` : categoryText,
    jobTooltip: `${fullJobTitle}\n${input.task.job_key}`,
    cronExpression: input.cronExpression,
    nextRunLabel: input.nextRunLabel,
    nextRunTooltip: input.nextRunTooltip,
    statusLabel: input.statusLabel,
    recentResultLabel: input.recentResultLabel,
    recentResultTooltip: input.recentResultTooltip,
  };
}

function localizeMessageKey(i18n: ScheduledTaskPresenterI18n, key?: string) {
  const trimmed = key?.trim();
  if (!trimmed) {
    return '';
  }

  return i18n.te(trimmed) ? i18n.t(trimmed) : '';
}

function localizeDisplayText(
  i18n: ScheduledTaskPresenterI18n,
  messageKey?: string,
  fallback?: string | null,
  preferMessageKey = false,
) {
  const localized = localizeMessageKey(i18n, messageKey);
  if (preferMessageKey && localized) {
    return localized;
  }

  const fallbackText = fallback?.trim() ?? '';
  if (!fallbackText) {
    return localized;
  }

  return localizeMessageKey(i18n, fallbackText) || fallbackText || localized;
}

export function taskTitle(task: ScheduledTaskItem, i18n: ScheduledTaskPresenterI18n) {
  return localizeDisplayText(i18n, task.title_key, task.title, task.builtin) || task.task_key;
}

export function taskDescription(task: ScheduledTaskItem, i18n: ScheduledTaskPresenterI18n) {
  return localizeDisplayText(i18n, task.description_key, task.description, task.builtin);
}

export function jobTitle(job: ScheduledTaskJobDisplay, i18n: ScheduledTaskPresenterI18n) {
  return localizeDisplayText(i18n, job.title_key, job.title, true) || job.job_key;
}

export function jobShortTitle(job: ScheduledTaskJobDisplay, i18n: ScheduledTaskPresenterI18n) {
  return localizeDisplayText(i18n, job.short_title_key, job.short_title, true) || jobTitle(job, i18n);
}

export function jobDescription(job: ScheduledTaskJobDisplay, i18n: ScheduledTaskPresenterI18n) {
  return localizeDisplayText(i18n, job.description_key, job.description, true);
}

export function jobCategoryLabel(
  job: Pick<ScheduledTaskJobDefinitionItem, 'category' | 'category_key'>,
  i18n: ScheduledTaskPresenterI18n,
) {
  return localizeDisplayText(i18n, job.category_key, '', true) || i18n.t(`scheduler.job.category.${job.category}`);
}

export function moduleLabel(moduleKey: string, i18n: ScheduledTaskPresenterI18n) {
  const key = `module.${moduleKey}.title`;
  return i18n.te(key) ? i18n.t(key) : moduleKey;
}
