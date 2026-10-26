### 2026.10.25.1730

- Fix: hooks from `fitness init` run the latest fitness through `fitness-install` in `$FITNESS_INSTALL_DIR`, so no `fitness` on the PATH is needed.
- Test: every hook requirement runs with no fitness on the PATH, through the installer.
- Docs: the hooks comment and the amend note name the installer, not a bare `fitness`.
- Test: requirements 0010 and 0067 name the installer path in their Given and Then.
