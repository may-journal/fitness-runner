/** Terminal column width (stdout.columns when valid, else 80). */
export function getColumns(): number {
  const c = process.stdout.columns;
  return typeof c === 'number' && c > 0 ? c : 80;
}
