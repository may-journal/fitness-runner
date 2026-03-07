import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

/** True if suffix is empty or a single extension (e.g. .js). */
function isExtensionSuffix(suffix: string): boolean {
  return suffix === '' || /^\.[a-z0-9]+$/i.test(suffix);
}

/** True if the module with the given import.meta.url is the Node process entry. argv[1] may omit extension (node foo → foo.js); ESM path has extension. See https://2ality.com/2022/07/nodejs-esm-main.html */
export function isMainModule(importMetaUrl: string): boolean {
  const argvPath = resolve(process.argv[1] ?? '');
  const modulePath = resolve(fileURLToPath(importMetaUrl));
  const suffix = modulePath.startsWith(argvPath) ? modulePath.slice(argvPath.length) : null;
  return modulePath === argvPath || (suffix !== null && isExtensionSuffix(suffix));
}
