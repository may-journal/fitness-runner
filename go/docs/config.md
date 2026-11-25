# config

`.fitnessrc.json` is the optional file at a repo's root that tunes which checks run and how strictly they judge.

The runner loads it with go/pkg/conf. It takes the same keys as the TypeScript FitnessConfig. It is JSON because the standard library parses it, and a config file must not require a toolchain.

## FileName

FileName is the config file the runner reads at the repo root.

## SchemaURL

SchemaURL is the JSON Schema dialect the schema follows.

## Defaults

Defaults holds the value each Config key takes when the config file leaves it unset.

## Config

Config is the shape of .fitnessrc.json, the optional file at a repo's root that tunes which checks run and how strictly they judge.

Fields that choose the checks:

- `policy` selects org or external use, and both default to all checks.
  - External lets a repo pick, replace, and turn off checks.
  - Org runs every check as the shared bar.
- `checks` optionally selects checks in external mode, which org ignores.
- `localChecks` names a repo directory of the repo's own check programs, searched before the bundled ones, as ADR 0009 describes.
- `disabledChecks` names checks external policy leaves out and org policy refuses, as ADR 0009 describes.
- `requiredChecks` names checks whose internal error fails the run instead of showing info, under either policy; `--require` adds to them.
- `checkVersions` pins each check the repo uses to a release such as v1.2.3 or a major such as v1.
  - A `"*"` key pins all.
  - Unset or empty, the repo uses every check at its current major.

Fields that shape a run:

- `planApprovers` lists the GitHub logins whose comment saying approve approves a Plan for the pre-push gate, under either policy.
  - Unset, the repo's owner and its org's members may approve, as GitHub reports each comment author's association.
- `scopedPullRequests` true skips a pull request's full scan, for a huge repo. A push to main still scans every file.
- `packages` lists globs, such as `"cmd/*"`, that name a monorepo's package folders, so each check judges every package on its own.
  - A glob matching no folder of tracked files fails the run, naming the glob.
  - Unset, the repo root is the one package.
  - The runner hands every check the matched roots in `FITNESS_PACKAGES`, and `checkkit.PackageRoots` reads them.

## PackageRoots

PackageRoots returns the folders of files that the packages globs match, sorted and once each. A glob that is malformed or matches no folder fails, naming it.

## Load

Load reads root's config, and a missing file returns (nil, nil) so the caller falls back to defaults. A config that still sets a retired exclusion key returns an error naming each one.

## Key

Key is one key the config file may set, as a registered section or a check's `--describe` declares it.

## Keys

Keys lists every key the registered sections declare, in registration and declaration order, with the doc of the type that holds them.

## KeysOf

KeysOf lists the keys a struct of top-level keys declares, as Register takes it, without registering them. A check puts them in its `--describe`, so the engine learns keys it does not register itself.

## Section

Section is a registered package's keys in the config file. Loading keys takes a Section, and only Register makes one, so no code can read keys the schema does not describe.

## Register

Register declares the keys a package reads from the config file, panicking when another section already declares a key.

## Section.Load

Load reads root's config into a T, which holds the keys a caller owns, such as a language module's options. It returns (nil, nil) without a config file and fails as the package's Load does.

## Setting

Setting is one key's value in a repo.

Fields:

- `key` is the key's dotted JSON path.
- `value` is what the key holds: the file's value, else its default.
- `source` says where Value came from: file, default, or unset.
- `description` is the doc comment on the key's Go field.

## Settings

Settings lists each key in root's config file with its value and source. The registered keys come in Keys order, then the declared ones.

## EngineKey

EngineKey reports whether a registered section declares top-level key, which no check may then declare.

## Schema

Schema is a JSON Schema for the config file, with one property per key of the registered sections and the checks.
