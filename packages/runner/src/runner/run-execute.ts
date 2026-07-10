import { existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Worker } from 'node:worker_threads';
import { isPathLoadedCheck } from '../checks/load-check.js';
import { CheckName } from '../types/check-name.js';
import type { Check, RunContext } from '../types/index.types.js';
import { interpolate } from '../utils/interpolate.js';

/** Worker error-channel key: both the reply-message discriminant and the worker event name. */
const ERROR_KEY = 'error';
import { enUS } from './enUS.js';

const CHECK_TIMEOUT_MS = 5000;

/** Resolves check timeout ms: test override wins, then the check's own timeoutMs, then the default. */
function getCheckTimeoutMs(check: Check, context?: RunContext): number {
  return context?._checkTimeoutMsForTesting ?? check.timeoutMs ?? CHECK_TIMEOUT_MS;
}

/** Rejects after ms; used for in-process path-based checks that may hang async. */
function timeoutAfter(ms: number): Promise<never> {
  return new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), ms));
}

type WorkerReply =
  | { ms: number; result: { errors: string[]; meta?: { filesChecked?: number }; ok: boolean } }
  | { error: string; ms: number };

/** Row shape returned by runOneCheck / worker. */
export type ResultRowLike = {
  errors: string[];
  filesChecked: number;
  ms: number;
  name: string;
  ok: boolean;
};

/** Formats thrown value for in-process check (timeout vs other). */
function formatInProcessError(err: unknown, timeoutMs: number = CHECK_TIMEOUT_MS): string {
  if (err instanceof Error && err.message === 'timeout') {
    return interpolate(enUS.CheckTimeout, { seconds: timeoutMs / 1000 });
  }
  return err instanceof Error ? err.message : String(err);
}

/** Runs check in main thread (read-repo-first, vitest-coverage-full, path-based, or when VITEST). */
async function runOneCheckInProcess(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const start = performance.now();
  const timeoutMs = getCheckTimeoutMs(check, context);
  const runPromise =
    check.name === CheckName.ReadRepoFirst
      ? check.run(root, context)
      : Promise.race([
          Promise.resolve().then(() => check.run(root, context)),
          timeoutAfter(timeoutMs),
        ]);
  try {
    const result = await runPromise;
    const ms = Math.round(performance.now() - start);
    const filesChecked = result.meta?.filesChecked ?? -1;
    return { errors: result.errors, filesChecked, ms, name: check.name, ok: result.ok };
  } catch (err: unknown) {
    const ms = Math.round(performance.now() - start);
    return {
      errors: [formatInProcessError(err, timeoutMs)],
      filesChecked: -1,
      ms,
      name: check.name,
      ok: false,
    };
  }
}

/** Runs check in worker thread; main thread terminates worker after CHECK_TIMEOUT_MS. */
function runOneCheckInWorker(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const workerDir = dirname(fileURLToPath(import.meta.url));
  const workerNextToRun = resolve(workerDir, 'run-one-check-worker.js');
  const workerPath = existsSync(workerNextToRun)
    ? workerNextToRun
    : resolve(process.cwd(), 'packages/runner/dist/runner/run-one-check-worker.js');
  const worker = new Worker(workerPath, {
    type: 'module',
    workerData: { checkName: check.name, context, root },
  } as import('node:worker_threads').WorkerOptions);
  const timeoutMs = getCheckTimeoutMs(check, context);
  return new Promise((resolve) => {
    let settled = false;
    const timeoutMsg = interpolate(enUS.CheckTimeout, { seconds: timeoutMs / 1000 });
    const timeoutId = setTimeout(() => {
      /* v8 ignore start - defensive; timeout is cleared on message/error so settled is never true here */
      if (settled) return;
      /* v8 ignore stop */
      settled = true;
      worker.terminate();
      resolve({
        errors: [timeoutMsg],
        filesChecked: -1,
        ms: timeoutMs,
        name: check.name,
        ok: false,
      });
    }, timeoutMs);
    worker.on('message', (msg: WorkerReply) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutId);
      if (ERROR_KEY in msg) {
        resolve({ errors: [msg.error], filesChecked: -1, ms: msg.ms, name: check.name, ok: false });
      } else {
        const filesChecked = msg.result.meta?.filesChecked ?? -1;
        resolve({
          errors: msg.result.errors,
          filesChecked,
          ms: msg.ms,
          name: check.name,
          ok: msg.result.ok,
        });
      }
    });
    worker.on(ERROR_KEY, (err: Error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutId);
      resolve({ errors: [err.message], filesChecked: -1, ms: 0, name: check.name, ok: false });
    });
  });
}

/** Runs a single check; package checks use worker + 5s terminate; runInProcess, path-loaded, and tests run in-process. */
export async function runOneCheck(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const runInProcess =
    check.runInProcess === true || isPathLoadedCheck(check) || process.env.VITEST === 'true';
  return runInProcess
    ? runOneCheckInProcess(check, root, context)
    : runOneCheckInWorker(check, root, context);
}
