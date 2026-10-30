import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { EventEmitter } from 'node:events';
import { fileURLToPath } from 'node:url';

// The installer must test its compiled overlay. Developers can opt into source
// mode explicitly on Node 24 with --experimental-transform-types.
const distUrl = new URL('../dist/workbridge-process-admission.js', import.meta.url);
const sourceUrl = new URL('../src/workbridge-process-admission.ts', import.meta.url);
const sourceMode = process.env.WORKBRIDGE_TEST_SOURCE === '1';
assert.ok(sourceMode || existsSync(fileURLToPath(distUrl)),
  'compiled overlay missing: run npm build before the installer regression');
const admissionModule = await import(sourceMode ? sourceUrl : distUrl);
const { ProcessAdmissionGate, spawnWithProcessAdmission, runWithRequestAbort,
  bindProcessAdmissionToStdioDisconnect } = admissionModule;

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

// An MCP request aborted while queued must never acquire a permit later and
// launch a stale command when an unrelated child finally exits.
const cancellationGate = new ProcessAdmissionGate(1);
const releaseCancellationOwner = await cancellationGate.acquire();
const cancellation = new AbortController();
const canceledWaiter = cancellationGate.acquire({ signal: cancellation.signal });
await delay(10);
cancellation.abort();
try {
  await assert.rejects(within(canceledWaiter, 300,
    'aborted queued admission was not removed'), error => error?.name === 'ProcessAdmissionCancelledError');
} finally {
  releaseCancellationOwner();
}
const releaseAfterCancellation = await within(cancellationGate.acquire(), 1000,
  'canceled queued request retained a permit');
releaseAfterCancellation();

const cancellationFairness = new ProcessAdmissionGate(1);
const releaseFairnessOwner = await cancellationFairness.acquire();
const canceledHead = new AbortController();
const oldHead = cancellationFairness.acquire({ signal: canceledHead.signal });
const survivingWaiter = cancellationFairness.acquire();
canceledHead.abort();
await assert.rejects(oldHead, error => error?.name === 'ProcessAdmissionCancelledError');
releaseFairnessOwner();
const releaseSurvivor = await within(survivingWaiter, 1000,
  'aborted first waiter blocked the next FIFO waiter');
releaseSurvivor();

// Abort can race the transfer itself: the promise was resolved as a slot
// holder, but its continuation has not yet run. That slot must be reclaimed.
const transferGate = new ProcessAdmissionGate(1);
const releaseTransferOwner = await transferGate.acquire();
const transferAbort = new AbortController();
const transferred = transferGate.acquire({ signal: transferAbort.signal });
releaseTransferOwner();
transferAbort.abort();
await assert.rejects(transferred,
  error => error?.name === 'ProcessAdmissionCancelledError');
const releaseAfterTransferAbort = await within(transferGate.acquire(), 1000,
  'abort after permit transfer leaked occupied slot');
releaseAfterTransferAbort();

// The MCP server binds SDK extra.signal around its tool handler. A queued
// spawn must inherit that request-local signal even across async awaits.
const contextualGate = new ProcessAdmissionGate(1);
const releaseContextOwner = await contextualGate.acquire();
const contextualCancellation = new AbortController();
let contextualSpawned = false;
const contextualJob = runWithRequestAbort(contextualCancellation.signal, async () => {
  await delay(5);
  return spawnWithProcessAdmission(() => {
    contextualSpawned = true;
    return spawn(process.execPath, ['-e', 'process.exit(0)']);
  }, contextualGate);
});
await delay(25);
contextualCancellation.abort();
try {
  await assert.rejects(within(contextualJob, 1000, 'contextual abort did not settle'),
    error => error?.name === 'ProcessAdmissionCancelledError');
} finally {
  releaseContextOwner();
}
assert.equal(contextualSpawned, false, 'abandoned contextual start_process spawned after cancellation');
const releaseContextNext = await within(contextualGate.acquire(), 1000,
  'contextual cancellation leaked admission');
releaseContextNext();

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
let pendingFifthChild = null;
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

  pendingFifthChild = spawnWithProcessAdmission(startPersistentChild, liveChildren).then(child => {
    fifthChildStarted = true;
    return child;
  });
  await delay(75);
  assert.equal(fifthChildStarted, false, 'fifth persistent child spawned before an active child exited');

  const firstExit = new Promise(resolve => firstFour[0].once('exit', resolve));
  assert.equal(firstFour[0].kill(), true, 'first child should accept termination');
  await within(firstExit, 5000, 'first child did not exit');
  const fifthChild = await within(pendingFifthChild, 5000, 'fifth child was not admitted after first exited');
  assert.equal(fifthChildStarted, true);
  assert.ok(fifthChild.pid && fifthChild.exitCode === null);
} finally {
  // Killing the first four releases a permit and may START the queued fifth.
  // Wait for that admission, then repeat cleanup against a fresh child list.
  const terminateKnown = () => Promise.all(children.map(async child => {
    if (child.exitCode !== null || child.signalCode !== null) return;
    const exited = new Promise(resolve => child.once('exit', resolve));
    child.kill();
    await within(exited, 5000, 'test child did not terminate');
  }));
  await terminateKnown();
  if (pendingFifthChild) {
    await within(pendingFifthChild.catch(() => null), 5000,
      'queued child did not settle during test cleanup');
  }
  await terminateKnown();
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
  let pendingFifthResult = null;
  try {
    const firstFour = await within(Promise.all(Array.from({ length: 4 }, () =>
      manager.executeCommand(persistentCommand, 5000, shell))),
    15000, 'four start_process calls did not return their initial PID');
    assert.ok(firstFour.every(result => result.pid > 0 && manager.getSession(result.pid)),
      'four prompt-returned PIDs must still have active sessions');

    let fifthReturned = false;
    pendingFifthResult = manager.executeCommand(persistentCommand, 5000, shell).then(result => {
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
    const fifthResult = await within(pendingFifthResult, 10000, 'fifth start_process failed to obtain released slot');
    assert.ok(fifthResult.pid > 0 && manager.getSession(fifthResult.pid));
  } finally {
    const terminateActive = () => Promise.all(manager.listActiveSessions().map(async ({ pid }) => {
      const session = manager.getSession(pid);
      if (!session) return;
      const exited = new Promise(resolve => session.process.once('exit', resolve));
      session.process.kill();
      await within(exited, 5000, `start_process child ${pid} did not terminate`);
    }));
    await terminateActive();
    if (pendingFifthResult) {
      await within(pendingFifthResult.catch(() => null), 10000,
        'queued start_process did not settle during test cleanup');
    }
    await terminateActive();
  }
}
// SDK v1.25.1 does not abort extra.signal on transport close, and its stdio
// transport does not emit onclose when stdin reaches EOF. The overlay must
// explicitly cancel all not-yet-admitted processes at either boundary.
const eofGate = new ProcessAdmissionGate(1);
const releaseEofOwner = await eofGate.acquire();
const input = new EventEmitter();
let originalCloseCount = 0;
const sdkServer = { onclose: () => { originalCloseCount += 1; } };
bindProcessAdmissionToStdioDisconnect(sdkServer, input);
let spawnedAfterEof = false;
const queuedAfterEof = runWithRequestAbort(undefined, async () => {
  await delay(5);
  return spawnWithProcessAdmission(() => {
    spawnedAfterEof = true;
    return spawn(process.execPath, ['-e', 'process.exit(0)']);
  }, eofGate);
});
await delay(20);
input.emit('end');
await assert.rejects(within(queuedAfterEof, 1000, 'stdin EOF left stale admission queued'),
  error => error?.name === 'ProcessAdmissionCancelledError');
releaseEofOwner();
assert.equal(spawnedAfterEof, false, 'a fifth child started after stdin EOF');
// Close events can follow EOF; wiring must be idempotent and preserve the SDK
// server's prior onclose handler.
input.emit('close');
sdkServer.onclose();
assert.equal(originalCloseCount, 1, 'the previous server onclose handler was lost');
console.log('WORKBRIDGE_PROCESS_CONCURRENCY_V1 PASS');
