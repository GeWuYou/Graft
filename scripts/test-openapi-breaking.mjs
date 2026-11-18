import assert from 'node:assert/strict';
import test from 'node:test';
import { applyApprovals, parseLedger, projectionHash, stableStringify } from './openapi-breaking.mjs';

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
