import assert from 'node:assert/strict';
import test from 'node:test';
import { applyApprovals, parseArguments, parseLedger, projectionHash, stableStringify } from './openapi-breaking.mjs';

function fixtureDocument() {
  return {
    openapi: '3.1.0',
    info: { title: 'Fixture', version: '1' },
    paths: {
      '/items': {
        get: {
          operationId: 'listItems',
          responses: {
            '200': {
              description: 'OK',
              content: { 'application/json': { schema: { $ref: '#/components/schemas/Item' } } },
            },
          },
        },
      },
      '/other': {
        get: { operationId: 'listOther', responses: { '200': { description: 'OK' } } },
      },
    },
    components: {
      schemas: { Item: { type: 'object', required: ['id'], properties: { id: { type: 'string' } } } },
    },
  };
}

function finding(operation, path, id) {
  return { id, level: 3, operation, path, section: 'paths', text: 'fixture finding' };
}

test('stable serialization sorts object keys but preserves array order', () => {
  assert.equal(stableStringify({ b: 1, a: 2 }), stableStringify({ a: 2, b: 1 }));
  assert.equal(stableStringify({ required: ['b', 'a'] }), stableStringify({ required: ['b', 'a'] }));
  assert.notEqual(stableStringify({ required: ['a', 'b'] }), stableStringify({ required: ['b', 'a'] }));
});

test('contract projection covers transitive schema dependencies and key order', () => {
  const base = fixtureDocument();
  const reordered = JSON.parse(stableStringify(base));
  assert.equal(projectionHash(base, '/items', 'get'), projectionHash(reordered, '/items', 'get'));

  const changed = structuredClone(base);
  changed.components.schemas.Item.properties.id.type = 'integer';
  assert.notEqual(projectionHash(base, '/items', 'get'), projectionHash(changed, '/items', 'get'));

  const unrelated = structuredClone(base);
  unrelated.paths.other = unrelated.paths['/other'];
  delete unrelated.paths['/other'];
  assert.equal(projectionHash(base, '/items', 'get'), projectionHash(unrelated, '/items', 'get'));

  const removed = structuredClone(base);
  delete removed.paths['/items'];
  assert.equal(projectionHash(removed, '/items', 'get'), null);
});

test('path-item-level shared parameters are covered but sibling operations are not', () => {
  const base = fixtureDocument();
  base.paths['/items'].parameters = [{ name: 'limit', in: 'query', schema: { type: 'integer', minimum: 1 } }];
  base.paths['/items'].post = { operationId: 'createItem', responses: { '200': { description: 'OK' } } };

  // 共享参数变化改变该 path 上每个 operation 的投影。
  const paramChanged = structuredClone(base);
  paramChanged.paths['/items'].parameters[0].schema.minimum = 10;
  assert.notEqual(projectionHash(base, '/items', 'get'), projectionHash(paramChanged, '/items', 'get'));
  assert.notEqual(projectionHash(base, '/items', 'post'), projectionHash(paramChanged, '/items', 'post'));

  // 兄弟 operation 变化不影响 get 的投影，无关端点不互相使台账失效。
  const siblingChanged = structuredClone(base);
  siblingChanged.paths['/items'].post = { operationId: 'renamedItem', responses: { '200': { description: 'OK' } } };
  assert.equal(projectionHash(base, '/items', 'get'), projectionHash(siblingChanged, '/items', 'get'));

  // 绑定含共享参数投影的审批：未评审的共享参数变化使审批过期并拒绝 finding。
  const reviewed = structuredClone(base);
  delete reviewed.components.schemas.Item.required;
  const approval = {
    reason: 'reviewed delta',
    method: 'GET',
    path: '/items',
    changeKind: 'response-required-property-removed',
    baseContractSha256: projectionHash(base, '/items', 'get'),
    newContractSha256: projectionHash(reviewed, '/items', 'get'),
  };
  const findings = [finding('GET', '/items', 'response-required-property-removed')];
  assert.deepEqual(applyApprovals(findings, base, reviewed, [approval]).unapproved, []);
  const polluted = applyApprovals(findings, base, paramChanged, [approval]);
  assert.equal(polluted.unapproved.length, 1);
  assert.equal(polluted.stale.length, 1);
});

test('approval follows effective security and its named scheme references', () => {
  const base = fixtureDocument();
  base.security = [{ privateAccess: [] }];
  base.components.securitySchemes = {
    privateAccess: { $ref: '#/components/securitySchemes/bearer' },
    bearer: { type: 'http', scheme: 'bearer' },
    key: { type: 'apiKey', in: 'header', name: 'X-Fixture-Key' },
  };
  const reviewed = structuredClone(base);
  delete reviewed.components.schemas.Item.required;
  const approval = {
    reason: 'reviewed delta',
    method: 'GET',
    path: '/items',
    changeKind: 'response-required-property-removed',
    baseContractSha256: projectionHash(base, '/items', 'get'),
    newContractSha256: projectionHash(reviewed, '/items', 'get'),
  };
  const findings = [finding('GET', '/items', approval.changeKind)];
  assert.deepEqual(applyApprovals(findings, base, reviewed, [approval]).approved, findings);

  const changedRequirement = structuredClone(reviewed);
  changedRequirement.security = [{ key: [] }];
  const changedScheme = structuredClone(reviewed);
  changedScheme.components.securitySchemes.bearer.bearerFormat = 'JWT';
  for (const revision of [changedRequirement, changedScheme]) {
    const result = applyApprovals(findings, base, revision, [approval]);
    assert.deepEqual(result.unapproved, findings);
    assert.equal(result.stale.length, 1);
  }
  assert.equal(applyApprovals(findings, changedScheme, reviewed, [approval]).stale.length, 1);

  const unrelatedScheme = structuredClone(reviewed);
  unrelatedScheme.components.securitySchemes.key.name = 'X-Other-Fixture-Key';
  assert.deepEqual(applyApprovals(findings, base, unrelatedScheme, [approval]).approved, findings);

  const publicOperation = structuredClone(reviewed);
  publicOperation.paths['/items'].get.security = [];
  const publicWithRootChange = structuredClone(publicOperation);
  publicWithRootChange.security = changedRequirement.security;
  assert.equal(projectionHash(publicOperation, '/items', 'get'), projectionHash(publicWithRootChange, '/items', 'get'));
  assert.notEqual(projectionHash(publicOperation, '/items', 'get'), projectionHash(reviewed, '/items', 'get'));
});

test('only the effective server list enters the operation projection', () => {
  const base = fixtureDocument();
  base.servers = [{ url: '/first' }];
  const rootChange = structuredClone(base);
  rootChange.servers = [{ url: '/second' }];
  assert.notEqual(projectionHash(base, '/items', 'get'), projectionHash(rootChange, '/items', 'get'));

  const pathOverride = [{ url: '/path' }];
  base.paths['/items'].servers = pathOverride;
  rootChange.paths['/items'].servers = pathOverride;
  assert.equal(projectionHash(base, '/items', 'get'), projectionHash(rootChange, '/items', 'get'));

  const pathChange = structuredClone(base);
  pathChange.paths['/items'].servers = [{ url: '/different-path' }];
  assert.notEqual(projectionHash(base, '/items', 'get'), projectionHash(pathChange, '/items', 'get'));

  const operationOverride = [{ url: '/operation' }];
  base.paths['/items'].get.servers = operationOverride;
  pathChange.paths['/items'].get.servers = operationOverride;
  assert.equal(projectionHash(base, '/items', 'get'), projectionHash(pathChange, '/items', 'get'));
});

test('operation removals are approvable through the deleted sentinel', () => {
  const base = fixtureDocument();
  const target = fixtureDocument();
  delete target.paths['/items'];
  const removal = [finding('GET', '/items', 'api-removed-without-deprecation')];
  const approval = {
    reason: 'reviewed removal',
    method: 'GET',
    path: '/items',
    changeKind: 'api-removed-without-deprecation',
    baseContractSha256: projectionHash(base, '/items', 'get'),
    newContractSha256: 'deleted',
  };
  assert.deepEqual(applyApprovals(removal, base, target, [approval]).unapproved, []);

  // operation 仍在 revision 时哨兵审批一律过期：不能放行修改而非删除的变更。
  const modified = fixtureDocument();
  modified.paths['/items'].get.responses['200'].description = 'Changed';
  const stillThere = applyApprovals(removal, base, modified, [approval]);
  assert.equal(stillThere.unapproved.length, 1);
  assert.equal(stillThere.stale.length, 1);
  assert.equal(stillThere.stale[0].actualNewContractSha256, projectionHash(modified, '/items', 'get'));

  // base 哈希漂移时同样拒绝。
  assert.equal(applyApprovals(removal, modified, target, [approval]).stale.length, 1);
});

test('ledger parsing fails closed on schema drift', () => {
  assert.deepEqual(parseLedger('{"schemaVersion":1,"approvals":[]}'), []);
  assert.throws(() => parseLedger('{"schemaVersion":2,"approvals":[]}'));
  assert.throws(() => parseLedger('{"schemaVersion":1}'));
  assert.throws(() => parseLedger('not json'));
  const approval = {
    reason: 'reviewed',
    method: 'GET',
    path: '/items',
    changeKind: 'response-required-property-removed',
    baseContractSha256: 'a'.repeat(64),
    newContractSha256: 'b'.repeat(64),
  };
  assert.deepEqual(parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [approval] }))[0].method, 'get');
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...approval, extra: true }] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...approval, baseContractSha256: 'zz' }] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...approval, method: 'BREW' }] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...approval, path: 'items' }] })));
});

test('ledger parsing accepts schema docs and the deleted sentinel but rejects drift', () => {
  assert.deepEqual(parseLedger(JSON.stringify({ schemaVersion: 1, schemaDocs: 'how entries bind', approvals: [] })), []);
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, extra: true, approvals: [] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, schemaDocs: 5, approvals: [] })));
  const sentinel = {
    reason: 'reviewed removal',
    method: 'GET',
    path: '/items',
    changeKind: 'api-removed-without-deprecation',
    baseContractSha256: 'a'.repeat(64),
    newContractSha256: 'deleted',
  };
  assert.deepEqual(parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [sentinel] }))[0].newContractSha256, 'deleted');
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...sentinel, newContractSha256: 'Deleted' }] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...sentinel, newContractSha256: 'gone' }] })));
  assert.throws(() => parseLedger(JSON.stringify({ schemaVersion: 1, approvals: [{ ...sentinel, baseContractSha256: 'deleted' }] })));
});

test('argument parsing accepts a positional or --base ref but never both', () => {
  assert.deepEqual(parseArguments([]), { base: undefined, mode: 'gate' });
  assert.deepEqual(parseArguments(['main']), { base: 'main', mode: 'gate' });
  assert.deepEqual(parseArguments(['origin/main']), { base: 'origin/main', mode: 'gate' });
  assert.deepEqual(parseArguments(['--base', 'origin/main']), { base: 'origin/main', mode: 'gate' });
  assert.deepEqual(parseArguments(['--mode', 'report']), { base: undefined, mode: 'report' });
  assert.deepEqual(parseArguments(['--base', 'HEAD', '--mode', 'report']), { base: 'HEAD', mode: 'report' });
  assert.throws(() => parseArguments(['main', 'dev']));
  assert.throws(() => parseArguments(['main', '--base', 'dev']));
  assert.throws(() => parseArguments(['--base', 'dev', 'main']));
  assert.throws(() => parseArguments(['--wat']));
  assert.throws(() => parseArguments(['--base']));
  assert.throws(() => parseArguments(['--mode', 'nope']));
});

test('approvals apply only to the exact reviewed delta on unchanged contracts', () => {
  const base = fixtureDocument();
  const target = fixtureDocument();
  // 真实破坏性变更：移除被传递引用的 schema required 属性。
  delete target.components.schemas.Item.required;
  const approval = {
    reason: 'reviewed fixture delta',
    method: 'GET',
    path: '/items',
    changeKind: 'response-required-property-removed',
    baseContractSha256: projectionHash(base, '/items', 'get'),
    newContractSha256: projectionHash(target, '/items', 'get'),
  };
  const intended = [finding('GET', '/items', 'response-required-property-removed')];

  // 同一 finding 且契约与评审时一致：审批生效。
  assert.deepEqual(applyApprovals(intended, base, target, [approval]).unapproved, []);

  // 不同 method/path/change-kind 都不能复用审批。
  assert.equal(applyApprovals([finding('POST', '/items', 'response-required-property-removed')], base, target, [approval]).unapproved.length, 1);
  assert.equal(applyApprovals([finding('GET', '/other', 'response-required-property-removed')], base, target, [approval]).unapproved.length, 1);
  assert.equal(applyApprovals([finding('GET', '/items', 'different-kind')], base, target, [approval]).unapproved.length, 1);

  // 契约任何一侧漂移都会让审批过期并被拒绝。
  const revisedScope = structuredClone(target);
  revisedScope.paths['/items'].get.summary = 'newly documented';
  const drifted = applyApprovals(intended, base, revisedScope, [approval]);
  assert.equal(drifted.unapproved.length, 1);
  assert.equal(drifted.stale.length, 1);
  assert.equal(applyApprovals(intended, target, target, [approval]).stale.length, 1);

  // 无关端点同时变化不影响已评审范围的审批，但另一端点的 finding 仍被拒绝。
  const widened = structuredClone(target);
  widened.paths['/other'].get = { operationId: 'listOther' };
  const result = applyApprovals(
    [finding('GET', '/items', 'response-required-property-removed'), finding('GET', '/other', 'operation-removed')],
    base,
    widened,
    [approval],
  );
  assert.deepEqual(result.approved, [finding('GET', '/items', 'response-required-property-removed')]);
  assert.deepEqual(result.unapproved, [finding('GET', '/other', 'operation-removed')]);
});
