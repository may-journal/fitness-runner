import { parentPort, workerData } from 'node:worker_threads';
import { registry } from '../checks/index.js';
import type { CheckResult } from '../types/check-result.types.js';
import type { RunContext } from '../types/index.types.js';

type WorkerPayload = { checkName: string; context?: RunContext; root: string };

type WorkerReply = { ms: number; result: CheckResult } | { error: string; ms: number };

/** Posts error reply to parent. */
function sendError(msg: string, ms: number): void {
  parentPort?.postMessage({ error: msg, ms } satisfies WorkerReply);
}

/** Posts success reply to parent. */
function sendResult(result: CheckResult, ms: number): void {
  parentPort?.postMessage({ ms, result } satisfies WorkerReply);
}

/** Worker entry: run one registry check and post result or error. */
async function run(): Promise<void> {
  const { checkName, context, root } = workerData as WorkerPayload;
  const start = performance.now();
  const check = registry.find((c) => c.name === checkName);
  if (!check) {
    sendError(`Unknown check: ${checkName}`, 0);
    return;
  }
  try {
    const result = await check.run(root, context);
    sendResult(result, Math.round(performance.now() - start));
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    sendError(msg, Math.round(performance.now() - start));
  }
}

run();
