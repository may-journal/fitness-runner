### 2026.11.01.2339

- Fix: prose-budget passes a comment paragraph of one inline, reference or auto link, such as a requirement test's doc link.
- Fix: prose-budget passes a comment paragraph that is only one quoted example, straight or curly, with or without a closing stop.
- Fix: such a paragraph counts as no sentence, so a two-sentence comment above a link no longer fails `comment-one-statement`.
- Fix: only a whole paragraph of one link or quote passes, so a short sentence holding a link still fails.
