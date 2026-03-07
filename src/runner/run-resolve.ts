import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { registry } from '../checks/index.js';
import type { Check, RunContext } from '../types/index.types.js';
import { loadConfig } from '../config/load.js';
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

/** Resolves checks from config.checks list when present; dedupes by name so each check runs once. */
function checksFromConfigList(config: ReturnType<typeof loadConfig>): Check[] | null {
  if (!config?.checks?.length) return null;
  const byName = new Map(registry.map((c) => [c.name, c]));
  const seen = new Set<string>();
  return config.checks
    .map((name) => byName.get(name))
    .filter((c): c is Check => c != null && (seen.has(c.name) ? false : (seen.add(c.name), true)));
}

/** Resolves checks from registry, optionally excluding disabledChecks. */
function registryMinusDisabled(config: ReturnType<typeof loadConfig>): Check[] {
  const disabled = new Set(config?.disabledChecks ?? []);
  return disabled.size ? registry.filter((c) => !disabled.has(c.name)) : [...registry];
}

/** Resolves checks from config (if present) or full registry, in order. */
function resolveChecks(root: string): Check[] {
  const config = loadConfig(root);
  const fromList = checksFromConfigList(config);
  if (fromList != null) return fromList;
  return registryMinusDisabled(config);
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
    return getCheckFromModule(mod);
  } catch {
    return null;
  }
}

/** Returns checks to run for a given spec (name or path), or all from config when spec is undefined. */
async function resolveChecksBySpec(spec: string | undefined, root: string): Promise<Check[]> {
  if (!spec) return resolveChecks(root);
  if (isPathSpec(spec)) {
    const check = await loadCheckFromPath(root, spec);
    return check ? [check] : [];
  }
  const one = registry.find((r) => r.name === spec);
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

/** Builds context with registeredCheckNames, enabledCheckNames, and optional staged/inline/passthrough data. */
function buildContext(
  staged: RunContext | undefined,
  inlineFragment: RunContext | undefined,
  enabledChecks: Check[],
  passthroughArgs: string[]
): RunContext {
  return {
    enabledCheckNames: enabledChecks.map((c) => c.name),
    registeredCheckNames: registry.map((c) => c.name),
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

/** Returns checks to run, spec for error display, and optional context from argv. */
export async function getChecks(
  argv: string[],
  root: string
): Promise<{
  checks: Check[];
  context: RunContext | undefined;
  spec: string | undefined;
}> {
  const spec = resolveCheckSpec(argv);
  const checks = await resolveChecksBySpec(spec, root);
  const staged = getStagedContext();
  const argsAfterSpec = checks.length === 1 ? getArgsAfterSpec(argv, spec) : [];
  const inlineFragment =
    checks.length === 1 ? getInlineContextFragment(argsAfterSpec, checks[0]) : undefined;
  const passthrough = checks.length === 1 ? getPassthroughArgs(argsAfterSpec, checks[0]) : [];
  return { checks, context: buildContext(staged, inlineFragment, checks, passthrough), spec };
}
