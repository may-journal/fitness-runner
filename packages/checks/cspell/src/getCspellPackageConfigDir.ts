import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Directory containing this package's bundled cspell.json (next to compiled output). */
export function getCspellPackageConfigDir(): string {
  return dirname(fileURLToPath(new URL('../cspell.json', import.meta.url)));
}
