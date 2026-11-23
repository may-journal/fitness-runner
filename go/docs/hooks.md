# hooks

`fitness init` installs the canonical may-journal git hooks, which go/internal/hooks embeds, into a consumer repo.

The hooks run the latest fitness through the installer in `$FITNESS_INSTALL_DIR` (default ~/.local/share/fitness), so no fitness on the PATH is needed.

## FS

FS holds the hook scripts under files/.

## Names

Names lists the installed hook filenames, in no particular order.

## Read

Read returns the embedded script for one hook name.
