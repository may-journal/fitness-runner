import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { loadConfig } from '@mayjournal/fitness-shared';
import {
  loadCheck,
  markPathLoadedCheck,
  resolveCheckNames,
  tryLoadCheck,
} from '../checks/load-check.js';
import type { Check, RunContext } from '../types/index.types.js';
import { getContextInlineFromArgs } from '../utils/contextInlineFromArgs.js';

/** Staged paths under node_modules are never passed to checks. */
function stripNodeModulesFromStaged(paths: string[]): string[] {
  return paths.filter((p) => !p.includes('node_modules'));
}

/** Returns staged file paths from git for context (always attempted); excludes node_modules. */
export function getStagedContext(): RunContext | undefined {
  try {
    const out = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    const raw = out.trim() ? out.trim().split('\n') : [];
    return { stagedFiles: stripNodeModulesFromStaged(raw) };
  } catch {
    return undefined;
  }
}

/** Returns args after the check spec when running a single check; strips contextInline arg when registered. */
function getPassthroughArgs(argsAfterSpec: string[], check: Check): string[] {
  const inline = check.contextInline;
  if (!inline) return argsAfterSpec;
  const parsed = getContextInlineFromArgs(argsAfterSpec, inline.argName);
  if (!parsed) return argsAfterSpec;
  return argsAfterSpec.filter((_, i) => !parsed.stripIndices.includes(i));
}

/** Loads checks for resolved names; skips names with no installed package when allowMissing. */
async function loadChecksByNames(
  names: string[],
  root: string,
  allowMissing = false
): Promise<Check[]> {
  const checks: Check[] = [];
  for (const name of names) {
    let check: Check | null = null;
    if (allowMissing) {
      check = await tryLoadCheck(name, root);
    } else {
      check = await loadCheck(name, root);
    }
    if (check) checks.push(check);
  }
  return checks;
}

/** Resolves checks from config or bundle default list, in order. */
async function resolveChecks(root: string): Promise<Check[]> {
  const config = loadConfig(root);
  const names = await resolveCheckNames(root);
  const fromConfig = Boolean(config?.checks?.length);
  return loadChecksByNames(names, root, fromConfig);
}

/** Returns value of --check=<name-or-path> from argv if present. */
function getCheckSpecFromArg(argv: string[]): string | undefined {
  return argv.find((a) => a.startsWith('--check='))?.slice(8);
}

/** Returns first positional as check name or path. */
function getPositionalSpec(argv: string[]): string | undefined {
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  return positionals.length >= 1 ? positionals[0] : undefined;
}

/** Resolves check spec (name or path) from --check= or single positional. */
function resolveCheckSpec(argv: string[]): string | undefined {
  return getCheckSpecFromArg(argv) ?? getPositionalSpec(argv);
}

/** True if spec looks like a file path (for loading a check module). */
function isPathSpec(spec: string): boolean {
  return /[/\\]/.test(spec) || /\.(?:js|mjs|cjs|ts)$/i.test(spec);
}

/** True if value looks like a Check. */
function isCheckLike(v: unknown): v is Check {
  return !!v && typeof (v as Check).run === 'function' && (v as Check).name != null;
}

/** Returns default or first Check-like export from module. */
function getCheckFromModule(mod: { [k: string]: unknown; default?: unknown }): Check | null {
  if (isCheckLike(mod.default)) return mod.default;
  for (const v of Object.values(mod)) if (isCheckLike(v)) return v;
  return null;
}

/** Loads a Check from a module path (default or first Check-like export). */
async function loadCheckFromPath(root: string, spec: string): Promise<Check | null> {
  const abs = resolve(root, spec);
  if (!existsSync(abs)) return null;
  try {
    const url = pathToFileURL(abs).href;
    const mod = (await import(url)) as { [k: string]: unknown; default?: unknown };
    const check = getCheckFromModule(mod);
    if (check) markPathLoadedCheck(check);
    return check;
  } catch {
    return null;
  }
}

/** Returns checks to run for a given spec (name or path), or all resolved when spec is undefined. */
async function resolveChecksBySpec(spec: string | undefined, root: string): Promise<Check[]> {
  if (!spec) return resolveChecks(root);
  if (isPathSpec(spec)) {
    const check = await loadCheckFromPath(root, spec);
    return check ? [check] : [];
  }
  const one = await tryLoadCheck(spec, root);
  return one ? [one] : [];
}

/** Builds context fragment from check.contextInline and args after spec; null if none. */
function getInlineContextFragment(argsAfterSpec: string[], check: Check): RunContext | undefined {
  const inline = check.contextInline;
  if (!inline) return undefined;
  const parsed = getContextInlineFromArgs(argsAfterSpec, inline.argName);
  if (!parsed) return undefined;
  return { [inline.contextKey]: parsed.value } as RunContext;
}

/** Map of check name to folder when it differs from name (from loaded checks). */
function getCheckFolderByName(checks: Check[]): Record<string, string> {
  const map: Record<string, string> = {};
  for (const c of checks) {
    if (c.folder != null && c.folder !== c.name) map[c.name] = c.folder;
  }
  return map;
}

/** Builds context with enabled/registered names, checkFolderByName, and optional staged/inline/passthrough data. */
function buildContext(
  staged: RunContext | undefined,
  inlineFragment: RunContext | undefined,
  enabledChecks: Check[],
  passthroughArgs: string[]
): RunContext {
  const names = enabledChecks.map((c) => c.name);
  return {
    checkFolderByName: getCheckFolderByName(enabledChecks),
    enabledCheckNames: names,
    registeredCheckNames: names,
    ...(staged ?? {}),
    ...(inlineFragment ?? {}),
    ...(passthroughArgs.length > 0 ? { passthroughArgs } : {}),
  };
}

/** Returns args after the check spec (for single-check run). */
function getArgsAfterSpec(argv: string[], spec: string | undefined): string[] {
  const specFromPositional = spec !== undefined && getPositionalSpec(argv) === spec;
  return specFromPositional
    ? argv.slice(3)
    : argv.slice(2).filter((a) => !a.startsWith('--check='));
}

/** Formats a resolution failure for display. */
function resolutionErrorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/** Resolves checks and builds run context when resolution succeeds. */
async function resolveChecksSuccess(
  argv: string[],
  spec: string | undefined,
  root: string
): Promise<{ checks: Check[]; context: RunContext | undefined; spec: string | undefined }> {
  const checks = await resolveChecksBySpec(spec, root);
  const staged = getStagedContext();
  const argsAfterSpec = checks.length === 1 ? getArgsAfterSpec(argv, spec) : [];
  const inlineFragment =
    checks.length === 1 ? getInlineContextFragment(argsAfterSpec, checks[0]) : undefined;
  const passthrough = checks.length === 1 ? getPassthroughArgs(argsAfterSpec, checks[0]) : [];
  return { checks, context: buildContext(staged, inlineFragment, checks, passthrough), spec };
}

/** Returns checks to run, spec for error display, and optional context from argv. */
export async function getChecks(
  argv: string[],
  root: string
): Promise<{
  checks: Check[];
  context: RunContext | undefined;
  resolutionError?: string;
  spec: string | undefined;
}> {
  const spec = resolveCheckSpec(argv);
  try {
    return await resolveChecksSuccess(argv, spec, root);
  } catch (err) {
    return { checks: [], context: undefined, resolutionError: resolutionErrorMessage(err), spec };
  }
}
