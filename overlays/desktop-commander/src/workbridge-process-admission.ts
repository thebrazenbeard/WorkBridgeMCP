import type { ChildProcess } from 'node:child_process';

export class ProcessAdmissionGate {
  private active = 0;
  private readonly waiters: Array<() => void> = [];

  constructor(private readonly maxConcurrent = 4) {
    if (!Number.isInteger(maxConcurrent) || maxConcurrent < 1) {
      throw new Error('maxConcurrent must be a positive integer');
    }
  }

  async acquire(): Promise<() => void> {
    if (this.active >= this.maxConcurrent) {
      // The released slot is reserved for this waiter before it resumes.
      await new Promise<void>(resolve => this.waiters.push(resolve));
    } else {
      this.active += 1;
    }
    let released = false;
    return () => {
      if (released) return;
      released = true;
      const next = this.waiters.shift();
      if (next) next(); // Transfer the occupied slot; newcomers cannot barge.
      else this.active -= 1;
    };
  }

  async run<T>(fn: () => Promise<T>): Promise<T> {
    const release = await this.acquire();
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
  onProcessError?: (error: Error) => void
): Promise<T> {
  const release = await gate.acquire();
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
