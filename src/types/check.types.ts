import type { CheckResult } from './check-result.types.js';
import type { RunContext } from './run-context.types.js';
import type { CheckName } from './check-name.js';

/** When set, the runner injects the value of this argv arg into context[contextKey] and strips it from passthrough. */
export type ContextInline = { argName: string; contextKey: keyof RunContext };

/** A fitness check with name and run function. Root defaults to process.cwd() when omitted. */
export type Check = {
  /** Optional: inject a named argv arg value into context and strip from passthrough. */
  contextInline?: ContextInline;
  /** Optional: folder under src/checks when it differs from name (convention: folder = name). */
  folder?: string;
  name: CheckName;
  run: (root?: string, context?: RunContext) => Promise<CheckResult>;
  /** When true, run in main thread; otherwise run in worker (when not in tests). */
  runInProcess?: boolean;
};
