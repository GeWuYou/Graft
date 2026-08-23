import { access, lstat } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { resolve } from 'node:path';

const repositoryRoot = resolve(import.meta.dirname, '..');
const bundleScriptPath = resolve(repositoryRoot, 'scripts/openapi-bundle.mjs');
const bundlePath = resolve(repositoryRoot, 'openapi/dist/openapi.bundle.json');

function runBundleGenerator() {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(process.execPath, [bundleScriptPath], {
      cwd: repositoryRoot,
      stdio: 'inherit',
    });

    child.once('error', reject);
    child.once('exit', (code, signal) => {
      if (code === 0) {
        resolvePromise();
        return;
      }
      reject(new Error(`OpenAPI bundle generation failed${signal ? ` (${signal})` : ` with exit code ${code}`}`));
    });
  });
}

await runBundleGenerator();

try {
  const bundleInfo = await lstat(bundlePath);
  if (!bundleInfo.isFile() || bundleInfo.isSymbolicLink()) {
    throw new Error('bundle path is not a regular file');
  }
  await access(bundlePath);
} catch (error) {
  throw new Error(`OpenAPI bundle was not created at ${bundlePath}: ${error.message}`);
}
