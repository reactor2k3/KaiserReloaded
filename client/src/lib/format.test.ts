// Tests für Zeitangaben: kurze Zugzeiten als m:ss, lange (asynchron) in Stunden und Tagen.
import { describe, expect, it } from 'vitest';
import { restzeit, zugzeit } from './format';

describe('Zeitangaben', () => {
  it('Restzeit', () => {
    expect(restzeit(75)).toBe('1:15');
    expect(restzeit(3 * 3600 + 7 * 60)).toBe('3 h 07 min');
    expect(restzeit(2 * 86400 + 5 * 3600)).toBe('2 T 5 h');
    expect(restzeit(-5)).toBe('0:00');
  });
  it('Zugzeit', () => {
    expect(zugzeit(0)).toBe('unbegrenzt');
    expect(zugzeit(90)).toBe('90 Sekunden');
    expect(zugzeit(3600)).toBe('1 Stunde');
    expect(zugzeit(43200)).toBe('12 Stunden');
    expect(zugzeit(259200)).toBe('3 Tage');
  });
});
