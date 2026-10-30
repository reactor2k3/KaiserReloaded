<script lang="ts">
  // Spielhilfe: Ziel, Jahreslauf, Kennzahlen, Titel, Krieg und Tipps.
  // Abschnitte als <details>, damit man schnell zum gesuchten Thema springt.
  import Modal from './Modal.svelte';
  import { COUNTRIES } from '../lib/format';
  import { COUNTRY_FILES, loadArmsCredits, type ArmsCredit } from '../lib/heraldry';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  // Bildnachweis der Landeswappen (Pflichtangaben der freien Lizenzen), beim ersten Öffnen geladen
  let credits = $state<Record<string, ArmsCredit>>({});
  $effect(() => {
    if (open && !Object.keys(credits).length) {
      loadArmsCredits().then((c) => (credits = c));
    }
  });
</script>

<Modal bind:open title="Wie spielt man Kaiser?" wide>
  <div class="help">
    <details open>
      <summary>Ziel des Spiels</summary>
      <p>Sie herrschen über eines der neun Länder des Reichs. Durch kluge Wirtschaft, Bauten und notfalls Krieg steigen Sie vom <em>Herrn</em> über Baron, Landgraf, Markgraf, Fürst, Herzog, Kurfürst und König bis zum <strong>Kaiser</strong> auf. Wer zuerst Kaiser wird, gewinnt. Jeder Herrscher ist sterblich – ab etwa 1750 kann es jedes Jahr vorbei sein.</p>
    </details>

    <details>
      <summary>Ein Jahr im Überblick</summary>
      <ol>
        <li><strong>Ernte und Markt:</strong> Das Wetter bestimmt die Ernte. Ein Teil der Reserve verfault (Mühlen senken das). Kaufen oder verkaufen Sie Korn und Land.</li>
        <li><strong>Volk versorgen:</strong> Verteilen Sie 20–80 % der Kornreserve. Genug Korn bringt Geburten, Überfluss lockt Einwanderer, Mangel kostet Leben.</li>
        <li><strong>Steuern und Justiz:</strong> Zoll, Mehrwert- und Einkommensteuer bringen Geld. Zu hohe Steuern (zusammen über 90 %) oder harte Justiz treiben Leute aus dem Land.</li>
        <li><strong>Staatskäufe:</strong> Märkte und Mühlen bringen jedes Jahr Geld, Palast und Kathedrale Ansehen. Hier werben Sie auch Truppen an oder erklären Krieg.</li>
      </ol>
      <p>Alle spielen gleichzeitig. Sind alle fertig, werden Kriege ausgetragen und das nächste Jahr beginnt. Die Uhr oben zeigt die Zugzeit je Phase.</p>
    </details>

    <details>
      <summary>Kennzahlen</summary>
      <dl>
        <dt>Land</dt><dd>Hektar. Sie brauchen mindestens so viel Land wie Einwohner, sonst werden Sie ein Jahr enthoben. Je 1000 ha ist Platz für einen Markt und eine Mühle.</dd>
        <dt>Staatskasse</dt><dd>Taler. Schulden kosten 10 % Zins; sinkt die Kasse zu tief, folgt der Bankrott.</dd>
        <dt>Einwohner</dt><dd>Mehr Einwohner bedeuten mehr Steuern, brauchen aber mehr Korn.</dd>
        <dt>Kornreserve</dt><dd>Davon lebt Ihr Volk. „Bedarf“ ist die Menge, die alle satt macht.</dd>
        <dt>Armee</dt><dd>Soldaten (je Einheit 20 Mann). Kosten jedes Jahr Sold, Söldner deutlich mehr.</dd>
        <dt>Moral</dt><dd>Kampfkraft der Armee. Sinkt jedes Jahr, Manöver heben sie.</dd>
      </dl>
    </details>

    <details>
      <summary>Korn und Einwohner</summary>
      <ul>
        <li>Unter dem Bedarf hungert das Volk, Menschen sterben.</li>
        <li>Mit dem Bedarf sind alle satt, es gibt Geburten.</li>
        <li>Ab etwa 110 bis 150 % des Bedarfs kommen Einwanderer – je mehr Überschuss, desto mehr (höchstens 10 % Einwohner im Jahr). Hohe Steuern bremsen sie.</li>
        <li>Verteilt werden dürfen nur 20 bis 80 % der Reserve. Wer mehr geben will, kauft vorher im Markt zu.</li>
        <li>Mehr Einwohner brauchen mehr Land: unter einem Hektar je Einwohner droht die Amtsenthebung.</li>
      </ul>
    </details>

    <details>
      <summary>Wie steigt man auf?</summary>
      <p>Am Jahresende zählt eine Wertung aus Märkten, Mühlen, Palast, Kathedrale, Kasse, Land, Einwohnern, Armee und Ansehen. Reicht sie, steigen Sie um eine Stufe auf – aber nur mit positiver Kasse.</p>
      <ul>
        <li><strong>König</strong> nur mit vollständigem Palast (16 Teile, ab 13.000 ha baubar).</li>
        <li><strong>Kaiser</strong> nur mit vollständiger Kathedrale (14 Teile, ab 25.000 ha), mindestens 15 Mühlen, 25 Märkten und 100.000 Talern.</li>
        <li>Krieg erklären darf man erst ab <strong>Baron</strong>.</li>
      </ul>
    </details>

    <details>
      <summary>Krieg</summary>
      <p>In den Staatskäufen erklären Sie einem Mitspieler den Krieg; ausgetragen wird am Jahresende. Die übrigen Herrscher entscheiden, ob sie neutral bleiben, Ihren Durchmarsch erlauben oder einer Seite Truppen leihen. Ein langer Weg senkt die Moral.</p>
      <ul>
        <li><strong>Kavallerie:</strong> stärkster Angriff, große Reichweite.</li>
        <li><strong>Infanterie:</strong> solide, mittlere Reichweite.</li>
        <li><strong>Artillerie:</strong> schießt über die Linien, trifft auch Gebäude.</li>
        <li><strong>Miliz:</strong> entsteht aus Märkten, Mühlen und Prachtbauten, verteidigt nur.</li>
      </ul>
      <p>Im taktischen Modus stellen Sie Ihre Truppen selbst auf; im schnellen Modus übernimmt das der Feldherr. Jeder Vorstoß ins Feindesland gewinnt Hektar; der Verlierer büßt Kasse und Einwohner ein.</p>
    </details>

    <details>
      <summary>Tipps für den Anfang</summary>
      <ul>
        <li>Erst Märkte und Mühlen bauen – sie zahlen sich jedes Jahr aus.</li>
        <li>Etwas mehr als den Bedarf verteilen bringt Einwanderer.</li>
        <li>Land rechtzeitig dazukaufen, bevor Einwohner es übersteigen.</li>
        <li>Steuern moderat halten; die Vorschau zeigt sofort, was sie bringen.</li>
        <li>Ein paar Soldaten schrecken ab – wer gar keine hat, wird leicht Opfer.</li>
      </ul>
    </details>

    <!-- Quellen der Grafiken; Wappenbilder sind eigene Zeichnungen und brauchen keinen Fremdnachweis -->
    <details>
      <summary>Bildnachweis</summary>
      <p>Landeswappen von <a href="https://commons.wikimedia.org" target="_blank" rel="noopener">Wikimedia Commons</a>, unverändert übernommen, unter ihren eigenen Lizenzen:</p>
      <ul class="credits">
        {#each COUNTRY_FILES.slice(1) as key, i (key)}
          {@const eigene = Object.entries(credits).filter(([k]) => k.startsWith(key + '-'))}
          <li>{COUNTRIES[i + 1]}:
            {#each eigene as [k, c], j (k)}{j ? '; ' : ' '}<a href={c.seite} target="_blank" rel="noopener">{c.datei}</a> von {c.urheber},
              {#if c.lizenzUrl}<a href={c.lizenzUrl} target="_blank" rel="noopener">{c.lizenz}</a>{:else}{c.lizenz}{/if}{:else} …{/each}</li>
        {/each}
      </ul>
      <p>Reichskarte aus Daten von <a href="https://www.naturalearthdata.com" target="_blank" rel="noopener">Natural Earth</a> (gemeinfrei).</p>
    </details>
  </div>
  {#snippet actions()}
    <button class="btn btn-primary" type="button" onclick={() => (open = false)}>Verstanden</button>
  {/snippet}
</Modal>

<style>
  .help { display: flex; flex-direction: column; gap: 6px; font-size: 17px; line-height: 1.4; }
  details { border-bottom: 1px solid var(--line); padding: 4px 0 8px; }
  summary { cursor: pointer; font-family: var(--font-label); font-size: 15px; font-weight: 700; letter-spacing: .05em; min-height: 36px; display: flex; align-items: center; gap: 8px; list-style: none; }
  summary::-webkit-details-marker { display: none; }
  summary::before { content: '◆'; color: var(--gold); font-size: 10px; transition: transform .15s; }
  details[open] summary::before { color: var(--blood); transform: rotate(45deg); }
  summary:focus-visible { outline: 3px solid var(--gold); outline-offset: 2px; }
  p, ol, ul, dl { margin: 6px 0; }
  dl { display: grid; grid-template-columns: 130px 1fr; gap: 4px 12px; }
  dt { font-weight: 600; }
  dd { margin: 0; }
  .credits { font-size: 14px; padding-left: 18px; }
</style>
