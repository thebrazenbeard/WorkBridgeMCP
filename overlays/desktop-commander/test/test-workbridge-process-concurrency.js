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
console.log('WORKBRIDGE_PROCESS_CONCURRENCY_V1 PASS');
