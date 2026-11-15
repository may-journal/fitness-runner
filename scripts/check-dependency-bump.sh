#!/usr/bin/env bash
# Passes a build(deps) PR that moves only dev packages by a patch or minor bump, and writes the commit it checked as sha.
# cspell:ignore slurpfile
set -e
view="$(gh pr view "$PR" --json title,state,isDraft,headRefOid,baseRefOid)"
if [[ "$(jq -r .title <<< "$view")" != 'build(deps)'* ]]; then
  echo "::notice::PR #$PR is not titled build(deps), so a person approves it."
  exit 0
fi
if [ "$(jq -r '.state + " " + (.isDraft | tostring)' <<< "$view")" != "OPEN false" ]; then
  echo "::notice::PR #$PR is not open and ready, so it gets no approval."
  exit 0
fi
files="$(gh api "repos/$GITHUB_REPOSITORY/pulls/$PR/files" --paginate |
  jq -rs 'add | .[] | .filename, (.previous_filename // empty)')"
others="$(grep -vxF -e package.json -e package-lock.json <<< "$files" || true)"
if [ -z "$files" ] || [ -n "$others" ]; then
  echo "::notice::PR #$PR changes more than package.json and package-lock.json, so a person approves it: ${others//$'\n'/, }"
  exit 0
fi
# The base side is the merge base, as the file list is. A file
# missing on one side reads as {}; any other error fails.
head="$(jq -r .headRefOid <<< "$view")"
base="$(gh api "repos/$GITHUB_REPOSITORY/compare/$(jq -r .baseRefOid <<< "$view")...$head" --jq .merge_base_commit.sha)"
dir="$(mktemp -d)"
for side_sha in "base:$base" "head:$head"; do
  side="${side_sha%%:*}"
  sha="${side_sha#*:}"
  for file in package.json package-lock.json; do
    if ! gh api -H 'Accept: application/vnd.github.raw+json' "repos/$GITHUB_REPOSITORY/contents/$file?ref=$sha" \
      > "$dir/$side-$file" 2> "$dir/error"; then
      if ! grep -q 'HTTP 404' "$dir/error"; then
        echo "::error::Could not read $file at $sha: $(cat "$dir/error")"
        exit 1
      fi
      echo '{}' > "$dir/$side-$file"
    fi
  done
done
problems="$(jq -rn --slurpfile bm "$dir/base-package.json" --slurpfile hm "$dir/head-package.json" \
  --slurpfile bl "$dir/base-package-lock.json" --slurpfile hl "$dir/head-package-lock.json" '
  # A 0.x version breaks on a minor bump, as semver allows.
  def line: if type == "string" and test("^[^0-9]*[0-9]+")
    then capture("^[^0-9]*(?<major>[0-9]+)(\\.(?<minor>[0-9]+))?") | if .major == "0" then "0.\(.minor)" else .major end
    else null end;
  def header: if has("packages") then del(.packages, .dependencies) else . end;
  def bump($name; $from; $to):
    if ($from | line) == null or ($from | line) != ($to | line)
    then "\($name) moves from \($from) to \($to), not a patch or minor bump" else empty end;
  $bm[0] as $b | $hm[0] as $h | ($b.devDependencies // {}) as $bd | ($h.devDependencies // {}) as $hd |
  ($bl[0].packages // {}) as $bp | ($hl[0].packages // {}) as $hp |
  (if ($b | del(.devDependencies)) != ($h | del(.devDependencies)) then "package.json changes more than devDependencies" else empty end),
  (if ($bd | keys) != ($hd | keys) then "package.json adds or removes a dev dependency" else empty end),
  ($bd | to_entries[] | select(.key as $k | $hd | has($k)) | select(.value != $hd[.key]) | bump(.key; .value; $hd[.key])),
  (if ($bl[0] | header) != ($hl[0] | header) then "package-lock.json changes more than its packages" else empty end),
  (if ($bp[""] | del(.devDependencies)) != ($hp[""] | del(.devDependencies)) then "package-lock.json changes the root package" else empty end),
  (($bp + $hp) | keys[] | select(. != "") | . as $k | select($bp[$k] != $hp[$k]) |
    if [$bp[$k], $hp[$k]] | map(select(. != null)) | all(.dev == true) | not then "\($k) is not a dev-only package"
    elif $hp[$k] != null and ($hp[$k].resolved // "https://registry.npmjs.org/" | startswith("https://registry.npmjs.org/") | not) then "\($k) resolves outside the npm registry"
    elif $bp[$k] != null and $hp[$k] != null and $bp[$k].version == $hp[$k].version then "\($k) changes without a new version"
    elif $bp[$k] != null and $hp[$k] != null then bump($k; $bp[$k].version; $hp[$k].version)
    else empty end)')"
if [ -n "$problems" ]; then
  echo "::notice::PR #$PR is not a patch or minor dev-dependency bump, so a person approves it: ${problems//$'\n'/; }"
  exit 0
fi
printf 'sha=%s\n' "$head" >> "$GITHUB_OUTPUT"
