import assert from 'node:assert/strict';
import { ProcessAdmissionGate } from '../dist/workbridge-process-admission.js';

const gate = new ProcessAdmissionGate(4);
let active = 0;
let peak = 0;
let fifthStarted = false;
let releaseFirst;
const firstHold = new Promise(resolve => { releaseFirst = resolve; });

const jobs = Array.from({ length: 5 }, (_, i) => gate.run(async () => {
  active += 1;
  peak = Math.max(peak, active);
  if (i === 4) fifthStarted = true;
  try {
    if (i === 0) await firstHold;
    else await new Promise(resolve => setTimeout(resolve, 75));
  } finally {
    active -= 1;
  }
}));

await new Promise(resolve => setTimeout(resolve, 25));
assert.equal(peak, 4, `expected four concurrent workers, observed ${peak}`);
assert.equal(fifthStarted, false, 'fifth process started before capacity was released');
releaseFirst();
await Promise.all(jobs);
assert.equal(fifthStarted, true, 'fifth process never started after capacity was released');

// Releasing a slot must reserve it for the oldest waiter. A newcomer arriving
// before that waiter's promise continuation must not overtake it or exceed the cap.
const handoff = new ProcessAdmissionGate(1);
const releaseOwner = await handoff.acquire();
const waitingSecond = handoff.acquire();
releaseOwner();

const admissionOrder = [];
const waitingThird = handoff.acquire().then(release => {
  admissionOrder.push('third');
  return release;
});
const releaseSecond = await waitingSecond;
admissionOrder.push('second');
await new Promise(resolve => setImmediate(resolve));
assert.deepEqual(admissionOrder, ['second'], 'newcomer barged ahead of reserved waiter');

releaseSecond();
const releaseThird = await waitingThird;
assert.deepEqual(admissionOrder, ['second', 'third'], 'waiter handoff order changed');
releaseOwner(); // Double-release must not affect the new owner.
releaseSecond();
releaseThird();

// A rejected operation must still return its slot for the next operation.
const afterFailure = new ProcessAdmissionGate(1);
await assert.rejects(afterFailure.run(async () => {
  throw new Error('intentional test failure');
}), /intentional test failure/);
const releaseAfterFailure = await afterFailure.acquire();
releaseAfterFailure();
console.log('WORKBRIDGE_PROCESS_CONCURRENCY_V1 PASS');
