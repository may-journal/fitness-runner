### 2026.10.18.1913

- Fix: comment-check skips a comment only when a known bot wrote it, so a pasted fitness marker no longer hides a comment.
- Fix: the login fitness posts as is always skipped, even when a caller sets `ignore-authors`.
- Feat: `comment-structure` fails a comment that carries a fitness marker outside code, naming its line.
- Docs: the workflows README and the check README describe who may post a fitness marker.
