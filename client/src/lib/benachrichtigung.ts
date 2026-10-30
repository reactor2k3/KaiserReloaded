// Hinweis „Sie sind dran“ für asynchrone Spiele: Ist der Tab im Hintergrund, zeigt der Titel ein
// Zeichen und – wenn der Spieler es erlaubt hat – erscheint eine Browser-Benachrichtigung.
// Ohne Erlaubnis bleibt es beim Titel; kommt der Tab nach vorn, wird der Titel zurückgesetzt.

const TITEL = 'KaiserReloaded';
const DRAN = '● Sie sind dran – ' + TITEL;

/** Ob der Browser Benachrichtigungen kann und noch um Erlaubnis gefragt werden darf */
export const kannFragen = () => typeof Notification !== 'undefined' && Notification.permission === 'default';

/** Fragt nach der Erlaubnis (nur aus einem Klick heraus aufrufen) */
export async function erlaubnisAnfragen(): Promise<boolean> {
  if (typeof Notification === 'undefined') return false;
  return (await Notification.requestPermission()) === 'granted';
}

/** Meldet, dass der Spieler wieder am Zug ist (nur wenn der Tab im Hintergrund liegt) */
export function sieSindDran(jahr: number, phase: string) {
  if (!document.hidden) return;
  document.title = DRAN;
  if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
    // tag: höchstens eine Meldung je Spiel sichtbar, neue ersetzt die alte
    new Notification('Sie sind dran', { body: `Anno ${jahr}: ${phase}`, icon: '/icon-192.png', tag: 'kaiser-dran' });
  }
}

/** Meldet dem Gastgeber eine neue Beitrittsanfrage, wenn der Tab gerade nicht sichtbar ist. */
export function neueAnfrage(name: string) {
  if (!document.hidden) return;
  document.title = '● Beitrittsanfrage – ' + TITEL;
  if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
    new Notification('Beitrittsanfrage', { body: `${name} möchte Ihrem Spiel beitreten.`, icon: '/icon-192.png', tag: 'kaiser-anfrage' });
  }
}

// Titel zurücksetzen, sobald der Spieler den Tab wieder ansieht
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden && document.title.startsWith('● ')) document.title = TITEL;
  });
}
