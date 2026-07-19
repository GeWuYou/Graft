import { readFileSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { resolve } from 'node:path';

const repositoryRoot = resolve(import.meta.dirname, '..');
const bundlePath = resolve(repositoryRoot, 'openapi/dist/openapi.bundle.json');
const outputPath = resolve(repositoryRoot, 'web/src/contracts/generated/openapi-runtime-paths.ts');
const requireFromWeb = createRequire(resolve(repositoryRoot, 'web/package.json'));
const prettier = requireFromWeb('prettier');
const prettierConfig = (await prettier.resolveConfig(outputPath)) ?? {};
const methods = new Set(['delete', 'get', 'head', 'options', 'patch', 'post', 'put']);
const checkOnly = process.argv.includes('--check');

function fail(message) {
  throw new Error('OpenAPI runtime path projection: ' + message);
}

function readOperations() {
  const bundle = JSON.parse(readFileSync(bundlePath, 'utf8'));
  const operations = [];
  const seen = new Set();

  for (const [path, pathItem] of Object.entries(bundle.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!methods.has(method)) continue;
      if (!operation || typeof operation !== 'object' || Array.isArray(operation)) {
        fail(method.toUpperCase() + ' ' + path + ' must define an operation object');
      }
      if (typeof operation.operationId !== 'string' || operation.operationId.trim() === '') {
        fail(method.toUpperCase() + ' ' + path + ' is missing operationId');
      }
      if (seen.has(operation.operationId)) fail('duplicate operationId ' + JSON.stringify(operation.operationId));
      seen.add(operation.operationId);
      operations.push({ operationId: operation.operationId, path });
    }
  }

  return operations.sort((left, right) => left.operationId.localeCompare(right.operationId));
}

async function render(operations) {
  const entries = operations.map(({ operationId, path }) => '  ' + operationId + ': ' + JSON.stringify(path) + ',').join('\n');
  const source = [
    '/**',
    ' * Generated from openapi/openapi.yaml through openapi/dist/openapi.bundle.json.',
    ' * Do not edit manually; run just generate.',
    ' */',
    '',
    'export const OPENAPI_RUNTIME_PATH = {',
    entries,
    '} as const;',
    '',
    'export type OpenApiOperationId = keyof typeof OPENAPI_RUNTIME_PATH;',
    '',
    'export type OpenApiPathParameter = string | number;',
    '',
    '/**',
    ' * Builds a runtime API path from an OpenAPI operation id and percent-encodes path parameters.',
    ' */',
    'export function buildOpenApiRuntimePath(',
    '  operationId: OpenApiOperationId,',
    '  parameters: Record<string, OpenApiPathParameter> = {},',
    '): string {',
    '  return OPENAPI_RUNTIME_PATH[operationId].replace(/\\{([^}]+)\\}/g, (_placeholder, parameterName: string) => {',
    '    if (!Object.prototype.hasOwnProperty.call(parameters, parameterName)) {',
    "      throw new Error('Missing path parameter ' + parameterName + ' for OpenAPI operation ' + operationId);",
    '    }',
    '    return encodeURIComponent(String(parameters[parameterName]));',
    '  });',
    '}',
    '',
  ].join('\n');
  return prettier.format(source, { ...prettierConfig, filepath: outputPath });
}

const expected = await render(readOperations());
if (checkOnly) {
  const actual = readFileSync(outputPath, 'utf8');
  if (actual !== expected) {
    fail('generated artifact is stale; run just generate');
  }
} else {
  writeFileSync(outputPath, expected);
}
