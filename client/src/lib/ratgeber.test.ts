// Tests für den Ratgeber: Warnungen müssen bei den Regelgrenzen auftauchen.
import { describe, expect, it } from 'vitest';
import { advise } from './ratgeber';
import type { Player, SelfView } from './types';

const self = (phase: string, over: Partial<Player> = {}): SelfView => ({
  seat: 0, phase: phase as SelfView['phase'], deadline: 0, feedMin: 9000, feedMax: 14000, freeLand: -1,
  taxes: { customs: 0, vat: 0, incomeTax: 0, justice: 0, total: 0 },
  unitCosts: { Cavalry: 0, Artillery: 0, Infantry: 0 }, mercCosts: { Cavalry: 0, Artillery: 0, Infantry: 0 },
  player: {
    name: 'Micha', country: 5, female: false, land: 10000, treasury: 15000, population: 2000, grain: 20000, title: 1,
    customs: 25, vat: 10, incomeTax: 5, justice: 2, deathYear: 1770, suspendedUntil: 0,
    markets: 0, mills: 0, palace: 0, cathedral: 0, cavalry: 0, artillery: 0, infantry: 0, militia: 0,
    soldiers: 20, mercenaries: 0, morale: 90, nobility: 4, clergy: 5, burghers: 10, trade: 0, mood: 5, moodInit: true, gendarmes: 0, training: 0, equipment: 0,
    motto: '', avatar: '', arms: { kind: 'country', media: '' },
    year: { weather: 3, grainNeed: 12000, landPrice: 2, grainPrice: 80, grainBought: 0, fed: false },
    ...over,
  },
});

describe('Ratgeber', () => {
  it('nennt zuerst, was die Phase verlangt, wenn nichts drängt', () => {
    const tips = advise(self('feed'), 1700);
    expect(tips.some((t) => t.level === 'muss')).toBe(false);
    expect(tips.map((t) => t.text).join(' ')).toContain('9.000');
  });

  it('warnt vor Landmangel und fehlendem Korn', () => {
    const tips = advise(self('market', { population: 12000, grain: 5000 }), 1700);
    expect(tips[0].level).toBe('muss');
    expect(tips.map((t) => t.text).join(' ')).toMatch(/2\.000 ha.*Amtsenthebung/);
    expect(tips.some((t) => t.text.includes('7.000 Maß'))).toBe(true);
  });

  it('warnt vor dem Bankrott', () => {
    const tips = advise(self('taxes', { treasury: -7000 }), 1700);
    expect(tips[0]).toMatchObject({ level: 'muss' });
    expect(tips[0].text).toContain('Bankrott');
  });

  it('rät zu ungenutzten Schiffen und freier Kolonialdeckung (Kaiser II)', () => {
    const v = self('build', {
      traders: 3, warships: 0,
      colonies: [{ area: 2000, traders: 1, warships: 0 }, { area: 0, traders: 0, warships: 0 }, { area: 0, traders: 0, warships: 0 }],
    });
    v.finance = { bonitaet: 0, bondPrice: [], bondDiv: [], bondFree: [], holdings: [], issuedOwn: 0, dividendPA: 0 };
    const text = advise(v, 1710).map((t) => t.text).join(' ');
    expect(text).toContain('2 Handelsschiffe liegen ungenutzt');
    expect(text).toMatch(/Afrika könnten 2\.000 km²/);
  });
});
