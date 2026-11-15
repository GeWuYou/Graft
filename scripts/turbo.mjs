// 公共任务入口固定缓存边界，并将实际工具链身份纳入前端构建缓存。
import { execFileSync, spawn } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

export function taskArguments(args) {
  if (!['build', 'test', 'check:code'].includes(args[0])) {
    throw new Error('Expected build, test, or check:code');
  }
  const options = args.slice(1);
  if (options.some((arg) => !/^--(?:filter=.+|dry(?:=json)?|force(?:=true)?)$/.test(arg))) {
    throw new Error('Supported options are --filter=package, --dry=json, and --force');
  }
  const force = options.some((arg) => /^--force(?:=true)?$/.test(arg));
  const tasks = args[0] === 'check:code' ? ['check:code', 'check:tests', 'check:build'] : [args[0]];
  return [
    'run', ...tasks, force ? '--cache=local:w' : '--cache=local:rw',
    '--env-mode=loose', '--continue=dependencies-successful', '--log-order=stream',
    ...options.filter((arg) => !/^--force/.test(arg)),
  ];
}

export function buildContext(version = (binary) => execFileSync(binary, ['--version'], { encoding: 'utf8' }).trim()) {
  return JSON.stringify({ platform: process.platform, arch: process.arch, bun: version('bun'), node: version('node') });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  try {
    const root = dirname(dirname(fileURLToPath(import.meta.url)));
    const child = spawn(process.execPath, [join(root, 'node_modules/turbo/bin/turbo'), ...taskArguments(process.argv.slice(2)), `--cache-dir=${join(root, '.turbo/cache')}`], {
      cwd: root, stdio: 'inherit',
      env: { ...process.env, GRAFT_BUILD_CONTEXT: buildContext(), TURBO_TELEMETRY_DISABLED: '1' },
    });
    for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => child.kill(signal));
    child.on('error', (error) => { console.error(error.message); process.exitCode = 1; });
    child.on('exit', (code, signal) => { process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 1); });
  } catch (error) {
    console.error(error.message);
    process.exitCode = 2;
  }
}
