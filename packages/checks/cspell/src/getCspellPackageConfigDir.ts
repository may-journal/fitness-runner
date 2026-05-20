import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Directory containing the shared cspell config (`@mayjournal/fitness-shared/cspell`). */
export function getCspellPackageConfigDir(): string {
  const resolved = fileURLToPath(import.meta.resolve('@mayjournal/fitness-shared/cspell'));
  return dirname(resolved);
}
