/** Quote a string for safe use in shell commands (double-quote with internal " escaped). */
export function quoteForShell(s: string): string {
  return `"${s.replace(/"/g, '\\"')}"`;
}
