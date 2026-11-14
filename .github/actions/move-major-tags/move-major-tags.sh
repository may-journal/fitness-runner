#!/usr/bin/env bash
# Moves each released component's major tag, v1 for the root and a component's own for a component.
# Input is Release Please's outputs as JSON in OUTPUTS, with the App token in GH_TOKEN, and releases goes to GITHUB_OUTPUT.
set -euo pipefail

refs="repos/$GITHUB_REPOSITORY/git/refs"
releases='[]'
# Reading the rows into a variable first makes a jq failure stop the step.
rows="$(jq -r '. as $o | (($o.paths_released // "[\".\"]") | fromjson)[] as $path
  | (if $path == "." then "" else $path + "--" end) as $p
  | [$path, $o[$p + "tag_name"] // "", $o[$p + "major"] // "", $o[$p + "sha"] // ""] | join("|")' <<<"$OUTPUTS")"
# A first pass checks every release, so a bad one moves no tag.
plan=''
while IFS='|' read -r path tag major sha; do
  if [ -z "$tag" ] || [ -z "$major" ] || [ -z "$sha" ]; then
    echo "::error::Release Please gave no tag, major or sha for path $path"
    exit 1
  fi
  base="${tag%-v*}"
  if [ "$path" = . ]; then MAJOR="v$major"
  elif [ "$base" != "$tag" ]; then MAJOR="$base-v$major"
  else
    echo "::error::Tag $tag does not fit path $path: a component tags <component>-vX.Y.Z"
    exit 1
  fi
  plan+="$MAJOR|$sha"$'\n'
  releases="$(jq -c --arg path "$path" --arg tag "$tag" --arg major_tag "$MAJOR" --arg sha "$sha" \
    '. + [{path: $path, tag: $tag, major_tag: $major_tag, sha: $sha}]' <<<"$releases")"
done <<<"$rows"
while IFS='|' read -r MAJOR sha; do
  if found="$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$MAJOR" --silent 2>&1)"; then
    gh api -X PATCH "$refs/tags/$MAJOR" -f sha="$sha" -F force=true --silent
  elif [[ "$found" == *"Not Found"* ]]; then
    gh api -X POST "$refs" -f ref="refs/tags/$MAJOR" -f sha="$sha" --silent
  else
    echo "::error::Could not read tag $MAJOR: $found"
    exit 1
  fi
done <<<"${plan%$'\n'}"
echo "releases=$releases" >> "$GITHUB_OUTPUT"
