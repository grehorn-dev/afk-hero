const integerDraftPattern = /^\d*$/;

export function isIntegerDraft(value: string): boolean {
  return integerDraftPattern.test(value);
}

export function parseIntegerDraft(value: string): number | null {
  if (value === '' || !integerDraftPattern.test(value)) {
    return null;
  }

  const parsed = Number(value);
  if (!Number.isSafeInteger(parsed)) {
    return null;
  }

  return parsed;
}
