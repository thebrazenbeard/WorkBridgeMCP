export const DEFAULT_WORKBRIDGE_PROCESS_CONCURRENCY = 4;
export const MAX_WORKBRIDGE_PROCESS_CONCURRENCY = 32;

export function resolveWorkBridgeProcessConcurrency(
  raw = process.env.WORKBRIDGE_EXECUTION_CAPACITY
): number {
  if (raw === undefined || raw.trim() === '') {
    return DEFAULT_WORKBRIDGE_PROCESS_CONCURRENCY;
  }
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < 1 || value > MAX_WORKBRIDGE_PROCESS_CONCURRENCY) {
    throw new Error(
      `WORKBRIDGE_EXECUTION_CAPACITY must be an integer between 1 and ${MAX_WORKBRIDGE_PROCESS_CONCURRENCY}`
    );
  }
  return value;
}

export class ProcessAdmissionGate {
  private active = 0;
  private readonly waiters: Array<() => void> = [];

  constructor(private readonly maxConcurrent = DEFAULT_WORKBRIDGE_PROCESS_CONCURRENCY) {
    if (!Number.isInteger(maxConcurrent) || maxConcurrent < 1 || maxConcurrent > MAX_WORKBRIDGE_PROCESS_CONCURRENCY) {
      throw new Error(`maxConcurrent must be an integer between 1 and ${MAX_WORKBRIDGE_PROCESS_CONCURRENCY}`);
    }
  }

  async acquire(): Promise<() => void> {
    if (this.active >= this.maxConcurrent) {
      await new Promise<void>(resolve => this.waiters.push(resolve));
    }
    this.active += 1;
    let released = false;
    return () => {
      if (released) return;
      released = true;
      this.active -= 1;
      this.waiters.shift()?.();
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

export const workbridgeProcessAdmission = new ProcessAdmissionGate(
  resolveWorkBridgeProcessConcurrency()
);
