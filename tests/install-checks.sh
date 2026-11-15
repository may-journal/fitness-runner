#!/bin/sh
# Prints the folder of checks the requirement tests run. The checks ship from
# fitness-checks, so the candidate installer at FITNESS_INSTALL fetches each
# one's newest v1 release, as a repo without checkVersions does.
set -eu
: "${FITNESS_INSTALL:?set FITNESS_INSTALL to the candidate fitness-install}"
pins="$(mktemp -d)"
checks="$(cd "$pins" && "$FITNESS_INSTALL" --with-checks --install-only | sed -n 2p)"
rm -rf "$pins"
if [ -z "$checks" ]; then
  echo "tests/install-checks.sh: no checks installed from fitness-checks" >&2
  exit 1
fi
echo "$checks"
