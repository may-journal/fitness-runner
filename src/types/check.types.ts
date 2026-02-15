import type { CheckResult } from './check-result.types.js';
import type { RunContext } from './run-context.types.js';

/** A fitness check with name and run function. Root defaults to process.cwd() when omitted. */
export type Check = {
  name: string;
  run: (root?: string, context?: RunContext) => Promise<CheckResult>;
};
