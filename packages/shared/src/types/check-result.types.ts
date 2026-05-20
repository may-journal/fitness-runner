/** Result from a fitness check. */
export type CheckResult = {
  errors: string[];
  meta?: { filesChecked?: number };
  ok: boolean;
};
