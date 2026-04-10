import { describe, expect, it } from 'vitest';
import { isIntegerDraft, parseIntegerDraft } from './integerInput';

describe('integerInput helpers', () => {
  it('accepts empty and digit-only drafts', () => {
    expect(isIntegerDraft('')).toBe(true);
    expect(isIntegerDraft('0')).toBe(true);
    expect(isIntegerDraft('123')).toBe(true);
  });

  it('rejects non-digit drafts', () => {
    expect(isIntegerDraft('-1')).toBe(false);
    expect(isIntegerDraft('1.5')).toBe(false);
    expect(isIntegerDraft('abc')).toBe(false);
  });

  it('parses valid integer drafts and rejects empty input', () => {
    expect(parseIntegerDraft('42')).toBe(42);
    expect(parseIntegerDraft('0007')).toBe(7);
    expect(parseIntegerDraft('')).toBeNull();
  });
});
