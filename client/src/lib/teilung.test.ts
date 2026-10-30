// Tests für geteilte Länder: Streifen müssen den Hektaranteilen folgen.
import { describe, expect, it } from 'vitest';
import { mix, sampleLand, slices } from './teilung';

describe('Teilung', () => {
  // Rechteck 100 × 40: breiter als hoch, also senkrechte Schnitte entlang x
  const rect = 'M0 0L100 0L100 40L0 40Z';

  it('teilt nach Anteilen', () => {
    const s = sampleLand(rect);
    expect(s.horizontal).toBe(false);
    const [a, b] = slices(s, [3, 1]);
    expect(a.from).toBe(0);
    expect(a.to).toBeGreaterThan(72);
    expect(a.to).toBeLessThan(78);
    expect(b.to).toBe(100);
    expect(b.cx).toBeGreaterThan(80);
  });

  it('beachtet Löcher (gerade-ungerade)', () => {
    const s = sampleLand('M0 0L60 0L60 60L0 60ZM20 20L40 20L40 40L20 40Z');
    expect(s.pts.some(([x, y]) => x > 21 && x < 39 && y > 21 && y < 39)).toBe(false);
  });

  it('setzt den Sitz immer ins eigene Gebiet, auch bei U-förmigen Ländern', () => {
    // U-Form: Schwerpunkt läge in der Lücke zwischen den Schenkeln
    const s = sampleLand('M0 0L20 0L20 50L40 50L40 0L60 0L60 70L0 70Z');
    const [a] = slices(s, [1]);
    expect(a.cx > 20 && a.cx < 40 && a.cy < 50).toBe(false);
  });

  it('mischt Farben', () => {
    expect(mix('#000000', '#ffffff', 0.5)).toBe('#808080');
  });
});
