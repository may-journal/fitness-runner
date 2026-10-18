---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1400

- Feat: the `requirements` check fails a Swift requirement test on XCTest, Xcode UI tests, simctl, or another UI test framework.
- Feat: a Swift requirement test fails unless it, or a file in its folder or below, imports UITestingLibrary from apple-ui-testing-library.
- Feat: an end-to-end test that imports UITestingLibrarySystem may still use XCTest and simctl.
