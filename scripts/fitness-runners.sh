#!/bin/bash
# cspell:ignore sysadminctl runnergroup bootout
# Sets up the self-hosted Fitness runners on a Mac as a runner-only user, kept alive by launchd.
# The guide in docs/self-hosted-runners.md lists the options, and --dry-run writes only the launchd jobs.
set -eu

org=may-journal
user=fitness-runner
count=2
group=fitness
out=/Library/LaunchDaemons
dry=0
version=2.338.2

die() {
  echo "fitness-runners: $*" >&2
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) dry=1 ;;
    --org | --user | --count | --out)
      [ $# -ge 2 ] || die "$1 needs a value"
      case "$1" in
        --org) org=$2 ;;
        --user) user=$2 ;;
        --count) count=$2 ;;
        --out) out=$2 ;;
      esac
      shift
      ;;
    *) die "unknown option $1; see docs/self-hosted-runners.md" ;;
  esac
  shift
done

case "$count" in
  '' | *[!0-9]* | 0) die "--count must be a whole number of at least 1, got '$count'" ;;
esac

home=/Users/$user

# plist prints the launchd job that keeps runner $1 alive as $user.
plist() {
  cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin</string>
	</dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>com.vanderbot.fitness-runner.$1</string>
	<key>ProgramArguments</key>
	<array>
		<string>$home/runner-$1/run.sh</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardErrorPath</key>
	<string>$home/runner-$1/launchd.log</string>
	<key>StandardOutPath</key>
	<string>$home/runner-$1/launchd.log</string>
	<key>UserName</key>
	<string>$user</string>
	<key>WorkingDirectory</key>
	<string>$home/runner-$1</string>
</dict>
</plist>
EOF
}

if [ "$dry" = 1 ]; then
  mkdir -p "$out"
  n=1
  while [ "$n" -le "$count" ]; do
    plist "$n" >"$out/com.vanderbot.fitness-runner.$n.plist"
    echo "wrote $out/com.vanderbot.fitness-runner.$n.plist for runner-$n in group $group of $org"
    n=$((n + 1))
  done
  exit 0
fi

[ "$(id -u)" = 0 ] || die "run with sudo, or pass --dry-run to only write the launchd jobs"
command -v gh >/dev/null || die "gh is not installed"

# The group admits private repos only, so a fork's pull request in a public
# repo never reaches this Mac.
if ! gh api "orgs/$org/actions/runner-groups" --jq '.runner_groups[].name' | grep -qx "$group"; then
  gh api -X POST "orgs/$org/actions/runner-groups" -f name="$group" -f visibility=private >/dev/null
fi

# The runners get their own user, without the agents' gh login or keychain.
if ! id "$user" >/dev/null 2>&1; then
  sysadminctl -addUser "$user" -home "$home" -shell /bin/zsh -password - >/dev/null
fi
mkdir -p "$home"
chown "$user" "$home"

arch=arm64
[ "$(uname -m)" = arm64 ] || arch=x64
tarball=$home/actions-runner-osx-$arch-$version.tar.gz
[ -f "$tarball" ] || curl -fsSL -o "$tarball" "https://github.com/actions/runner/releases/download/v$version/actions-runner-osx-$arch-$version.tar.gz"

n=1
while [ "$n" -le "$count" ]; do
  dir=$home/runner-$n
  if [ ! -f "$dir/.runner" ]; then
    mkdir -p "$dir"
    tar -xzf "$tarball" -C "$dir"
    chown -R "$user" "$dir"
    # The registration token lives for one hour and is never written to disk.
    token=$(gh api -X POST "orgs/$org/actions/runners/registration-token" --jq .token)
    sudo -u "$user" "$dir/config.sh" --unattended --replace --url "https://github.com/$org" \
      --token "$token" --name "vanderbot-fitness-$n" --runnergroup "$group" --labels "$group" --work _work
  fi
  label=com.vanderbot.fitness-runner.$n
  plist "$n" >"$out/$label.plist"
  chown root:wheel "$out/$label.plist"
  launchctl bootout "system/$label" 2>/dev/null || true
  launchctl bootstrap system "$out/$label.plist"
  echo "started $label"
  n=$((n + 1))
done
