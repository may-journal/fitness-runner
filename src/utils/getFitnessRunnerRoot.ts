import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Resolves the bundled fitness config directory (@mayjournal/fitness-shared/config). */
export function getFitnessRunnerRoot(): string {
  return dirname(fileURLToPath(import.meta.resolve('@mayjournal/fitness-shared/cspell')));
}
