# verdict.sh: what every shell check in this fitness folder shares.
#
# Usage: a check sources it first, from the repo root, which is where
# fitness runs every check:
#
#   . fitness/_lib/verdict.sh
#   for f in $(git ls-files 'docs/*.md'); do judged; grep -q x "$f" || fail "$f: add x"; done
#   finish
#
# Inputs: the check's arguments. A --describe handshake is answered with no
# metadata, so the check's README gives its summary.
# Outputs: finish prints the check protocol's JSON result on stdout and exits
# 0 when no fail was called, else 1. A message holds no double quote or
# backslash, since it goes into JSON as it is.

if [ "${1:-}" = --describe ]; then
  echo '{}'
  exit 0
fi

errors=""
files=0

# judged counts one more file the check read.
judged() {
  files=$((files + 1))
}

# fail records one problem, named for a person to fix.
fail() {
  errors="$errors\"$1\","
}

# finish prints the verdict and exits with its code.
finish() {
  if [ -z "$errors" ]; then
    printf '{"ok":true,"errors":[],"filesChecked":%s}\n' "$files"
    exit 0
  fi
  printf '{"ok":false,"errors":[%s],"filesChecked":%s}\n' "${errors%,}" "$files"
  exit 1
}
