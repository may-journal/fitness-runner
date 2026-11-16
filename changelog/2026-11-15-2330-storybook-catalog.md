### 2026.11.15.2330

> The requirement tests count whichever checks fitness-checks ships, so a new check no longer breaks them.

- Test: the requirement tests build their check catalog from the installed checks, not a hand-kept list that lagged `swift-storybook-requirements`.
- Test: the catalog and its count helpers move out of the 0004 test into their own file, and `rowCells` joins the shared helpers.
- Docs: requirement 0004.5 says the job summary shows every installed check, and its test now checks each one.
