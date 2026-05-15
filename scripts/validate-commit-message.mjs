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
  .join('\n');
const escapedControlTextPattern = /\\(?:0|a|b|f|n|r|t|v)/u;

if (escapedControlTextPattern.test(effectiveMessage)) {
  console.error('提交信息不能包含字面量转义控制字符，如 \\n、\\t、\\r；请改用真实换行或缩进。');
  process.exit(1);
}
