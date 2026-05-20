import assert from 'node:assert/strict';
import { execSync } from 'node:child_process';
import { readFileSync, statSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, it } from 'node:test';

const dir = dirname(fileURLToPath(import.meta.url));
const scriptPath = join(dir, 'nvm-use.sh');

describe('nvm-use.sh', () => {
  it('exists', () => {
    statSync(scriptPath);
  });

  it('is a valid bash script (shebang and syntax)', () => {
    const content = readFileSync(scriptPath, 'utf8');
    assert.match(content, /^#!\/usr\/bin\/env bash\n/);
    execSync(`bash -n ${JSON.stringify(scriptPath)}`, { stdio: 'pipe' });
  });

  it('sets NVM_DIR and sources nvm.sh', () => {
    const content = readFileSync(scriptPath, 'utf8');
    assert.match(content, /NVM_DIR/);
    assert.match(content, /"\$NVM_DIR\/nvm\.sh"/);
    assert.match(content, /\.\s+"\$NVM_DIR\/nvm\.sh"/);
  });

  it('calls nvm use', () => {
    const content = readFileSync(scriptPath, 'utf8');
    assert.match(content, /\bnvm use\b/);
  });

  it('sources mock nvm and runs nvm use', () => {
    const tmp = mkdtempSync(join(tmpdir(), 'nvm-use-test-'));
    const marker = join(tmp, 'nvm-use-called');
    writeFileSync(
      join(tmp, 'nvm.sh'),
      `nvm() {
  if [ "$1" = "use" ]; then
    echo called > ${JSON.stringify(marker)}
  fi
}
`
    );
    try {
      execSync(`bash ${JSON.stringify(scriptPath)}`, {
        env: { ...process.env, HOME: tmp, NVM_DIR: tmp },
        stdio: 'pipe',
      });
      assert.equal(readFileSync(marker, 'utf8').trim(), 'called');
    } finally {
      rmSync(tmp, { force: true, recursive: true });
    }
  });
});
