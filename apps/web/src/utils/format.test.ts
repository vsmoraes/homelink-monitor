import { describe, expect, it } from 'vitest';
import { bytes, duration, mbps, ms } from './format';
import { statusColor } from './status';

describe('format helpers', () => {
  it('formats missing values', () => {
    expect(mbps()).toBe('—');
    expect(ms()).toBe('—');
  });

  it('formats durations', () => {
    expect(duration('2026-01-01T00:00:00Z', '2026-01-01T00:02:00Z')).toBe('2m');
  });

  it('formats byte values', () => {
    expect(bytes()).toBe('—');
    expect(bytes(512)).toBe('512 B');
    expect(bytes(1536)).toBe('1.5 KB');
    expect(bytes(2 * 1024 * 1024)).toBe('2.0 MB');
  });

  it('maps status colors', () => {
    expect(statusColor('healthy')).toBe('success');
    expect(statusColor('down')).toBe('error');
  });
});
