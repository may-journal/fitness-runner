import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { getFitnessRunnerRoot } from './getFitnessRunnerRoot.js';

/** Path to a config file in @mayjournal/fitness when the consumer project has no local copy. */
export function resolveFitnessConfigPath(
  root: string,
  localName: string,
  fitnessRunnerRoot?: string
): string {
  const local = join(root, localName);
  if (existsSync(local)) return local;
  return join(fitnessRunnerRoot ?? getFitnessRunnerRoot(), localName);
}
