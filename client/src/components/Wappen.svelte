<script lang="ts">
  // Wappen eines Herrschers: Landeswappen in der gewählten Variante (freie SVG von Wikimedia Commons,
  // Bildnachweis in der Hilfe) oder hochgeladene Datei. Beides nur als <img> (Uploads säubert der Server).
  import type { ArmsSpec } from '../lib/types';
  import { countryArmsSrc } from '../lib/heraldry';

  let { arms, country, variant, size = 64, label = 'Wappen' }:
    { arms?: ArmsSpec; country: number; variant?: number; size?: number; label?: string } = $props();

  // Alte Baukasten-Wappen (kind "builder") zeigen das Landeswappen
  const src = $derived(arms?.kind === 'upload' && arms.media ? `/media/${arms.media}`
    : countryArmsSrc(country, variant ?? (arms?.kind === 'country' ? arms.variant ?? 0 : 0)));
</script>

<img {src} alt={label} width={size} height={size * 1.2} class="wappen" />

<style>
  .wappen { display: block; flex-shrink: 0; object-fit: contain; }
</style>
