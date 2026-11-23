# languages

A language module brings its checks and tools to a run, and go/pkg/lang orders what modules claim against what is installed or embedded.

## TestScanEnv

TestScanEnv names the file in which the runner shares one scan of the repo's requirement tests with every check that reads them.

## Module

Module is one language's entry in the registry.

| Field | Holds |
| --- | --- |
| `Name` | The module's label. |
| `Detect` | Whether the repo at root uses the module's language. |
| `Exts` | The source file endings the language owns. |
| `Manifests` | The file names that mark a project of the language. |
| `Lockfiles` | The file names its package managers write on install, each a name or a path.Match pattern for one. |
| `Wrappers` | The files its build tools' wrappers write, which a repo commits as written to pin the tool's version. |
| `BuildOutput` | When set, the tracked files that are the language's build output, which git must not track. |
| `ToolConfig` | When set, whether a source file configures a tool, such as a bundler config, rather than product code. |
| `Tools` | The programs its checks run. |
| `Checks` | The check names the module claims. |
| `Dicts` | The embedded spell dictionary files the module claims. |
| `Tests` | How it finds the language's requirement tests. |
| `TestFile` | When set, whether the language's test runner finds a file by its name or folder. |
| `Technology` | The name the language's runtime products give their technology. |
| `TechnologyWords` | The lowercase words an architecture container's technology string may use to name the language. |
| `Runtime` | The runtimes a repo ships, read from the language's manifests and sources. |
| `Views` | The code declarations, relations and component references the architecture views read from its sources. |
| `Inventory` | The source endings the architecture views list as the language's sources. |

## Testing

Testing returns the modules that find requirement tests, in registry order.

## ModuleTest

ModuleTest is a requirement test and the name of the module that found it.

Fields:

- `Test`: Test is the test as its module found it.
- `Module` names the module that found the test.

## Scanned

Scanned is one scan of every testing module's tests.

Fields:

- `Root` is the repo the scan read.
- `Modules` names the testing modules the scan asked, in registry order.
- `Tests` are the tests found, in registry order.
- `Files` are the files holding tests.
- `Written` names the modules whose language the repo uses.

## ScanTests

ScanTests reads every testing module's tests in registry order. It reads the runner's shared scan instead when that scan asked the same modules of root.

## Test

Test is one test declaration and the acceptance ID its name carries, if any.

Fields:

- `Name` is the test's name as declared.
- `ID` is the acceptance ID the name carries, or "" for none.
- `Loc` is where the test is declared, as file:line.
- `File` is the path of the file declaring the test.

## TestPrefix

TestPrefix names tests that carry IDs: the prefix, then the acceptance ID with `_` for its dot, so `Test0001_1` owns 0001.1.

## TestPrefix.ID

ID returns the acceptance ID a test name carries, or "".

## TestPrefix.Name

Name is the test name that owns id.

## Tests

Tests is how a module finds its requirement tests.

Fields:

- `Scan`, when set, reads the repo's tests and the files holding them, and reports whether the repo is written in the language.
  - A module without it finds no requirement tests yet.
- `Name` is the test name that owns an acceptance ID, such as `Test0001_1`.
- `Source`, when set, reports a source path only tests build, which the architecture views class as a test.
  - It gets the path's own case, as tracked, since a language may tell folders apart by it.
- `Numbers`, when set, lists the requirement numbers a test file's name carries, or none for a file not named for requirements.
- `Tools`, when set, judges how the repo's tests are written and run, given the files Scan found tests in.
  - It returns one error per test file or run that uses a tool the language's requirement tests may not use.
- `Reads` reports a tracked file Scan or Tools may read, so a test-judging copy of the repo holds only those.

## BuildOutput

BuildOutput returns the tracked build output files every module finds in the repo at root, in registry order.

## Catalog

Catalog orders the default checks: unclaimed, unignored installed checks sorted, then each module's claimed checks in registry order.

## Dicts

Dicts orders the dictionary files: the embedded files no module claims, sorted, then each module's claimed files, sorted, in registry order.

## Exts

Exts returns every module's source endings, in registry order.

## Ignored

Ignored returns the names in the ignores folder, sorted.

## IgnoredWhy

IgnoredWhy is the reason the ignores folder gives for keeping check out of the default catalog, or "" when it names none.

## Inventory

Inventory returns every module's inventory endings, in registry order.

## IsLockfile

IsLockfile returns a test of whether a file's base name is a package manager's lockfile, by any module's patterns or the lockfiles folder.

## IsTestFile

IsTestFile reports whether any module's test runner finds file by name.

## IsTestSource

IsTestSource reports whether a module classes the source path as built only for tests.

## IsToolConfig

IsToolConfig reports whether the module owning file's ending says it configures a tool.

## IsToolWritten

IsToolWritten returns a test of whether a file is a lockfile or a module's build wrapper file, which a repo commits as written.

## Lockfiles

Lockfiles returns every module's lockfile names, in registry order, then those of the ecosystems no module covers.

## Runtimes

Runtimes returns every module's runtime reader, in registry order.

## ShareTestScan

ShareTestScan scans root afresh and writes the scan to a temporary file for the run's checks. Its env always replaces an inherited scan, and a scan that fails or outlasts wait shares none.

## Sources

Sources maps each role to the endings of the languages no module covers that play it, in file name order.

## TechnologyWords

TechnologyWords maps each module's technology name to its words.

## TestFileNumbers

TestFileNumbers lists the requirement numbers a test file's name carries, from the first module that reads any.

## TestReads

TestReads reports whether any module's Tests.Reads keeps file.

## Views

Views returns every module's architecture view reader, in registry order, leaving out modules that read none.
