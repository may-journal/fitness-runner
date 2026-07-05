import {
  checkResult,
  execSyncResult,
  findFilesByExtension,
  getExecSync,
} from '@mayjournal/fitness-shared';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import type { Check, CheckName, RunContext } from '@mayjournal/fitness';
import { enUS } from './enUS.js';

/** One `npm outdated --json` record (npm lists a dependency only when it is behind). */
interface OutdatedEntry {
  current?: string;
  dependent?: string;
  latest?: string;
  location?: string;
  wanted?: string;
}

/** npm outdated keys a record per dependency, or an array when it is outdated in several places. */
type OutdatedReport = Record<string, OutdatedEntry | OutdatedEntry[]>;

// Direct declared deps only (no --all): you can only bump what you declare. Exits 1 when any are
// behind — expected, not an error. 2>&1 so a non-JSON npm error is captured for graceful handling.
const NPM_OUTDATED = 'npm outdated --json 2>&1';

/** Internal workspace packages are versioned in-repo, never "outdated" against the registry. */
const INTERNAL_SCOPE = /^@mayjournal\//;

/** Parses `npm outdated --json`; null when output is not a JSON object (e.g. registry unreachable). */
export function parseOutdated(output: string): OutdatedReport | null {
  const trimmed = output.trim();
  if (!trimmed) return {};
  try {
    const data: unknown = JSON.parse(trimmed);
    return data !== null && typeof data === 'object' ? (data as OutdatedReport) : null;
  } catch {
    return null;
  }
}

/** "name: current → latest" when a record is behind its latest published version, else null. */
function entryError(name: string, entry: OutdatedEntry): string | null {
  if (!entry.latest || entry.current === entry.latest) return null;
  return `${name}: ${entry.current ?? 'missing'} → ${entry.latest}`;
}

/** Behind-latest messages for one dependency (npm may report it as a single record or an array). */
function packageErrors(name: string, raw: OutdatedEntry | OutdatedEntry[]): string[] {
  const records = Array.isArray(raw) ? raw : [raw];
  return records
    .map((entry) => entryError(name, entry))
    .filter((line): line is string => line != null);
}

/**
 * Sorted, de-duplicated "name: current → latest" lines for every non-internal dependency behind
 * latest. Identical lines (the same dep at the same version across several workspaces) collapse to
 * one — bumping it is a single action.
 */
export function outdatedErrors(report: OutdatedReport): string[] {
  const errors = new Set<string>();
  for (const [name, raw] of Object.entries(report)) {
    if (INTERNAL_SCOPE.test(name)) continue;
    for (const line of packageErrors(name, raw)) errors.add(line);
  }
  return [...errors].sort();
}

/** Counts package.json manifests under root (node_modules and other skip dirs excluded). */
async function countManifests(root: string): Promise<number> {
  const files = await findFilesByExtension(root, 'package.json');
  return files.length;
}

/** Runs `npm outdated --json`; returns combined output (exit code ignored — 1 just means "found"). */
function runOutdated(root: string, execSyncFn: ExecSyncFn): string {
  return execSyncResult(root, NPM_OUTDATED, execSyncFn).output;
}

/**
 * Fails when any declared dependency is behind its latest published version — an installable
 * update means the project isn't at peak fitness. Internal `@mayjournal/*` workspace packages are
 * skipped, and an unreachable registry degrades to a pass so offline/CI runs aren't hard-blocked.
 */
export const dependencyCurrencyCheck = {
  name: 'dependency-currency' as CheckName,
  async run(root = process.cwd(), context?: RunContext) {
    const output = runOutdated(root, getExecSync(context));
    const filesChecked = await countManifests(root);
    const report = parseOutdated(output);
    if (report === null) return checkResult(true, [], filesChecked); // registry unreachable — don't block
    const outdated = outdatedErrors(report);
    const errors = outdated.length > 0 ? [enUS.Lead, ...outdated] : [];
    return checkResult(outdated.length === 0, errors, filesChecked);
  },
  runInProcess: true,
} satisfies Check;

export default dependencyCurrencyCheck;
