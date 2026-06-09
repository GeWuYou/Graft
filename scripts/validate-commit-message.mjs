// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import process from 'node:process';

const messageFile = process.argv[2];

if (!messageFile) {
  console.error('缺少 commit message 文件路径。');
  process.exit(1);
}

const message = readFileSync(messageFile, 'utf8');
const effectiveMessage = message
  .split(/\r?\n/u)
  .filter((line) => !line.startsWith('#'))
  .join('\n')
  .trim();
const escapedControlTextPattern = /\\(?:0|a|b|f|n|r|t|v)/u;

if (escapedControlTextPattern.test(effectiveMessage)) {
  console.error('提交信息不能包含字面量转义控制字符，如 \\n、\\t、\\r；请改用真实换行或缩进。');
  process.exit(1);
}

if (!effectiveMessage) {
  process.exit(0);
}

const lines = effectiveMessage.split('\n');
const subject = lines[0] ?? '';
const isMergeCommit = /^Merge\b/u.test(subject);
const isRevertCommit = /^Revert\b/u.test(subject);

if (isMergeCommit || isRevertCommit) {
  process.exit(0);
}

const bodyLines = lines.slice(1);
const hasBodyText = bodyLines.some((line) => line.trim().length > 0);
const hasBulletBody = bodyLines.some((line) => /^-\s+\S/u.test(line.trim()));

if (!hasBodyText || !hasBulletBody) {
  console.error('普通提交必须包含真实多行正文，并至少包含一条以 "- " 开头的变更说明。');
  process.exit(1);
}
