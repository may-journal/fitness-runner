import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  CHANGELOG_MD as ROOT_CHANGELOG,
  PACKAGE_JSON,
  PACKAGE_LOCK_JSON as PACKAGE_LOCK,
  checkResult,
} from '@mayjournal/fitness-shared';
import { CheckName, type Check } from '@mayjournal/fitness';

const ERROR_MISSING = 'missing root CHANGELOG.md';
const DATED_SECTION_RE = /^###\s+(\d{4}\.\d{2}\.\d{2}\.\d{4})/;
const VERSION_SUFFIX_RE = /-(\d{4}\.\d{2}\.\d{2}\.\d{4})$/;

export const MSG_VERSION_CHANGELOG =
  'package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)';
export const MSG_VERSION_LOCK = 'package-lock.json version must match package.json version';

/** Returns lines that are ### headings (level-3 only). */
function getH3Lines(content: string): string[] {
  return content.split('\n').filter((line) => /^###\s/.test(line));
}

/** Extract yyyy.mm.dd.HHMM from package version (e.g. 0.1.0-2026.02.16.1900). */
function getVersionTimestamp(version: string): string | null {
  const m = version.match(VERSION_SUFFIX_RE);
  return m ? m[1] : null;
}

/** Returns format errors for CHANGELOG content, or empty array if valid. */
function getChangelogFormatErrors(content: string): string[] {
  const h3s = getH3Lines(content);
  if (h3s.length === 0) return ['CHANGELOG.md must have at least one ### yyyy.mm.dd.HHMM section'];
  const invalid = h3s.filter((line) => !DATED_SECTION_RE.test(line));
  return invalid.map(
    (line) => `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "${line.trim()}")`
  );
}

/** Parse package.json at root; returns version or throws. */
function readPackageVersion(root: string): string | undefined {
  const pkg = JSON.parse(readFileSync(join(root, PACKAGE_JSON), 'utf8')) as { version?: string };
  return pkg.version;
}

/** Returns error if package version suffix does not match changelogTs. Exported for tests. */
export function getPackageChangelogMismatch(
  pkgVersion: string | undefined,
  changelogTs: string
): string | null {
  const pkgTs = pkgVersion != null ? getVersionTimestamp(String(pkgVersion)) : null;
  if (pkgTs === changelogTs) return null;
  return MSG_VERSION_CHANGELOG;
}

/** Returns error if lock version does not match pkgVersion, or lock invalid. */
function getLockVersionError(root: string, pkgVersion: string | undefined): string | null {
  const lockPath = join(root, PACKAGE_LOCK);
  if (!existsSync(lockPath)) return null;
  try {
    const lock = JSON.parse(readFileSync(lockPath, 'utf8')) as { version?: string };
    return lock.version !== pkgVersion ? MSG_VERSION_LOCK : null;
  } catch {
    return `${PACKAGE_LOCK} is invalid JSON`;
  }
}

/** Returns version-correlation errors for root (package.json / package-lock vs changelogTs). */
function getVersionErrors(root: string, changelogTs: string): string[] {
  if (!existsSync(join(root, PACKAGE_JSON))) return [];
  let pkgVersion: string | undefined;
  try {
    pkgVersion = readPackageVersion(root);
  } catch {
    return [`${PACKAGE_JSON} is invalid JSON`];
  }
  return [
    getPackageChangelogMismatch(pkgVersion, changelogTs),
    getLockVersionError(root, pkgVersion),
  ].filter((e): e is string => e != null);
}

/** Validates repo root has CHANGELOG.md; every ### heading must be ### yyyy.mm.dd.HHMM; package.json and package-lock version must match first heading. */
export const changelogCheck: Check = {
  name: CheckName.Changelog,
  async run(root = process.cwd()) {
    const changelogPath = join(root, ROOT_CHANGELOG);
    if (!existsSync(changelogPath)) return checkResult(false, [ERROR_MISSING], 1);
    const content = readFileSync(changelogPath, 'utf8');
    const formatErrors = getChangelogFormatErrors(content);
    if (formatErrors.length > 0) return checkResult(false, formatErrors, 1);
    const h3s = getH3Lines(content);
    const firstMatch = h3s.map((line) => line.match(DATED_SECTION_RE)).find((m) => m)!;
    const versionErrors = getVersionErrors(root, firstMatch[1]);
    if (versionErrors.length > 0) return checkResult(false, versionErrors, 1);
    return checkResult(true, [], 1);
  },
};

export default changelogCheck;
