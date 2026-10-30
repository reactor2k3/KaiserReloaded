// Tests für Texte und Zustandsverarbeitung des Clients (ohne Browser).
import { describe, expect, it } from 'vitest';
import { describeEvent, num, ruler, title } from './format';
import { app, receive } from './state.svelte';
import type { PublicPlayer } from './types';

const player = (seat: number, name: string, country: number, female = false): PublicPlayer => ({
  seat, name, country, female, title: 2, alive: true, phase: 'market', land: 10000, treasury: 15000,
  population: 2000, soldiers: 20, nobility: 4, motto: '', avatar: '', arms: { kind: 'country', media: '' }, bot: '', markets: 0, mills: 0, palace: 0, cathedral: 0,
});

describe('Texte', () => {
  it('formatiert Zahlen deutsch und Titel je Anrede', () => {
    expect(num(14200)).toBe('14.200');
    expect(title(2, true)).toBe('Baronin');
    expect(ruler({ title: 5, female: false, name: 'Micha', country: 5 })).toBe('Fürst Micha von Sachsen');
  });

  it('beschreibt Schlachten aus Sicht beider Seiten', () => {
    const ps = [player(0, 'Micha', 5), player(1, 'Augusta', 1, true)];
    const won = describeEvent({ type: 'battle', player: -1, year: 1710, data: { attacker: 0, defender: 1, result: { land: 400 } } }, ps, 0);
    expect(won).toContain('Baron Micha erobert 400 Hektar von Baronin Augusta');
    const lost = describeEvent({ type: 'battle', player: -1, year: 1710, data: { attacker: 0, defender: 1, result: { land: -120 } } }, ps, 0);
    expect(lost).toContain('Augusta schlägt Baron Micha zurück');
  });

  it('zeigt fremde Ernten nicht in der eigenen Chronik', () => {
    const ps = [player(0, 'Micha', 5), player(1, 'Augusta', 1)];
    expect(describeEvent({ type: 'harvest', player: 1, year: 1700, data: { weather: 3, rotPercent: 5 } }, ps, 0)).toBe('');
    expect(describeEvent({ type: 'harvest', player: 0, year: 1700, data: { weather: 3, rotPercent: 5 } }, ps, 0)).toContain('Gewöhnliche Ernte');
  });
});

describe('Zustand', () => {
  it('mischt öffentliche Deltas, statt die Liste zu ersetzen', () => {
    receive({ t: 'public', year: 1700, over: false, winner: -1, full: true, players: [player(0, 'Micha', 5), player(1, 'Augusta', 1)] });
    const before = app.players[1];
    receive({ t: 'public', year: 1701, over: false, winner: -1, full: false, players: [{ ...player(1, 'Augusta', 1), land: 12000 }] });
    expect(app.year).toBe(1701);
    expect(app.players[1].land).toBe(12000);
    expect(app.players[1]).toBe(before); // gleiches Objekt: nur geänderte Stellen werden neu gezeichnet
    expect(app.players[0].land).toBe(10000);
  });

  it('baut die Chronik beim Wiedereinstieg neu auf, ohne den Schlachtbericht zu öffnen', () => {
    app.seat = 0;
    receive({ t: 'history', events: [
      { type: 'newYear', player: -1, year: 1701 },
      { type: 'battle', player: -1, year: 1701, data: { attacker: 1, defender: 0, result: { land: 300 }, field: {}, units: [], steps: [] } },
    ] });
    expect(app.chronicle.length).toBe(2);
    expect(app.chronicle[0].text).toContain('erobert 300 Hektar');
    expect(app.lastBattle?.historic).toBe(true);
  });

  it('zeigt Serverfehler als Hinweis', () => {
    receive({ t: 'error', code: 'too_much', message: 'Das ist zu viel!' });
    expect(app.toasts.at(-1)?.text).toBe('Das ist zu viel!');
  });
});
