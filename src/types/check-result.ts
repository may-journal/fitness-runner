/** Result from a fitness check. */
export type CheckResult = {
  ok: boolean;
  errors: string[];
  meta?: { filesChecked?: number };
};
