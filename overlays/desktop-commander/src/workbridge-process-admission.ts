import type { ChildProcess } from 'node:child_process';
import { AsyncLocalStorage } from 'node:async_hooks';

export class ProcessAdmissionCancelledError extends Error {
  constructor() {
    super('request canceled before process admission');
    this.name = 'ProcessAdmissionCancelledError';
  }
}

export type ProcessAdmissionOptions = { signal?: AbortSignal };

// The MCP SDK supplies one signal per CallTool request. AsyncLocalStorage keeps
// concurrent tool calls isolated while the pinned upstream handlers await
// command validation and shell configuration before reaching the spawn seam.
const requestAbort = new AsyncLocalStorage<AbortSignal | undefined>();
export function runWithRequestAbort<T>(signal: AbortSignal | undefined, fn: () => Promise<T>): Promise<T> {
  return requestAbort.run(signal, fn);
}

export class ProcessAdmissionGate {
  private active = 0;
  private readonly waiters: Array<() => void> = [];

  constructor(private readonly maxConcurrent = 4) {
    if (!Number.isInteger(maxConcurrent) || maxConcurrent < 1) {
      throw new Error('maxConcurrent must be a positive integer');
    }
  }

  async acquire(options: ProcessAdmissionOptions = {}): Promise<() => void> {
    const signal = options.signal;
    if (signal?.aborted) throw new ProcessAdmissionCancelledError();
    if (this.active >= this.maxConcurrent) {
      // The released slot is reserved for this waiter before it resumes.
      await new Promise<void>((resolve, reject) => {
        const enter = () => {
          signal?.removeEventListener('abort', onAbort);
          resolve();
        };
        const onAbort = () => {
          const index = this.waiters.indexOf(enter);
          if (index < 0) return; // Already transferred to this waiter.
          this.waiters.splice(index, 1);
          signal?.removeEventListener('abort', onAbort);
          reject(new ProcessAdmissionCancelledError());
        };
        this.waiters.push(enter);
        signal?.addEventListener('abort', onAbort, { once: true });
        if (signal?.aborted) onAbort();
      });
    } else {
      this.active += 1;
    }
    // A signal can fire after the occupied slot was transferred but before
    // this async continuation resumed. Give that slot to the next waiter.
    if (signal?.aborted) {
      this.releaseOccupied();
      throw new ProcessAdmissionCancelledError();
    }
    let released = false;
    return () => {
      if (released) return;
      released = true;
      this.releaseOccupied();
    };
  }

  private releaseOccupied() {
    const next = this.waiters.shift();
    if (next) next(); // Transfer the occupied slot; newcomers cannot barge.
    else this.active -= 1;
  }

  async run<T>(fn: () => Promise<T>, options?: ProcessAdmissionOptions): Promise<T> {
    const release = await this.acquire(options);
    try {
      return await fn();
    } finally {
      release();
    }
  }
}

export const workbridgeProcessAdmission = new ProcessAdmissionGate(4);

/**
 * Keep an occupied admission slot for the life of the directly spawned child.
 * start_process may return a PID or prompt long before that child exits.
 * Descendants that outlive the direct child are outside this guarantee.
 */
export async function spawnWithProcessAdmission<T extends ChildProcess>(
  start: () => T,
  gate: ProcessAdmissionGate = workbridgeProcessAdmission,
  onProcessError?: (error: Error) => void,
  options?: ProcessAdmissionOptions
): Promise<T> {
  const signal = options?.signal ?? requestAbort.getStore();
  const release = await gate.acquire({ signal });
  if (signal?.aborted) {
    release();
    throw new ProcessAdmissionCancelledError();
  }
  let child: T;
  try {
    child = start();
  } catch (error) {
    release();
    throw error;
  }

  child.once('exit', release);
  child.once('close', release);
  // A failed spawn has no PID and may emit 'error' without emitting 'exit'.
  // An error on an already-running process (e.g. failed kill or IPC send)
  // must not release its slot before the child actually exits.
  child.on('error', (error: Error) => {
    if (!child.pid || child.exitCode !== null || child.signalCode !== null) release();
    onProcessError?.(error);
  });
  if (!child.pid) release();
  return child;
}
