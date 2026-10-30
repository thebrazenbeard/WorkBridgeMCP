import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

// The installer must test its compiled overlay. Developers can opt into source
// mode explicitly on Node 24 with --experimental-transform-types.
const distUrl = new URL('../dist/workbridge-process-admission.js', import.meta.url);
const sourceUrl = new URL('../src/workbridge-process-admission.ts', import.meta.url);
const sourceMode = process.env.WORKBRIDGE_TEST_SOURCE === '1';
assert.ok(sourceMode || existsSync(fileURLToPath(distUrl)),
  'compiled overlay missing: run npm build before the installer regression');
const admissionModule = await import(sourceMode ? sourceUrl : distUrl);
const { ProcessAdmissionGate, spawnWithProcessAdmission } = admissionModule;

const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
async function within(promise, timeoutMs, message) {
  let timer;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timer = setTimeout(() => reject(new Error(message)), timeoutMs);
      })
    ]);
  } finally {
    clearTimeout(timer);
  }
}

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

// A process can return its initial prompt/PID while remaining alive. Admission
// must be held until that child ends, not until start_process returns.
const children = [];
const liveChildren = new ProcessAdmissionGate(4);
let fifthChildStarted = false;
function startPersistentChild() {
  const child = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {
    stdio: ['pipe', 'pipe', 'pipe'],
    windowsHide: true
  });
  children.push(child);
  return child;
}

try {
  const firstFour = await Promise.all(Array.from({ length: 4 }, () =>
    spawnWithProcessAdmission(startPersistentChild, liveChildren)));
  assert.ok(firstFour.every(child => child.pid && child.exitCode === null),
    'four directly spawned children must remain alive');

  const fifth = spawnWithProcessAdmission(startPersistentChild, liveChildren).then(child => {
    fifthChildStarted = true;
    return child;
  });
  await delay(75);
  assert.equal(fifthChildStarted, false, 'fifth persistent child spawned before an active child exited');

  const firstExit = new Promise(resolve => firstFour[0].once('exit', resolve));
  assert.equal(firstFour[0].kill(), true, 'first child should accept termination');
  await within(firstExit, 5000, 'first child did not exit');
  const fifthChild = await within(fifth, 5000, 'fifth child was not admitted after first exited');
  assert.equal(fifthChildStarted, true);
  assert.ok(fifthChild.pid && fifthChild.exitCode === null);
} finally {
  // This regression must never leave children running on a developer or CI host.
  await Promise.all(children.map(async child => {
    if (child.exitCode !== null || child.signalCode !== null) return;
    const exited = new Promise(resolve => child.once('exit', resolve));
    child.kill();
    await within(exited, 5000, 'test child did not terminate');
  }));
}

// A synchronous start failure must return its permit immediately.
const failedStart = new ProcessAdmissionGate(1);
await assert.rejects(
  spawnWithProcessAdmission(() => { throw new Error('intentional spawn failure'); }, failedStart),
  /intentional spawn failure/
);
const releaseAfterSpawnFailure = await within(failedStart.acquire(), 1000, 'spawn failure leaked admission slot');
releaseAfterSpawnFailure();

// Node reports a missing executable asynchronously and without a PID.
const missingExecutable = new ProcessAdmissionGate(1);
const missingChild = await spawnWithProcessAdmission(() =>
  spawn('__workbridge_nonexistent_executable_20261030__', [], { shell: false }), missingExecutable);
missingChild.on('error', () => {});
assert.equal(missingChild.pid, undefined);
const releaseAfterMissingExecutable = await within(missingExecutable.acquire(), 1000,
  'failed asynchronous spawn leaked admission slot');
releaseAfterMissingExecutable();

if (!sourceMode) {
  // Exercise the actual start_process backend after the installer compiles the
  // overlay. These commands keep one direct shell PID alive without descendants.
  const { TerminalManager } = await import('../dist/terminal-manager.js');
  const manager = new TerminalManager();
  const shell = process.platform === 'win32' ? 'powershell.exe' : '/bin/bash';
  const persistentCommand = process.platform === 'win32'
    ? "Write-Output 'READY>'; while ($true) { Start-Sleep -Milliseconds 200 }"
    : "printf 'READY>\\n'; while :; do read -t 1 value || :; done";
  try {
    const firstFour = await within(Promise.all(Array.from({ length: 4 }, () =>
      manager.executeCommand(persistentCommand, 5000, shell))),
    15000, 'four start_process calls did not return their initial PID');
    assert.ok(firstFour.every(result => result.pid > 0 && manager.getSession(result.pid)),
      'four prompt-returned PIDs must still have active sessions');

    let fifthReturned = false;
    const fifth = manager.executeCommand(persistentCommand, 5000, shell).then(result => {
      fifthReturned = true;
      return result;
    });
    await delay(150);
    assert.equal(manager.listActiveSessions().length, 4,
      'fifth start_process spawned while four direct shell PIDs remained active');
    assert.equal(fifthReturned, false,
      'fifth start_process returned while four direct shell PIDs remained active');

    const first = manager.getSession(firstFour[0].pid);
    const firstExit = new Promise(resolve => first.process.once('exit', resolve));
    assert.equal(first.process.kill(), true);
    await within(firstExit, 5000, 'first start_process child did not exit');
    const fifthResult = await within(fifth, 10000, 'fifth start_process failed to obtain released slot');
    assert.ok(fifthResult.pid > 0 && manager.getSession(fifthResult.pid));
  } finally {
    await Promise.all(manager.listActiveSessions().map(async ({ pid }) => {
      const session = manager.getSession(pid);
      if (!session) return;
      const exited = new Promise(resolve => session.process.once('exit', resolve));
      session.process.kill();
      await within(exited, 5000, `start_process child ${pid} did not terminate`);
    }));
  }
}
console.log('WORKBRIDGE_PROCESS_CONCURRENCY_V1 PASS');
