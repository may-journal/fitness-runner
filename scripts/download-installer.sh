#!/usr/bin/env bash
# Downloads the Go installer for the requested version and checks it against its release checksum, writing its path to the step output.
set -eo pipefail
printf 'FITNESS_FAILURE_REPORTED=false\n' >> "$GITHUB_ENV"
if [ "$REQUESTED_VERSION" = latest ]; then
  LATEST_URL="$(curl -fsSI --retry 3 -o /dev/null -w '%{redirect_url}' https://github.com/may-journal/fitness-runner/releases/latest)"
  case "$LATEST_URL" in
    https://github.com/may-journal/fitness-runner/releases/tag/v*|https://github.com/may-journal/fitness-runner/releases/tag/go/v*)
      REQUESTED_VERSION="${LATEST_URL#https://github.com/may-journal/fitness-runner/releases/tag/}" ;;
    *) printf '::error::Invalid Fitness latest release redirect.\n'; exit 1 ;;
  esac
elif [ -z "$REQUESTED_VERSION" ]; then
  if [[ "$ACTION_REF" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || [[ "$ACTION_REF" =~ ^go/v0\.[0-9]{8}\.(0|[1-9][0-9]*)$ ]]; then
    REQUESTED_VERSION="$ACTION_REF"
  else
    # A major ref such as v1, or a branch, installs the newest release.
    REQUESTED_VERSION="$(curl -fsSI --retry 3 -o /dev/null -w '%{redirect_url}' https://github.com/may-journal/fitness-runner/releases/latest)"
    REQUESTED_VERSION="${REQUESTED_VERSION#https://github.com/may-journal/fitness-runner/releases/tag/}"
    if [[ "$ACTION_REF" =~ ^(v[0-9]+)$ ]] && [[ "$REQUESTED_VERSION" != "${BASH_REMATCH[1]}".* ]]; then
      printf '::error::The newest Fitness release %s is outside %s.\n' "$REQUESTED_VERSION" "$ACTION_REF"
      exit 1
    fi
  fi
fi
if [ -n "$REQUESTED_VERSION" ] && [ "$REQUESTED_VERSION" != latest ]; then
  REQUESTED_VERSION="${REQUESTED_VERSION#go/}"
  REQUESTED_VERSION="${REQUESTED_VERSION#v}"
  if [[ ! "$REQUESTED_VERSION" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    printf '::error::Invalid Fitness version; use numeric major.minor.patch or latest.\n'
    exit 1
  fi
  TAG="v$REQUESTED_VERSION"
  if [[ "$REQUESTED_VERSION" =~ ^0\.[0-9]{8}\.[0-9]+$ ]]; then
    TAG="go/$TAG"
  fi
  RELEASE_URL="https://github.com/may-journal/fitness-runner/releases/download/$TAG"
fi
ASSET="fitness-install-${RELEASE_URL##*/v}-$PLATFORM"
curl -fsSL --retry 3 "$RELEASE_URL/$ASSET" -o "$ASSET"
curl -fsSL --retry 3 "$RELEASE_URL/checksums.txt" -o "$ASSET.checksums"
grep "  $ASSET$" "$ASSET.checksums" | shasum -a 256 -c -
chmod +x "$ASSET"
printf 'executable=%s/%s\n' "$PWD" "$ASSET" >> "$GITHUB_OUTPUT"
