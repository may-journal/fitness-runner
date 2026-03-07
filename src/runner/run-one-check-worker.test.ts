import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const postMessage = vi.fn();
const workerDataRef: { checkName: string; root: string; context?: unknown } = {
  checkName: '',
  root: '',
};
vi.mock('node:worker_threads', () => ({
  parentPort: { postMessage },
  get workerData() {
    return workerDataRef;
  },
}));

const registryRef: {
  name: string;
  run: (
    root: string,
    context?: unknown
  ) => Promise<{ ok: boolean; errors: string[]; meta?: unknown }>;
}[] = [];
vi.mock('../checks/index.js', () => ({ registry: registryRef }));

describe('run-one-check-worker', () => {
  beforeEach(() => {
    postMessage.mockClear();
    workerDataRef.checkName = '';
    workerDataRef.root = '';
    registryRef.length = 0;
  });

  afterEach(() => {
    vi.resetModules();
  });

  it('posts error when checkName is not in registry', async () => {
    workerDataRef.checkName = 'missing';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    expect(postMessage).toHaveBeenCalledWith({ error: 'Unknown check: missing', ms: 0 });
  });

  it('posts result when check runs successfully', async () => {
    const result = { ok: true, errors: [] as string[], meta: { filesChecked: 1 } };
    registryRef.push({ name: 'fake', run: vi.fn().mockResolvedValue(result) });
    workerDataRef.checkName = 'fake';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    const [payload] = postMessage.mock.calls[0];
    expect(payload).toMatchObject({ ms: expect.any(Number), result });
  });

  it('posts error when check.run throws', async () => {
    registryRef.push({
      name: 'throws',
      run: vi.fn().mockRejectedValue(new Error('check failed')),
    });
    workerDataRef.checkName = 'throws';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    const [payload] = postMessage.mock.calls[0];
    expect(payload).toMatchObject({ error: 'check failed', ms: expect.any(Number) });
  });

  it('posts error string when check.run throws non-Error', async () => {
    registryRef.push({
      name: 'throws',
      run: vi.fn().mockRejectedValue('string error'),
    });
    workerDataRef.checkName = 'throws';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    expect(postMessage).toHaveBeenCalledWith({ error: 'string error', ms: expect.any(Number) });
  });
});
