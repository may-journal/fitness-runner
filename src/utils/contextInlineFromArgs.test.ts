import { describe, it, expect } from 'vitest';
import { getContextInlineFromArgs, tryEqualsArg, tryExactArg } from './contextInlineFromArgs.js';

describe('getContextInlineFromArgs', () => {
  it('returns null when args are empty', () => {
    expect(getContextInlineFromArgs([], '--message')).toBeNull();
  });

  it('returns null when no arg matches', () => {
    expect(getContextInlineFromArgs(['--other', '--write'], '--message')).toBeNull();
  });

  it('returns value and strip indices for --arg=value form', () => {
    expect(getContextInlineFromArgs(['--message=feat: x'], '--message')).toEqual({
      stripIndices: [0],
      value: 'feat: x',
    });
  });

  it('returns value and strip indices for --arg value form', () => {
    expect(getContextInlineFromArgs(['--message', 'feat: y'], '--message')).toEqual({
      stripIndices: [0, 1],
      value: 'feat: y',
    });
  });

  it('returns empty value when --arg has no value (next is empty)', () => {
    expect(getContextInlineFromArgs(['--message'], '--message')).toEqual({
      stripIndices: [0, 1],
      value: '',
    });
  });

  it('strips only the arg when next is a flag', () => {
    expect(getContextInlineFromArgs(['--message', '--write'], '--message')).toEqual({
      stripIndices: [0],
      value: '',
    });
  });
});

describe('tryExactArg', () => {
  it('returns null when args[i] is not argName', () => {
    expect(tryExactArg(['--message=foo'], '--message', 0)).toBeNull();
  });

  it('returns strip [i,i+1] and value when next is the value', () => {
    expect(tryExactArg(['--message', 'feat: a'], '--message', 0)).toEqual({
      stripIndices: [0, 1],
      value: 'feat: a',
    });
  });
});

describe('tryEqualsArg', () => {
  it('returns null when args[i] does not start with eq', () => {
    expect(tryEqualsArg(['--other'], '--message=', 0)).toBeNull();
  });

  it('returns strip [i] and value after eq', () => {
    expect(tryEqualsArg(['--message=feat: scope'], '--message=', 0)).toEqual({
      stripIndices: [0],
      value: 'feat: scope',
    });
  });
});
