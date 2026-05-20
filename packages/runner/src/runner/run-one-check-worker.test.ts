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

const loadCheckMock = vi.fn();
vi.mock('../checks/load-check.js', () => ({
  loadCheck: (...args: unknown[]) => loadCheckMock(...args),
}));

describe('run-one-check-worker', () => {
  beforeEach(() => {
    postMessage.mockClear();
    loadCheckMock.mockReset();
    workerDataRef.checkName = '';
    workerDataRef.root = '';
  });

  afterEach(() => {
    vi.resetModules();
  });

  it('posts error when loadCheck fails with non-Error', async () => {
    loadCheckMock.mockRejectedValue('missing package');
    workerDataRef.checkName = 'missing';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    expect(postMessage).toHaveBeenCalledWith({ error: 'missing package', ms: 0 });
  });

  it('posts error when loadCheck fails', async () => {
    loadCheckMock.mockRejectedValue(new Error('Check package not installed'));
    workerDataRef.checkName = 'missing';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    expect(postMessage).toHaveBeenCalledWith({
      error: 'Check package not installed',
      ms: 0,
    });
  });

  it('posts result when check runs successfully', async () => {
    const result = { ok: true, errors: [] as string[], meta: { filesChecked: 1 } };
    loadCheckMock.mockResolvedValue({ name: 'fake', run: vi.fn().mockResolvedValue(result) });
    workerDataRef.checkName = 'fake';
    workerDataRef.root = '/tmp';
    await import('./run-one-check-worker.js');
    await vi.waitFor(() => expect(postMessage).toHaveBeenCalledTimes(1));
    const [payload] = postMessage.mock.calls[0];
    expect(payload).toMatchObject({ ms: expect.any(Number), result });
  });

  it('posts error when check.run throws', async () => {
    loadCheckMock.mockResolvedValue({
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
    loadCheckMock.mockResolvedValue({
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
