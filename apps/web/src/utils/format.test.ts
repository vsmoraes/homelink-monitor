import { describe, expect, it } from 'vitest';
import { duration, mbps, ms } from './format';
import { statusColor } from './status';

describe('format helpers', () => {
  it('formats missing values', () => {
    expect(mbps()).toBe('—');
    expect(ms()).toBe('—');
  });

  it('formats durations', () => {
    expect(duration('2026-01-01T00:00:00Z', '2026-01-01T00:02:00Z')).toBe('2m');
  });

  it('maps status colors', () => {
    expect(statusColor('healthy')).toBe('success');
    expect(statusColor('down')).toBe('error');
  });
});
