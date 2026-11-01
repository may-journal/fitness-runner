### 2026.10.31.1500
- Feat: the requirements check fails a Measurement's `Source:` line and names the line to delete.
- Feat: the error points to `fitness requirements --fix`, which deletes every such line at once.
- Docs: the requirements check docs say a `Source:` line fails, not that it gets a note.
- Fix: a `Source:` line anywhere in a Measurement fails, and adding one fails the pair check.
- Fix: docErrors returns its notes beside errs and tooled errors, so an unreadable doc note still shows.
