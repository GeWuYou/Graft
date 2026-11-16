import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdirSync, mkdtempSync, rmSync, existsSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { taskArguments, buildContext } from './turbo.mjs';

test('public entrypoint fixes local caching and rejects overrides', () => {
  assert.deepEqual(taskArguments(['build', '--filter=graft-web']).slice(0, 3), ['run', 'build', '--cache=local:rw']);
  assert.ok(taskArguments(['build', '--force']).includes('--cache=local:w'));
  for (const option of ['--cache=remote:rw', '--env-mode=strict', '--token=secret', '--team=external']) {
    assert.throws(() => taskArguments(['build', option]));
  }
});
test('toolchain identity changes when installed versions change', () => {
  assert.notEqual(buildContext(() => 'one'), buildContext(() => 'two'));
});
test('only frontend build has result caching and complete shared inputs', () => {
  const config = JSON.parse(readFileSync(new URL('../turbo.json', import.meta.url), 'utf8'));
  assert.deepEqual(Object.entries(config.tasks).filter(([, value]) => value.cache === true).map(([name]) => name), ['graft-web#build']);
  const build = config.tasks['graft-web#build'];
  assert.ok(build.inputs.includes('$TURBO_DEFAULT$'));
  assert.ok(build.inputs.includes('$TURBO_ROOT$/openapi/**'));
  assert.ok(build.inputs.includes('$TURBO_ROOT$/web/.env*'));
  assert.ok(build.env.includes('GRAFT_BUILD_CONTEXT'));
  assert.ok(build.env.includes('VITE_*'));
  assert.ok(config.globalDependencies.includes('bun.lock'));
  assert.deepEqual(build.outputs, ['dist/**']);
});


test('real task hashes cover production inputs and cache restores outputs', () => {
  const directory = mkdtempSync(join(tmpdir(), 'graft-turbo-test-'));
  const binary = fileURLToPath(new URL('../node_modules/turbo/bin/turbo', import.meta.url));
  const config = readFileSync(new URL('../turbo.json', import.meta.url));
  const lock = readFileSync(new URL('../bun.lock', import.meta.url));
  const write = (path, content) => {
    const target = join(directory, path);
    mkdirSync(join(target, '..'), { recursive: true });
    writeFileSync(target, content);
  };
  const env = { ...process.env, GRAFT_BUILD_CONTEXT: 'fixture-v1', TURBO_TELEMETRY_DISABLED: '1' };
  const run = (extra = [], overrides = {}) => execFileSync('bun', [
    binary, 'run', 'build', '--filter=graft-web', '--cache=local:rw', '--env-mode=loose', ...extra,
  ], { cwd: directory, env: { ...env, ...overrides }, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  const hash = (overrides = {}) => JSON.parse(run(['--dry=json'], overrides)).tasks.find((task) => task.taskId === 'graft-web#build').hash;
  try {
    write('package.json', JSON.stringify({ name: 'graft-tools', private: true, packageManager: 'bun@1.4.4', workspaces: ['web', 'server'] }));
    write('bun.lock', lock);
    write('turbo.json', config);
    write('scripts/turbo.mjs', '// fixture wrapper input');
    write('.gitignore', 'node_modules/\n.turbo/\nweb/dist/\nweb/.env*\n.executions\n');
    write('server/package.json', JSON.stringify({ name: 'graft-server', private: true }));
    const build = "node -e \"const fs=require('fs');fs.mkdirSync('dist',{recursive:true});fs.writeFileSync('dist/index.html','built');fs.appendFileSync('../.executions','run\\n')\"";
    write('web/package.json', JSON.stringify({ name: 'graft-web', private: true, scripts: { build } }));
    write('web/src/main.ts', 'export const value = 1;');
    write('web/src/contracts/openapi/generated/schema.ts', 'export type Schema = string;');
    write('web/vite.config.ts', 'export default {};');
    write('openapi/openapi.yaml', 'openapi: 3.1.0');
    execFileSync('git', ['init', '-q'], { cwd: directory });
    const baseline = hash();
    for (const [path, initial] of [
      ['web/src/main.ts', 'export const value = 1;'],
      ['web/src/contracts/openapi/generated/schema.ts', 'export type Schema = string;'],
      ['web/vite.config.ts', 'export default {};'],
      ['openapi/openapi.yaml', 'openapi: 3.1.0'],
    ]) {
      write(path, initial + '\n// changed');
      assert.notEqual(hash(), baseline, path);
      write(path, initial);
    }
    write('web/.env.release', 'VITE_BASE_URL=/changed');
    assert.notEqual(hash(), baseline, 'ignored environment files are explicit inputs');
    rmSync(join(directory, 'web/.env.release'));
    write('bun.lock', Buffer.concat([lock, Buffer.from('\n')]));
    assert.notEqual(hash(), baseline, 'lock contents');
    write('bun.lock', lock);
    assert.notEqual(hash({ VITE_BASE_URL: '/other' }), baseline, 'build variables');
    assert.notEqual(hash({ GRAFT_BUILD_CONTEXT: 'fixture-v2' }), baseline, 'toolchain identity');
    run();
    assert.equal(readFileSync(join(directory, '.executions'), 'utf8'), 'run\n');
    rmSync(join(directory, 'web/dist'), { recursive: true });
    run();
    assert.equal(readFileSync(join(directory, '.executions'), 'utf8'), 'run\n');
    assert.ok(existsSync(join(directory, 'web/dist/index.html')));
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});


test('code graph retains one full backend gate and independent web checks', () => {
  const server = JSON.parse(readFileSync(new URL('../server/package.json', import.meta.url), 'utf8'));
  const web = JSON.parse(readFileSync(new URL('../web/package.json', import.meta.url), 'utf8'));
  assert.equal(server.scripts['check:code'], 'go run ./cmd/graft validate backend');
  assert.equal(server.scripts['check:tests'], undefined);
  assert.equal(server.scripts['check:source'], undefined);
  assert.ok(web.scripts['check:source']);
  const args = taskArguments(['check:code']);
  assert.deepEqual(args.slice(0, 4), ['run', 'check:code', 'check:tests', 'check:build']);
  assert.ok(args.includes('--continue=dependencies-successful'));
});
