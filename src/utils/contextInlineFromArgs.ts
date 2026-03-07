/** Handles --arg value form. */
export function tryExactArg(
  args: string[],
  argName: string,
  i: number
): { stripIndices: number[]; value: string } | null {
  if (args[i] !== argName) return null;
  const next = args[i + 1] ?? '';
  const value = next.startsWith('-') ? '' : next;
  const stripIndices = value === next ? [i, i + 1] : [i];
  return { stripIndices, value };
}

/** Handles --arg=value form. */
export function tryEqualsArg(
  args: string[],
  eq: string,
  i: number
): { stripIndices: number[]; value: string } | null {
  if (!args[i].startsWith(eq)) return null;
  return { stripIndices: [i], value: args[i].slice(eq.length) };
}

/** Parses args for a contextInline arg (--arg=value or --arg value); returns value and indices to strip. */
export function getContextInlineFromArgs(
  args: string[],
  argName: string
): { stripIndices: number[]; value: string } | null {
  const eq = argName + '=';
  for (let i = 0; i < args.length; i++) {
    const hit = tryExactArg(args, argName, i) ?? tryEqualsArg(args, eq, i);
    if (hit) return hit;
  }
  return null;
}
