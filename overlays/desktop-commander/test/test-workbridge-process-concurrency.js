import assert from 'node:assert/strict';
import {
  DEFAULT_WORKBRIDGE_PROCESS_CONCURRENCY,
  MAX_WORKBRIDGE_PROCESS_CONCURRENCY,
  ProcessAdmissionGate,
  resolveWorkBridgeProcessConcurrency
} from '../dist/workbridge-process-admission.js';

assert.equal(DEFAULT_WORKBRIDGE_PROCESS_CONCURRENCY, 4);
assert.equal(MAX_WORKBRIDGE_PROCESS_CONCURRENCY, 32);

const priorCapacity = process.env.WORKBRIDGE_EXECUTION_CAPACITY;
try {
  delete process.env.WORKBRIDGE_EXECUTION_CAPACITY;
  assert.equal(resolveWorkBridgeProcessConcurrency(), 4);
  process.env.WORKBRIDGE_EXECUTION_CAPACITY = '8';
  assert.equal(resolveWorkBridgeProcessConcurrency(), 8);
} finally {
  if (priorCapacity === undefined) delete process.env.WORKBRIDGE_EXECUTION_CAPACITY;
  else process.env.WORKBRIDGE_EXECUTION_CAPACITY = priorCapacity;
}

assert.equal(resolveWorkBridgeProcessConcurrency(''), 4);
assert.equal(resolveWorkBridgeProcessConcurrency('8'), 8);
assert.throws(() => new ProcessAdmissionGate(33), /between 1 and 32/);
for (const invalid of ['0', '-1', '33', '1.5', 'garbage']) {
  assert.throws(
    () => resolveWorkBridgeProcessConcurrency(invalid),
    /WORKBRIDGE_EXECUTION_CAPACITY/,
    `expected invalid capacity to fail closed: ${invalid}`
  );
}

async function assertAdmission(limit, total) {
  const gate = new ProcessAdmissionGate(limit);
  let active = 0;
  let peak = 0;
  const started = [];
  let releaseHolders;
  const holders = new Promise(resolve => { releaseHolders = resolve; });

  const jobs = Array.from({ length: total }, (_, i) => gate.run(async () => {
    active += 1;
    peak = Math.max(peak, active);
    started.push(i);
    try {
      if (i < limit) await holders;
    } finally {
      active -= 1;
    }
  }));

  await new Promise(resolve => setTimeout(resolve, 25));
  assert.equal(peak, limit, `expected ${limit} concurrent workers, observed ${peak}`);
  assert.equal(started.length, limit, `request beyond capacity started early: ${started}`);
  releaseHolders();
  await Promise.all(jobs);
  assert.equal(started.length, total, 'queued request never started after capacity released');
}

await assertAdmission(4, 5);
await assertAdmission(resolveWorkBridgeProcessConcurrency('8'), 9);

console.log('WORKBRIDGE_PROCESS_CONCURRENCY_V2 PASS');
