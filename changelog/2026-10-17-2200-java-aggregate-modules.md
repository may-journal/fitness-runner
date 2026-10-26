### 2026.10.17.2200

- Fix: Java coverage judges a module's own JaCoCo report when no aggregate report covers that module.
- Fix: an aggregate report covers a module when it lists any of the module's product sources.
- Fix: a module an aggregate leaves out, such as the aggregating module, is held to the coverage goal.
- Docs: requirement 0081.4 states that the aggregate and each module report it does not cover are judged.
