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
