<script lang="ts">
  // Galerie für wappen.html: alle Landeswappen-Varianten (Wikimedia Commons).
  import Wappen from '../components/Wappen.svelte';
  import { COUNTRIES } from '../lib/format';
  import { loadArmsCredits, variantCount, type ArmsCredit } from '../lib/heraldry';

  let credits = $state<Record<string, ArmsCredit>>({});
  loadArmsCredits().then((c) => (credits = c));
</script>

<main class="galerie">
  {#each [1, 2, 3, 4, 5, 6, 7, 8, 9] as c}
    <h2>{COUNTRIES[c]}</h2>
    <div class="row">
      {#each Array.from({ length: variantCount(credits, c) }, (_, i) => i) as v}
        <figure><Wappen country={c} variant={v} size={110} label="{COUNTRIES[c]} {v + 1}" /><figcaption>Variante {v + 1}</figcaption></figure>
      {/each}
    </div>
  {/each}
</main>

<style>
  .galerie { height: 100dvh; overflow-y: auto; padding: 16px; }
  .row { display: flex; flex-wrap: wrap; gap: 12px; }
  figure { margin: 0; display: flex; flex-direction: column; align-items: center; gap: 4px; }
</style>
