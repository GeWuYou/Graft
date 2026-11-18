import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { createHash } from 'node:crypto';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const exec = promisify(execFile);

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const sourceSpecPath = resolve(repositoryRoot, 'openapi/openapi.yaml');
const ledgerPath = resolve(repositoryRoot, 'openapi/approved-breaking.json');

const requireFromRoot = createRequire(resolve(repositoryRoot, 'package.json'));
const { bundle, createConfig } = requireFromRoot('@redocly/openapi-core');

// oasdiff marks breaking changes with level 3; this mirrors `oasdiff --fail-on ERR`.
const ERR_LEVEL = 3;
const HTTP_METHODS = new Set(['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace']);
const BASE_SOURCE_GRAPH = 'openapi';
const LEDGER_SCHEMA_VERSION = 1;
const APPROVAL_FIELDS = ['reason', 'method', 'path', 'changeKind', 'baseContractSha256', 'newContractSha256'];
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const REF_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._/-]*$/;

export function fail(message) {
  throw new Error('OpenAPI breaking gate: ' + message);
}

export function isRecord(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

// 键序归一化但保留数组顺序：required/enum 的顺序本身是契约语义。
export function stableStringify(value) {
  const serialized = JSON.stringify(value, (_key, item) =>
    isRecord(item)
      ? Object.fromEntries(Object.keys(item).sort().map((key) => [key, item[key]]))
      : item,
  );
  if (serialized === undefined) fail('cannot serialize contract projection');
  return serialized;
}

export function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

export function references(value) {
  if (Array.isArray(value)) return value.flatMap(references);
  if (!isRecord(value)) return [];
  const mapping = isRecord(value.discriminator) && isRecord(value.discriminator.mapping)
    ? value.discriminator.mapping
    : {};
  return [
    ...Object.values(mapping).map(String),
    ...Object.entries(value).flatMap(([key, child]) => (key === '$ref' ? [String(child)] : references(child))),
  ];
}

function pointer(document, fragment) {
  if (!fragment.startsWith('#/')) fail('bundled contract must only contain local references, got ' + fragment);
  let value = document;
  for (const part of fragment.slice(2).split('/')) {
    const key = decodeURIComponent(part).replaceAll('~1', '/').replaceAll('~0', '~');
    if (!isRecord(value) || !Object.hasOwn(value, key)) fail('unresolved contract pointer ' + fragment);
    value = value[key];
  }
  return value;
}

// 契约投影 = 该 operation 对象加上其传递引用的全部本地组件，
// 哈希只覆盖评审范围，避免无关端点变化使审批台账失效。
export function projectionHash(document, path, method) {
  const pathItem = isRecord(document?.paths) ? document.paths[path] : undefined;
  const operationKey = method.toLowerCase();
  if (!isRecord(pathItem) || !isRecord(pathItem[operationKey])) return null;
  const operation = pathItem[operationKey];
  const dependencies = {};
  const pending = references(operation);
  while (pending.length > 0) {
    const fragment = pending.pop();
    if (Object.hasOwn(dependencies, fragment)) continue;
    const value = pointer(document, fragment);
    dependencies[fragment] = value;
    pending.push(...references(value));
  }
  return sha256(stableStringify({ operation, dependencies }));
}

function normalizeMethod(value, label) {
  const method = String(value ?? '').trim().toLowerCase();
  if (!HTTP_METHODS.has(method)) fail(label + ' must be one of ' + [...HTTP_METHODS].join(', '));
  return method;
}

// 台账校验失败一律拒绝（fail closed）：字段缺失、未知字段或坏哈希都不进入匹配。
export function parseLedger(raw, label = ledgerPath) {
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    fail(label + ' is not valid JSON: ' + error.message);
  }
  if (!isRecord(parsed)) fail(label + ' must contain an object');
  if (parsed.schemaVersion !== LEDGER_SCHEMA_VERSION) {
    fail(label + ' schemaVersion must be ' + LEDGER_SCHEMA_VERSION + ', got ' + JSON.stringify(parsed.schemaVersion));
  }
  if (!Array.isArray(parsed.approvals)) fail(label + ' approvals must be an array');
  return parsed.approvals.map((entry, index) => {
    const where = label + ' approvals[' + index + ']';
    if (!isRecord(entry)) fail(where + ' must be an object');
    for (const field of Object.keys(entry)) {
      if (!APPROVAL_FIELDS.includes(field)) fail(where + ' has unknown field ' + JSON.stringify(field));
    }
    for (const field of APPROVAL_FIELDS) {
      if (typeof entry[field] !== 'string' || entry[field].trim() === '') {
        fail(where + ' field ' + field + ' must be a non-empty string');
      }
    }
    if (!SHA256_PATTERN.test(entry.baseContractSha256)) fail(where + ' baseContractSha256 must be a 64-char lowercase hex sha256');
    if (!SHA256_PATTERN.test(entry.newContractSha256)) fail(where + ' newContractSha256 must be a 64-char lowercase hex sha256');
    if (!entry.path.startsWith('/')) fail(where + ' path must start with /');
    return {
      reason: entry.reason,
      method: normalizeMethod(entry.method, where + ' method'),
      path: entry.path,
      changeKind: entry.changeKind,
      baseContractSha256: entry.baseContractSha256,
      newContractSha256: entry.newContractSha256,
    };
  });
}

const approvalKey = (approval) => approval.method.toUpperCase() + '|' + approval.path + '|' + approval.changeKind;

// 审批只在契约投影哈希与当前两侧文档完全一致时生效；哈希漂移即视为过期并拒绝。
export function applyApprovals(findings, baseDocument, revisionDocument, approvals) {
  const active = new Set();
  const stale = [];
  const cache = new Map();
  const projected = (document, path, method) => {
    const cacheKey = (document === baseDocument ? 'base|' : 'rev|') + method + '|' + path;
    if (!cache.has(cacheKey)) cache.set(cacheKey, projectionHash(document, path, method));
    return cache.get(cacheKey);
  };
  for (const approval of approvals) {
    const actualBase = projected(baseDocument, approval.path, approval.method);
    const actualRevision = projected(revisionDocument, approval.path, approval.method);
    if (actualBase === null || actualRevision === null || actualBase !== approval.baseContractSha256 || actualRevision !== approval.newContractSha256) {
      stale.push({ ...approval, actualBaseContractSha256: actualBase, actualNewContractSha256: actualRevision });
      continue;
    }
    active.add(approvalKey(approval));
  }
  const approved = [];
  const unapproved = [];
  for (const finding of findings) {
    const key = String(finding.operation ?? '').toUpperCase() + '|' + String(finding.path ?? '') + '|' + String(finding.id ?? '');
    (active.has(key) ? approved : unapproved).push(finding);
  }
  return { approved, unapproved, stale };
}

async function resolveBaseRevision(explicitBase) {
  if (explicitBase !== undefined) {
    if (!REF_PATTERN.test(explicitBase)) fail('base ref must match ' + REF_PATTERN + ', got ' + JSON.stringify(explicitBase));
    const revision = (await exec('git', ['rev-parse', '--verify', explicitBase + '^{commit}'], { cwd: repositoryRoot, maxBuffer: 1024 * 1024 })).stdout.trim();
    if (revision === '') fail('base ref ' + explicitBase + ' did not resolve to a commit');
    return revision;
  }
  try {
    const merged = (await exec('git', ['merge-base', 'origin/main', 'HEAD'], { cwd: repositoryRoot, maxBuffer: 1024 * 1024 })).stdout.trim();
    if (merged !== '') return merged;
  } catch {
    // merge-base 需要 origin/main 存在；缺失时退回 origin/main 本身。
  }
  return (await exec('git', ['rev-parse', '--verify', 'origin/main^{commit}'], { cwd: repositoryRoot, maxBuffer: 1024 * 1024 })).stdout.trim();
}

// openapi/dist 不入库，基线 bundle 必须从基线提交的 openapi 源图重新物化后再走同一 redocly 打包路径。
async function materializeBaseGraph(revision, destination) {
  const listed = (await exec('git', ['ls-tree', '-r', '--name-only', revision, '--', BASE_SOURCE_GRAPH + '/'], { cwd: repositoryRoot, maxBuffer: 16 * 1024 * 1024 })).stdout;
  const paths = listed.split('\n').filter((path) => path !== '');
  if (paths.length === 0) fail('base revision ' + revision + ' contains no ' + BASE_SOURCE_GRAPH + '/ source graph');
  for (const path of paths) {
    if (!path.startsWith(BASE_SOURCE_GRAPH + '/') || path.split('/').includes('..')) fail('unsafe baseline path ' + path);
    if (!/\.(yaml|yml|json)$/.test(path)) continue;
    const content = (await exec('git', ['show', revision + ':' + path], { cwd: repositoryRoot, maxBuffer: 32 * 1024 * 1024 })).stdout;
    const target = join(destination, path);
    await mkdirFor(target);
    await writeFile(target, content);
  }
  return join(destination, BASE_SOURCE_GRAPH, 'openapi.yaml');
}

async function mkdirFor(target) {
  await mkdir(dirname(target), { recursive: true });
}

async function bundleSpec(specPath, label) {
  const config = await createConfig({}, { extends: ['minimal'] });
  const result = await bundle({ config, ref: specPath });
  const problems = result.problems.filter((problem) => problem.severity === 'error');
  if (problems.length > 0) fail('bundle ' + label + ' spec failed:\n' + problems.map((problem) => problem.message).join('\n'));
  return result.bundle.parsed;
}

async function runOasdiff(baseSpecPath, revisionSpecPath) {
  const result = await exec(
    'go',
    ['tool', 'oasdiff', 'breaking', '--allow-external-refs=false', '--flatten-allof', '--format', 'json', baseSpecPath, revisionSpecPath],
    { cwd: resolve(repositoryRoot, 'server'), maxBuffer: 64 * 1024 * 1024 },
  );
  let parsed;
  try {
    parsed = JSON.parse(result.stdout || '[]');
  } catch (error) {
    fail('oasdiff produced invalid JSON: ' + error.message);
  }
  if (!Array.isArray(parsed)) fail('oasdiff JSON findings must be an array');
  for (const finding of parsed) {
    if (!isRecord(finding) || !Number.isInteger(finding.level) || typeof finding.id !== 'string') {
      fail('unexpected oasdiff finding shape: ' + stableStringify(finding).slice(0, 300));
    }
  }
  return parsed;
}

function printFinding(finding) {
  return String(finding.operation ?? '').toUpperCase() + ' ' + String(finding.path ?? '') + ' [' + String(finding.id ?? '') + '] level ' + finding.level + ': ' + String(finding.text ?? '');
}

function parseArguments(argv) {
  let base;
  let mode = 'gate';
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--base') {
      base = argv[index + 1];
      if (base === undefined) fail('--base requires a git ref argument');
      index += 1;
    } else if (argument === '--mode') {
      mode = argv[index + 1];
      if (mode !== 'gate' && mode !== 'report') fail("--mode must be 'gate' or 'report'");
      index += 1;
    } else {
      fail('unknown argument ' + JSON.stringify(argument) + '; supported: --base <ref>, --mode gate|report');
    }
  }
  return { base, mode };
}

async function main() {
  const { base, mode } = parseArguments(process.argv.slice(2));
  const baseRevision = await resolveBaseRevision(base);
  const temporary = await mkdtemp(join(tmpdir(), 'graft-openapi-breaking-'));
  try {
    const baseSpecPath = await materializeBaseGraph(baseRevision, temporary);
    const [baseDocument, revisionDocument] = await Promise.all([
      bundleSpec(baseSpecPath, 'base ' + baseRevision.slice(0, 12)),
      bundleSpec(sourceSpecPath, 'current'),
    ]);
    const baseSpecFile = join(temporary, 'base.bundle.json');
    const revisionSpecFile = join(temporary, 'revision.bundle.json');
    await writeFile(baseSpecFile, JSON.stringify(baseDocument));
    await writeFile(revisionSpecFile, JSON.stringify(revisionDocument));

    const findings = await runOasdiff(baseSpecFile, revisionSpecFile);
    const errors = findings.filter((finding) => finding.level >= ERR_LEVEL);
    const approvals = parseLedger(await readFile(ledgerPath, 'utf8'));
    const { approved, unapproved, stale } = applyApprovals(errors, baseDocument, revisionDocument, approvals);

    const summary = {
      baseRevision,
      findings: findings.length,
      breaking: errors.length,
      approved: approved.length,
      unapproved: unapproved.length,
      staleApprovals: stale.length,
      oasdiffFindings: findings,
      unapprovedFindings: unapproved,
      staleLedgerEntries: stale.map(({ reason, method, path, changeKind }) => ({ reason, method: method.toUpperCase(), path, changeKind })),
    };
    for (const entry of stale) {
      console.error('stale approval ledger entry (contract hashes no longer match): ' + entry.method.toUpperCase() + ' ' + entry.path + ' [' + entry.changeKind + '] ' + entry.reason);
    }
    process.stdout.write(JSON.stringify(summary, null, 2) + '\n');
    if (mode === 'report') return;
    for (const finding of unapproved) {
      console.error('unapproved breaking change: ' + printFinding(finding));
    }
    if (unapproved.length > 0) fail(unapproved.length + ' unapproved breaking API change(s) against base ' + baseRevision.slice(0, 12));
  } finally {
    await rm(temporary, { recursive: true, force: true });
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  try {
    await main();
  } catch (error) {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  }
}
